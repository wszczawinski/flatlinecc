package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/config"
)

type Settings struct {
	cfg    config.Config
	input  textinput.Model
	status string
}

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
		case "enter":
			return s.save()
		}
	}
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s Settings) save() (tea.Model, tea.Cmd) {
	cfg := s.cfg
	cfg.DataDir = strings.TrimSpace(s.input.Value())
	if cfg.DataDir == "" {
		s.status = errorStyle.Render("Path can't be empty")
		return s, nil
	}
	if err := config.Save(cfg); err != nil {
		s.status = errorStyle.Render("Save failed: " + err.Error())
		return s, nil
	}
	s.cfg = cfg
	s.status = selectedStyle.Render("Saved")
	return s, func() tea.Msg { return configSavedMsg{cfg} }
}

func (s Settings) View() string {
	body := labelStyle.Render("Data folder") + "\n" + s.input.View()
	if s.status != "" {
		body += "\n\n" + s.status
	}
	return frame(crumbs("Settings"), body, "enter save · esc back")
}
