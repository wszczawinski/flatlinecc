package tui

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/wahoo"
)

// route is a .gpx file on either side; a size of -1 means it's missing there.
type route struct {
	name         string
	local, onDev int64
	changed      bool // same size on both sides, different content
}

// push reports whether sync sends the route to the device. Local wins when
// both differ: routes are edited on the computer.
func (r route) push() bool { return r.local >= 0 && (r.local != r.onDev || r.changed) }

// pull reports whether sync copies the route from the device.
func (r route) pull() bool { return r.local < 0 }

func (r route) state() string {
	switch {
	case r.pull():
		return selectedStyle.Render("to computer")
	case r.push():
		return selectedStyle.Render("to device")
	}
	return mutedStyle.Render("synced")
}

type routesMsg struct {
	routes []route
	err    error
}

type routeSyncedMsg struct{ err error }

// Routes syncs .gpx files both ways between the data folder and the device.
// Nothing is deleted on either side.
type Routes struct {
	serial  string
	dir     string
	routes  []route // by name
	err     error
	status  string
	cursor  int
	queue   []route // routes left to sync
	synced  int
	loading bool
	syncing bool // auto sync is running; syncing waits so no route is copied twice at once
}

// NewRoutes returns the Routes screen for serial.
func NewRoutes(serial, dataDir string, syncing bool) Routes {
	return Routes{serial: serial, dir: routesDir(dataDir), loading: true, syncing: syncing}
}

func (r Routes) Init() tea.Cmd {
	return func() tea.Msg {
		routes, err := readRoutes(r.serial, r.dir)
		return routesMsg{routes, err}
	}
}

func (r Routes) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case routesMsg:
		r.loading, r.err, r.routes = false, msg.err, msg.routes
		r.cursor = min(r.cursor, max(len(r.routes)-1, 0))
	case syncStartedMsg:
		r.syncing = r.syncing || msg.routes
	case syncDoneMsg:
		if !msg.routes {
			return r, nil
		}
		r.syncing, r.status = false, ""
		return r, r.Init()
	case routeSyncedMsg:
		if msg.err != nil {
			r.queue = nil
			r.status = errorStyle.Render("Sync failed: " + msg.err.Error())
			return r, r.Init()
		}
		r.synced++
		r.queue = r.queue[1:]
		if len(r.queue) == 0 {
			r.status = selectedStyle.Render(fmt.Sprintf("Synced %d routes", r.synced))
			return r, r.Init()
		}
		return r, r.syncNext()
	case tea.KeyMsg:
		if len(r.queue) > 0 {
			return r, nil
		}
		switch msg.String() {
		case "esc":
			return r, back
		case "up", "k", "down", "j":
			r.cursor = moveCursor(r.cursor, len(r.routes), msg.String())
		case "s":
			if r.syncing {
				r.status = mutedStyle.Render("Auto sync is syncing routes, wait for it to finish")
				return r, nil
			}
			r.status = ""
			r.queue = pendingRoutes(r.routes)
			if len(r.queue) == 0 {
				r.status = mutedStyle.Render("Everything in sync")
				return r, nil
			}
			r.synced = 0
			return r, r.syncNext()
		}
	}
	return r, nil
}

func (r Routes) syncNext() tea.Cmd {
	rt := r.queue[0]
	return func() tea.Msg {
		return routeSyncedMsg{syncRoute(r.serial, r.dir, rt)}
	}
}

// mergeRoutes joins local and device routes by name, sorted by name.
func mergeRoutes(local, remote []wahoo.File) []route {
	byName := map[string]*route{}
	get := func(name string) *route {
		if byName[name] == nil {
			byName[name] = &route{name: name, local: -1, onDev: -1}
		}
		return byName[name]
	}
	for _, f := range local {
		get(f.Name).local = f.Size
	}
	for _, f := range remote {
		get(f.Name).onDev = f.Size
	}
	routes := make([]route, 0, len(byName))
	for _, rt := range byName {
		routes = append(routes, *rt)
	}
	slices.SortFunc(routes, func(a, b route) int { return cmp.Compare(a.name, b.name) })
	return routes
}

// localRoutes returns the .gpx files in dir. A missing dir has none.
func localRoutes(dir string) ([]wahoo.File, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var files []wahoo.File
	for _, e := range entries {
		// Dotfiles include macOS ._ metadata files and our own .part copies.
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") || !strings.EqualFold(filepath.Ext(e.Name()), ".gpx") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return nil, err
		}
		files = append(files, wahoo.File{Name: e.Name(), Size: info.Size()})
	}
	return files, nil
}

func (r Routes) View() string {
	switch {
	case r.err != nil:
		return frame(crumbs("Routes"), errorStyle.Render(r.err.Error()), "esc back")
	case r.loading:
		return frame(crumbs("Routes"), "Reading routes…", "esc back")
	case len(r.routes) == 0:
		return frame(crumbs("Routes"), "No routes. Put .gpx files in "+r.dir+".", "esc back")
	}

	pending := len(pendingRoutes(r.routes))
	lines := []string{mutedStyle.Render(fmt.Sprintf("%d routes · %d to sync · local folder %s", len(r.routes), pending, r.dir)), ""}
	start, end := window(r.cursor, len(r.routes), listRows)
	for i := start; i < end; i++ {
		rt := r.routes[i]
		size := max(rt.local, rt.onDev)
		lines = append(lines, listLine(fmt.Sprintf("%-32s %8s  ", rt.name, humanSize(size))+rt.state(), i == r.cursor))
	}
	switch {
	case len(r.queue) > 0:
		lines = append(lines, "", fmt.Sprintf("Syncing %d/%d…", r.synced+1, r.synced+len(r.queue)))
	case r.status != "":
		lines = append(lines, "", r.status)
	}
	return frame(crumbs("Routes"), strings.Join(lines, "\n"), "↑/↓ move · s sync · esc back")
}
