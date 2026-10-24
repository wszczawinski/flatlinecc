package tui

import (
	"context"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/wahoo"
)

// sessionBackupDir holds the config as it was when the session started.
// The backup is removed on a clean quit and kept otherwise.
func sessionBackupDir(dataDir string) string {
	return filepath.Join(dataDir, "backups", "session")
}

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

type backup struct {
	started, done bool
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
