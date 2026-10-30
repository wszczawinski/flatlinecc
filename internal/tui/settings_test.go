package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/config"
)

func TestSettingsToggleSaves(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var m tea.Model = NewSettings(config.Default())
	m, _ = m.Update(key("down"))
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || !cmd().(configSavedMsg).cfg.AutoSyncRides {
		t.Fatal("rides toggle not saved")
	}
	saved, err := config.Load()
	if err != nil || !saved.AutoSyncRides || saved.AutoSyncRoutes {
		t.Fatalf("saved %+v, %v", saved, err)
	}
	if s := m.(Settings); s.input.Focused() || s.input.Prompt != "  " {
		t.Fatal("input still looks selected on the toggle row")
	}
}
