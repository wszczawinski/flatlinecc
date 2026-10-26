package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// menuItems are the main menu entries and the screens they open. serial is
// the connected device.
var menuItems = []struct {
	label string
	open  func(a App, serial string) tea.Model
}{
	{"Screens", func(a App, serial string) tea.Model {
		return NewScreens(serial, a.SessionBackup() != "")
	}},
	{"Sounds", func(a App, serial string) tea.Model {
		return NewSounds(serial, a.SessionBackup() != "")
	}},
	{"Rides", func(a App, serial string) tea.Model {
		return NewRides(serial, a.cfg.DataDir)
	}},
	{"Routes", func(a App, serial string) tea.Model {
		return NewRoutes(serial, a.cfg.DataDir)
	}},
	{"Backup / restore", func(a App, serial string) tea.Model {
		return NewBackups(serial, a.cfg.DataDir, a.SessionBackup())
	}},
	{"Device info", func(a App, serial string) tea.Model {
		return NewDeviceInfo(serial)
	}},
	{"Settings", func(a App, serial string) tea.Model {
		return NewSettings(a.cfg)
	}},
}

// Menu is the main menu.
type Menu struct {
	cursor int
}

func (m Menu) Update(msg tea.Msg) (Menu, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "q":
		return m, tea.Quit
	case "up", "k", "down", "j":
		m.cursor = moveCursor(m.cursor, len(menuItems), key.String())
	case "enter":
		return m, selectItem(m.cursor)
	}
	return m, nil
}

func (m Menu) View() string {
	lines := make([]string, len(menuItems))
	for i, item := range menuItems {
		lines[i] = listLine(item.label, i == m.cursor)
	}
	return frame(crumbs(), strings.Join(lines, "\n"), "↑/↓ move · enter select · q quit")
}
