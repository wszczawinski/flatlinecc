package tui

import tea "github.com/charmbracelet/bubbletea"

// openMsg asks the app to show a screen.
type openMsg struct{ screen tea.Model }

// backMsg asks the app to return to the menu.
type backMsg struct{}

func open(screen tea.Model) tea.Cmd {
	return func() tea.Msg { return openMsg{screen} }
}

func back() tea.Msg { return backMsg{} }

// App is the root model: the menu, or a screen opened from it.
type App struct {
	menu   Menu
	screen tea.Model
}

func New() App {
	return App{menu: NewMenu()}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
	case openMsg:
		a.screen = msg.screen
		return a, a.screen.Init()
	case backMsg:
		a.screen = nil
		return a, nil
	}

	if a.screen != nil {
		var cmd tea.Cmd
		a.screen, cmd = a.screen.Update(msg)
		return a, cmd
	}
	m, cmd := a.menu.Update(msg)
	a.menu = m.(Menu)
	return a, cmd
}

func (a App) View() string {
	if a.screen != nil {
		return a.screen.View()
	}
	return a.menu.View()
}
