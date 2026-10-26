package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/adb"
	"github.com/wszczawinski/flatlinecc/internal/config"
)

func connectedApp(screen tea.Model) App {
	a := New(config.Default())
	a.devices = []adb.Device{{Serial: "serial", State: "device"}}
	a.screen = screen
	return a
}

func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestQuitBlockedWhileApplying(t *testing.T) {
	ctrlC := tea.KeyMsg{Type: tea.KeyCtrlC}
	if _, cmd := connectedApp(Screens{applyState: applyState{applying: true}}).Update(ctrlC); quits(cmd) {
		t.Fatal("ctrl+c quit during apply")
	}
	if _, cmd := connectedApp(Screens{}).Update(ctrlC); !quits(cmd) {
		t.Fatal("ctrl+c did not quit when idle")
	}

	// unplugged mid-apply: q on the connect screen must not quit either
	a := connectedApp(Screens{applyState: applyState{applying: true}})
	a.devices = nil
	if _, cmd := a.Update(key("q")); quits(cmd) {
		t.Fatal("q quit during apply while unplugged")
	}
}

func TestSessionBackupState(t *testing.T) {
	a := connectedApp(nil)
	m, _ := a.Update(backupMsg{path: "data/backups/session/x.xml"})
	a = m.(App)
	if a.SessionBackup() != "data/backups/session/x.xml" || a.KeepBackup() {
		t.Fatalf("after backup: %q keep=%v", a.SessionBackup(), a.KeepBackup())
	}
	m, _ = a.Update(keepBackupMsg{})
	if !m.(App).KeepBackup() {
		t.Fatal("failed apply did not keep the backup")
	}

	m, _ = connectedApp(nil).Update(backupMsg{err: errors.New("pull failed")})
	if m.(App).SessionBackup() != "" {
		t.Fatal("failed backup reported as saved")
	}
}

func TestBackupReachesOpenScreen(t *testing.T) {
	a := connectedApp(NewScreens("serial", false))
	m, _ := a.Update(backupMsg{path: "x.xml"})
	if !m.(App).screen.(Screens).backedUp {
		t.Fatal("screen opened before the backup still can't apply")
	}
}

func TestBackupRetriesAfterFailure(t *testing.T) {
	a := connectedApp(nil)
	m, cmd := a.Update(devicesMsg{devices: a.devices})
	a = m.(App)
	if !a.backup.started || cmd == nil {
		t.Fatal("backup not started on connect")
	}
	m, _ = a.Update(backupMsg{err: errors.New("device not ready")})
	a = m.(App)
	if a.backup.started {
		t.Fatal("failed backup will not be retried")
	}
	m, _ = a.Update(devicesMsg{devices: a.devices})
	if !m.(App).backup.started {
		t.Fatal("next poll did not retry the backup")
	}
}

func TestMenuOpensEveryScreen(t *testing.T) {
	for i, item := range menuItems {
		m, _ := connectedApp(nil).Update(selectMsg{i})
		if m.(App).screen == nil {
			t.Errorf("%s opened no screen", item.label)
		}
	}
}

func TestMoveCursor(t *testing.T) {
	tests := []struct {
		cursor, n int
		key       string
		want      int
	}{
		{0, 3, "down", 1},
		{2, 3, "j", 2},
		{1, 3, "up", 0},
		{0, 3, "k", 0},
		{0, 0, "down", 0},
		{1, 3, "x", 1},
	}
	for _, tt := range tests {
		if got := moveCursor(tt.cursor, tt.n, tt.key); got != tt.want {
			t.Errorf("moveCursor(%d, %d, %q) = %d, want %d", tt.cursor, tt.n, tt.key, got, tt.want)
		}
	}
}

func TestBackupStopsRetrying(t *testing.T) {
	a := connectedApp(nil)
	devices := a.devices
	for range backupTries {
		m, _ := a.Update(devicesMsg{devices: devices})
		m, _ = m.Update(backupMsg{err: errors.New("read-only data folder")})
		a = m.(App)
	}
	m, _ := a.Update(devicesMsg{devices: devices})
	if m.(App).backup.started {
		t.Fatal("backup retried past the limit")
	}

	// unplug and plug back in: tries again
	m, _ = m.Update(devicesMsg{})
	m, _ = m.Update(devicesMsg{devices: devices})
	if !m.(App).backup.started {
		t.Fatal("replug did not retry the backup")
	}
}
