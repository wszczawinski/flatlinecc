package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/config"
)

const (
	rowDataDir = iota
	rowSyncRides
	rowSyncRoutes
	settingsRows
)

// Settings edits the local data folder and auto sync. The folder is saved
// with enter; a toggle is saved as soon as it changes.
type Settings struct {
	cfg    config.Config
	input  textinput.Model
	row    int
	status string
}

// NewSettings returns the settings screen for cfg.
func NewSettings(cfg config.Config) Settings {
	input := textinput.New()
	input.SetValue(cfg.DataDir)
	input.Focus()
	return Settings{cfg: cfg, input: input}
}

func (s Settings) Init() tea.Cmd {
	return textinput.Blink
}

func (s Settings) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			return s, back
		case "tab", "down":
			return s.focus((s.row + 1) % settingsRows), nil
		case "shift+tab", "up":
			return s.focus((s.row + settingsRows - 1) % settingsRows), nil
		case "enter":
			cfg := s.cfg
			switch s.row {
			case rowDataDir:
				cfg.DataDir = strings.TrimSpace(s.input.Value())
				if cfg.DataDir == "" {
					s.status = errorStyle.Render("Path can't be empty")
					return s, nil
				}
			case rowSyncRides:
				cfg.AutoSyncRides = !cfg.AutoSyncRides
			case rowSyncRoutes:
				cfg.AutoSyncRoutes = !cfg.AutoSyncRoutes
			}
			return s.save(cfg)
		}
	}
	if s.row != rowDataDir {
		return s, nil
	}
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

// focus moves to row. Only the focused row shows a ">".
func (s Settings) focus(row int) Settings {
	s.row = row
	if row == rowDataDir {
		s.input.Prompt = "> "
		s.input.Focus()
	} else {
		s.input.Prompt = "  "
		s.input.Blur()
	}
	return s
}

func (s Settings) save(cfg config.Config) (tea.Model, tea.Cmd) {
	if err := config.Save(cfg); err != nil {
		s.status = errorStyle.Render("Save failed: " + err.Error())
		return s, nil
	}
	s.cfg = cfg
	s.status = selectedStyle.Render("Saved")
	return s, func() tea.Msg { return configSavedMsg{cfg} }
}

func (s Settings) View() string {
	label := "Data folder"
	if strings.TrimSpace(s.input.Value()) != s.cfg.DataDir {
		label += " *"
	}
	lines := []string{
		labelStyle.Render(label), s.input.View(), "",
		labelStyle.Render("Auto sync on connect"),
		listLine(fmt.Sprintf("%-8s %s", "Rides", onOff(s.cfg.AutoSyncRides)), s.row == rowSyncRides),
		listLine(fmt.Sprintf("%-8s %s", "Routes", onOff(s.cfg.AutoSyncRoutes)), s.row == rowSyncRoutes),
	}
	return frame(crumbs("Settings"), withStatus(lines, s.status), "↑/↓ move · enter save / toggle · esc back")
}
