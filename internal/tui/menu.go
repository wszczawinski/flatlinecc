package tui

import (
	"fmt"
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
	status string
}

func NewMenu() Menu {
	return Menu{}
}

func (m Menu) Init() tea.Cmd {
	return nil
}

func (m Menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "q", "ctrl+c":
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
		m.status = fmt.Sprintf("%s: not implemented yet", menuItems[m.cursor])
	}
	return m, nil
}

func (m Menu) View() string {
	var b strings.Builder
	b.WriteString("flatlinecc\n\n")
	for i, item := range menuItems {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		b.WriteString(cursor + item + "\n")
	}
	if m.status != "" {
		b.WriteString("\n" + m.status + "\n")
	}
	b.WriteString("\n↑/↓ move · enter select · q quit\n")
	return b.String()
}
