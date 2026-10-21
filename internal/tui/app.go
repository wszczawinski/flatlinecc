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
	width   int
	height  int
}

func New(cfg config.Config) App {
	return App{cfg: cfg}
}

func (a App) Init() tea.Cmd {
	return checkDevices
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, nil
	case devicesMsg:
		a.devices, a.adbErr, a.checked = msg.devices, msg.err, true
		return a, poll()
	case pollMsg:
		return a, checkDevices
	}

	if a.connected() == nil {
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "q" {
			return a, tea.Quit
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
		statusStyle.Render(statusLine(device)),
		body,
	)
}

func (a App) screenFor(item string) tea.Model {
	switch item {
	case "Settings":
		return NewSettings(a.cfg)
	case "Device info":
		return NewDeviceInfo(a.connected().Serial)
	}
	return NewPlaceholder(item)
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
