package tui

import (
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
		switch {
		case d == nil:
			a.backup.tries = 0 // plugging back in tries again
		case !a.backup.started && a.backup.tries < backupTries:
			a.backup.started = true
			a.backup.tries++
			return a, tea.Batch(poll(), backupConfig(d.Serial, a.cfg.DataDir))
		}
		return a, poll()
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
		return a, nil
	}

	if a.screen != nil {
		return a.updateScreen(msg)
	}
	var cmd tea.Cmd
	a.menu, cmd = a.menu.Update(msg)
	return a, cmd
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
		statusStyle.Render(statusLine(device)+"  "+a.backup.status()),
		body,
	)
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
