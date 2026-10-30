package tui

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/wahoo"
)

type ridesMsg struct {
	rides []wahoo.File
	err   error
}

type rideCopiedMsg struct {
	name string
	err  error
}

// Rides lists the rides on the device and copies new ones to the data folder.
// The device is never changed.
type Rides struct {
	serial  string
	dir     string
	rides   []wahoo.File // newest first
	synced  map[string]bool
	err     error
	status  string
	cursor  int
	queue   []string // rides left to copy
	copied  int
	loading bool
	syncing bool // auto sync is running; copying waits so no ride is copied twice at once
}

// NewRides returns the Rides screen for serial.
func NewRides(serial, dataDir string, syncing bool) Rides {
	return Rides{serial: serial, dir: ridesDir(dataDir), loading: true, syncing: syncing}
}

func (r Rides) Init() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), adbTimeout)
		defer cancel()
		rides, err := wahoo.ListRides(ctx, r.serial)
		return ridesMsg{rides, err}
	}
}

func (r Rides) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ridesMsg:
		r.loading, r.err, r.rides = false, msg.err, msg.rides
		// Names start with the start time, so name order is time order.
		slices.SortFunc(r.rides, func(a, b wahoo.File) int { return cmp.Compare(b.Name, a.Name) })
		r.synced = syncedRides(r.dir, r.rides)
	case syncStartedMsg:
		r.syncing = r.syncing || !msg.routes
	case syncDoneMsg:
		if msg.routes {
			return r, nil
		}
		r.syncing, r.status = false, ""
		return r, r.Init()
	case rideCopiedMsg:
		if msg.err != nil {
			r.queue = nil
			r.status = errorStyle.Render("Copy failed: " + msg.err.Error())
			return r, nil
		}
		r.synced[msg.name] = true
		r.copied++
		r.queue = r.queue[1:]
		if len(r.queue) == 0 {
			r.status = selectedStyle.Render(fmt.Sprintf("Copied %d rides to %s", r.copied, r.dir))
			return r, nil
		}
		return r, r.copyNext()
	case tea.KeyMsg:
		if len(r.queue) > 0 {
			return r, nil
		}
		switch msg.String() {
		case "esc":
			return r, back
		case "up", "k", "down", "j":
			r.cursor = moveCursor(r.cursor, len(r.rides), msg.String())
		case "s":
			if r.syncing {
				r.status = mutedStyle.Render("Auto sync is copying rides, wait for it to finish")
				return r, nil
			}
			r.status = ""
			r.queue = newRides(r.rides, r.synced)
			if len(r.queue) == 0 {
				r.status = mutedStyle.Render("All rides already copied")
				return r, nil
			}
			r.copied = 0
			return r, r.copyNext()
		}
	}
	return r, nil
}

func (r Rides) copyNext() tea.Cmd {
	name := r.queue[0]
	return func() tea.Msg {
		return rideCopiedMsg{name, copyRide(r.serial, name, r.dir)}
	}
}

// syncedRides reports which rides already exist locally with the same size.
func syncedRides(dir string, rides []wahoo.File) map[string]bool {
	synced := map[string]bool{}
	for _, ride := range rides {
		if info, err := os.Stat(filepath.Join(dir, ride.Name)); err == nil && info.Size() == ride.Size {
			synced[ride.Name] = true
		}
	}
	return synced
}

func (r Rides) View() string {
	switch {
	case r.err != nil:
		return frame(crumbs("Rides"), errorStyle.Render(r.err.Error()), "esc back")
	case r.loading:
		return frame(crumbs("Rides"), "Reading rides…", "esc back")
	case len(r.rides) == 0:
		return frame(crumbs("Rides"), "No rides on the device.", "esc back")
	}

	newCount := len(r.rides) - len(r.synced)
	lines := []string{mutedStyle.Render(fmt.Sprintf("%d rides · %d new", len(r.rides), newCount)), ""}
	start, end := window(r.cursor, len(r.rides), listRows)
	for i := start; i < end; i++ {
		ride := r.rides[i]
		state := selectedStyle.Render("new")
		if r.synced[ride.Name] {
			state = mutedStyle.Render("copied")
		}
		lines = append(lines, listLine(fmt.Sprintf("%-16s %8s  ", rideTime(ride.Name), humanSize(ride.Size))+state, i == r.cursor))
	}
	switch {
	case len(r.queue) > 0:
		lines = append(lines, "", fmt.Sprintf("Copying %d/%d…", r.copied+1, r.copied+len(r.queue)))
	case r.status != "":
		lines = append(lines, "", r.status)
	}
	return frame(crumbs("Rides"), strings.Join(lines, "\n"), "↑/↓ move · s copy new rides · esc back")
}

// rideTime formats the start time at the front of a ride's file name.
func rideTime(name string) string {
	if len(name) >= 17 {
		if t, err := time.Parse("2006-01-02-150405", name[:17]); err == nil {
			return t.Format("2006-01-02 15:04")
		}
	}
	return name
}
