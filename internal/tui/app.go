package tui

import (
	tea "github.com/charmbracelet/bubbletea"

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

// App is the root model: the menu, or a screen opened from it.
type App struct {
	cfg    config.Config
	menu   Menu
	screen tea.Model
}

func New(cfg config.Config) App {
	return App{cfg: cfg, menu: NewMenu()}
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

func (a App) screenFor(item string) tea.Model {
	switch item {
	case "Settings":
		return NewSettings(a.cfg)
	}
	return NewPlaceholder(item)
}
