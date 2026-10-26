package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/wahoo"
)

// backupTries is how many times the session backup is tried per connect.
const backupTries = 3

type backupMsg struct {
	path string
	err  error
}

// backupConfig saves the live config once per session, on first connect.
func backupConfig(serial, dataDir string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), adbTimeout)
		defer cancel()
		path, err := wahoo.BackupConfig(ctx, serial, sessionBackupDir(dataDir))
		return backupMsg{path, err}
	}
}

// backup is the state of the session backup.
type backup struct {
	started, done bool
	tries         int  // attempts since the device was plugged in
	keep          bool // an apply failed; keep the backup on quit
	path          string
	err           error
}

func (b backup) status() string {
	switch {
	case !b.done:
		return mutedStyle.Render("backing up…")
	case b.err != nil:
		return errorStyle.Render("backup failed: " + b.err.Error())
	}
	return mutedStyle.Render("backup saved")
}
