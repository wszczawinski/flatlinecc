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
	if _, cmd := connectedApp(Screens{applying: true}).Update(ctrlC); quits(cmd) {
		t.Fatal("ctrl+c quit during apply")
	}
	if _, cmd := connectedApp(Screens{}).Update(ctrlC); !quits(cmd) {
		t.Fatal("ctrl+c did not quit when idle")
	}

	// unplugged mid-apply: q on the connect screen must not quit either
	a := connectedApp(Screens{applying: true})
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
