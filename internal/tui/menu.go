package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var menuItems = []string{
	"Screens",
	"Sounds",
	"Rides",
	"Routes",
	"Backup / restore",
	"Device info",
	"Settings",
}

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
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(menuItems)-1 {
			m.cursor++
		}
	case "enter":
		return m, selectItem(menuItems[m.cursor])
	}
	return m, nil
}

func (m Menu) View() string {
	lines := make([]string, len(menuItems))
	for i, item := range menuItems {
		lines[i] = listLine(item, i == m.cursor)
	}
	return frame(crumbs(), strings.Join(lines, "\n"), "↑/↓ move · enter select · q quit")
}
