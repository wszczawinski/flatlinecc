package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/wahoo"
)

type appliedMsg struct{ err error }

// keepBackupMsg tells the app to keep the session backup on quit.
type keepBackupMsg struct{}

// applyEdit pulls a fresh copy of the live config, so anything the device
// saved since the screen opened is kept, runs edit on it and applies the result.
func applyEdit(serial string, edit func(doc []byte) ([]byte, error)) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), applyTimeout)
		defer cancel()
		doc, err := wahoo.PullConfig(ctx, serial)
		if err != nil {
			return appliedMsg{err}
		}
		if doc, err = edit(doc); err != nil {
			return appliedMsg{err}
		}
		return appliedMsg{wahoo.ApplyConfig(ctx, serial, doc)}
	}
}

// applyResult handles an appliedMsg: on failure the session backup is kept.
func applyResult(msg appliedMsg) (status string, cmd tea.Cmd) {
	if msg.err != nil {
		return errorStyle.Render("Apply failed: " + msg.err.Error()), func() tea.Msg { return keepBackupMsg{} }
	}
	return selectedStyle.Render("Applied"), nil
}
