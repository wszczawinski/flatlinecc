package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/wszczawinski/flatlinecc/internal/adb"
	"github.com/wszczawinski/flatlinecc/internal/config"
)

// selectMsg asks the app to open the screen for a menu item.
type selectMsg struct{ item string }

// backMsg asks the app to return to the menu.
type backMsg struct{}

// configSavedMsg tells the app the config changed.
type configSavedMsg struct{ cfg config.Config }

func selectItem(item string) tea.Cmd {
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
		if d := a.connected(); d != nil && !a.backup.started {
			a.backup.started = true
			return a, tea.Batch(poll(), backupConfig(d.Serial, a.cfg.DataDir))
		}
		return a, poll()
	case backupMsg:
		a.backup.path, a.backup.err, a.backup.done = msg.path, msg.err, true
		// Retry on the next poll: right after plugging in, the device may not
		// be ready yet.
		a.backup.started = msg.err == nil
		if a.screen != nil {
			// A screen opened before the backup finished can now apply.
			var cmd tea.Cmd
			a.screen, cmd = a.screen.Update(msg)
			return a, cmd
		}
		return a, nil
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
		case !ok && a.screen != nil:
			// Deliver results of work started before the device went away.
			var cmd tea.Cmd
			a.screen, cmd = a.screen.Update(msg)
			return a, cmd
		}
		return a, nil
	}

	switch msg := msg.(type) {
	case selectMsg:
		a.screen = a.screenFor(msg.item)
		return a, a.screen.Init()
	case backMsg:
		a.screen = nil
		return a, nil
	case configSavedMsg:
		a.cfg = msg.cfg
		return a, nil
	}

	if a.screen != nil {
		var cmd tea.Cmd
		a.screen, cmd = a.screen.Update(msg)
		return a, cmd
	}
	var cmd tea.Cmd
	a.menu, cmd = a.menu.Update(msg)
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

func (a App) screenFor(item string) tea.Model {
	switch item {
	case "Screens":
		return NewScreens(a.connected().Serial, a.SessionBackup() != "")
	case "Sounds":
		return NewSounds(a.connected().Serial, a.SessionBackup() != "")
	case "Rides":
		return NewRides(a.connected().Serial, a.cfg.DataDir)
	case "Routes":
		return NewRoutes(a.connected().Serial, a.cfg.DataDir)
	case "Backup / restore":
		return NewBackups(a.connected().Serial, a.cfg.DataDir, a.SessionBackup())
	case "Settings":
		return NewSettings(a.cfg)
	case "Device info":
		return NewDeviceInfo(a.connected().Serial)
	}
	panic("no screen for menu item " + item)
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
