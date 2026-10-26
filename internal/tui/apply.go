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

// applyState is the review-then-apply flow of screens that edit the live
// config.
type applyState struct {
	backedUp bool // a session backup exists; apply is refused without one
	confirm  bool // reviewing changes before apply
	applying bool
	status   string
}

// Busy reports whether an apply is writing to the device.
func (a applyState) Busy() bool {
	return a.applying
}

// start opens the review of changes, or says why it can't.
func (a *applyState) start(changed bool) {
	switch {
	case !changed:
		a.status = mutedStyle.Render("No changes")
	case !a.backedUp:
		a.status = errorStyle.Render("No session backup, apply disabled")
	default:
		a.confirm = true
	}
}

// update handles the flow's messages and keys: apply runs on "y", reload
// after a successful apply. It returns false for anything else.
func (a *applyState) update(msg tea.Msg, apply, reload func() tea.Cmd) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case backupMsg:
		a.backedUp = msg.err == nil
		return nil, true
	case appliedMsg:
		a.applying = false
		var cmd tea.Cmd
		if a.status, cmd = applyResult(msg); cmd != nil {
			return cmd, true
		}
		return reload(), true
	case tea.KeyMsg:
		if a.applying {
			return nil, true
		}
		if !a.confirm {
			return nil, false
		}
		switch msg.String() {
		case "esc", "n":
			a.confirm = false
		case "y":
			a.confirm, a.applying = false, true
			return apply(), true
		}
		return nil, true
	}
	return nil, false
}
