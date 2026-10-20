package tui

import tea "github.com/charmbracelet/bubbletea"

// Placeholder stands in for screens that aren't built yet.
type Placeholder struct {
	title string
}

func NewPlaceholder(title string) Placeholder {
	return Placeholder{title: title}
}

func (p Placeholder) Init() tea.Cmd {
	return nil
}

func (p Placeholder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		return p, back
	}
	return p, nil
}

func (p Placeholder) View() string {
	return frame(p.title, "Not implemented yet.", "esc back")
}
