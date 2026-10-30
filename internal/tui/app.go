package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wszczawinski/flatlinecc/internal/adb"
	"github.com/wszczawinski/flatlinecc/internal/config"
)

// selectMsg asks the app to open the screen for menuItems[item].
type selectMsg struct{ item int }

// backMsg asks the app to return to the menu.
type backMsg struct{}

// configSavedMsg tells the app the config changed.
type configSavedMsg struct{ cfg config.Config }

func selectItem(item int) tea.Cmd {
	return func() tea.Msg { return selectMsg{item} }
}

func back() tea.Msg { return backMsg{} }

// App is the root model: the connect screen until a device is ready,
// then the menu or a screen opened from it.
type App struct {
	cfg     config.Config
	menu    Menu
	screen  tea.Model
	devices []adb.Device
	adbErr  error
	checked bool
	backup  backup
	rides   syncState // auto sync, per connect
	routes  syncState
	width   int
	height  int
}

// New returns the root model for tea.NewProgram.
func New(cfg config.Config) App {
	return App{cfg: cfg}
}

func (a App) Init() tea.Cmd {
	return checkDevices
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if a.busy() {
			// Quitting mid-apply could leave a half-written config on the device.
			break
		}
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, nil
	case devicesMsg:
		a.devices, a.adbErr, a.checked = msg.devices, msg.err, true
		d := a.connected()
		if d == nil {
			a.backup.tries = 0                           // plugging back in tries again
			a.rides, a.routes = syncState{}, syncState{} // and syncs again
			return a, poll()
		}
		cmds := []tea.Cmd{poll()}
		if !a.backup.started && a.backup.tries < backupTries {
			a.backup.started = true
			a.backup.tries++
			cmds = append(cmds, backupConfig(d.Serial, a.cfg.DataDir))
		}
		cmds = append(cmds, a.startSyncs(d.Serial)...)
		return a, tea.Batch(cmds...)
	case syncStartedMsg:
		return a.updateScreen(msg)
	case syncListedMsg:
		part := a.syncPart(msg.routes)
		if !part.started || part.listed {
			return a, nil // from before an unplug
		}
		part.listed, part.steps, part.err = true, msg.steps, msg.err
		return a.syncNext(msg.routes)
	case syncStepMsg:
		part := a.syncPart(msg.routes)
		if len(part.steps) == 0 {
			return a, nil // from before an unplug
		}
		part.err = msg.err
		if msg.err == nil {
			part.n++
			part.steps = part.steps[1:]
		}
		return a.syncNext(msg.routes)
	case backupMsg:
		a.backup.path, a.backup.err, a.backup.done = msg.path, msg.err, true
		// Retry on the next poll: right after plugging in, the device may not
		// be ready yet.
		a.backup.started = msg.err == nil
		// A screen opened before the backup finished can now apply.
		return a.updateScreen(msg)
	case keepBackupMsg:
		a.backup.keep = true
		return a, nil
	case pollMsg:
		return a, checkDevices
	}

	if a.connected() == nil {
		key, ok := msg.(tea.KeyMsg)
		switch {
		case ok && key.String() == "q" && !a.busy():
			return a, tea.Quit
		case !ok:
			// Deliver results of work started before the device went away.
			return a.updateScreen(msg)
		}
		return a, nil
	}

	switch msg := msg.(type) {
	case selectMsg:
		a.screen = menuItems[msg.item].open(a, a.connected().Serial)
		return a, a.screen.Init()
	case backMsg:
		a.screen = nil
		return a, nil
	case configSavedMsg:
		a.cfg = msg.cfg
		// A toggle turned on syncs now, not on the next poll.
		syncs := a.startSyncs(a.connected().Serial)
		return a, tea.Batch(syncs...)
	}

	if a.screen != nil {
		return a.updateScreen(msg)
	}
	var cmd tea.Cmd
	a.menu, cmd = a.menu.Update(msg)
	return a, cmd
}

// startSyncs starts the auto sync parts that are turned on and haven't run
// since the device was plugged in.
func (a *App) startSyncs(serial string) []tea.Cmd {
	var cmds []tea.Cmd
	if a.cfg.AutoSyncRides && !a.rides.started {
		a.rides.started = true
		cmds = append(cmds, listSync(serial, a.cfg.DataDir, false), syncStarted(false))
	}
	if a.cfg.AutoSyncRoutes && !a.routes.started {
		a.routes.started = true
		cmds = append(cmds, listSync(serial, a.cfg.DataDir, true), syncStarted(true))
	}
	return cmds
}

func (a *App) syncPart(routes bool) *syncState {
	if routes {
		return &a.routes
	}
	return &a.rides
}

// syncNext runs the next copy, or ends the part after the last one or an error.
func (a App) syncNext(routes bool) (tea.Model, tea.Cmd) {
	part := a.syncPart(routes)
	if part.err == nil && len(part.steps) > 0 {
		return a, runStep(routes, part.steps[0])
	}
	part.done, part.steps = true, nil
	// An open Rides or Routes screen can copy again, and reloads.
	return a.updateScreen(syncDoneMsg{routes})
}

// updateScreen passes msg to the open screen, if any.
func (a App) updateScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.screen == nil {
		return a, nil
	}
	var cmd tea.Cmd
	a.screen, cmd = a.screen.Update(msg)
	return a, cmd
}

func (a App) View() string {
	device := a.connected()
	if device == nil {
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center,
			connectView(a.checked, a.devices, a.adbErr))
	}

	body := a.menu.View()
	if a.screen != nil {
		body = a.screen.View()
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		statusStyle.Render(a.statusLines(device)),
		body,
	)
}

// statusLines shows the device, then the backup and sync below it.
func (a App) statusLines(d *adb.Device) string {
	lines := []string{statusLine(d), a.backup.status()}
	for _, s := range []string{a.rides.status("rides", "new ride", "copied"), a.routes.status("routes", "route", "synced")} {
		if s != "" {
			lines = append(lines, s)
		}
	}
	return strings.Join(lines, "\n")
}

// SessionBackup returns the session backup's path, or "" if none was saved.
func (a App) SessionBackup() string {
	if a.backup.err != nil {
		return ""
	}
	return a.backup.path
}

// KeepBackup reports whether the session backup must stay after quit.
func (a App) KeepBackup() bool {
	return a.backup.keep
}

// busy reports whether the open screen is writing to the device.
func (a App) busy() bool {
	b, ok := a.screen.(interface{ Busy() bool })
	return ok && b.Busy()
}

// connected returns the first ready device, or nil.
func (a App) connected() *adb.Device {
	for i := range a.devices {
		if a.devices[i].State == "device" {
			return &a.devices[i]
		}
	}
	return nil
}
