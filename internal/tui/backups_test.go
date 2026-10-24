package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func writeBackup(t *testing.T, dir, stamp string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "BCfgManager-Bolt-"+stamp+".xml")
	if err := os.WriteFile(path, []byte("<map />"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestListBackups(t *testing.T) {
	data := t.TempDir()
	writeBackup(t, manualBackupDir(data), "20261020-091500")
	writeBackup(t, manualBackupDir(data), "20261025-140200")
	writeBackup(t, sessionBackupDir(data), "20261024-234000")
	os.WriteFile(filepath.Join(manualBackupDir(data), "notes.txt"), nil, 0o644)

	got, err := listBackups(data)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		label   string
		session bool
	}{
		{"2026-10-25 14:02:00", false},
		{"2026-10-20 09:15:00", false},
		{"2026-10-24 23:40:00", true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d entries: %+v", len(got), got)
	}
	for i, w := range want {
		if got[i].label != w.label || got[i].session != w.session {
			t.Errorf("entry %d: got %+v, want %+v", i, got[i], w)
		}
	}
}

func TestListBackupsEmpty(t *testing.T) {
	got, err := listBackups(t.TempDir())
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func pressBackups(b Backups, keys ...string) Backups {
	for _, k := range keys {
		m, _ := b.Update(key(k))
		b = m.(Backups)
	}
	return b
}

func TestBackupsDelete(t *testing.T) {
	data := t.TempDir()
	path := writeBackup(t, manualBackupDir(data), "20261020-091500")
	b := NewBackups("serial", data, filepath.Join(sessionBackupDir(data), "none.xml"))

	b = pressBackups(b, "down", "d", "esc")
	if _, err := os.Stat(path); err != nil {
		t.Fatal("esc still deleted the backup")
	}
	b = pressBackups(b, "d", "y")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("backup not deleted")
	}
	if len(b.entries) != 0 || b.cursor != 0 {
		t.Fatalf("after delete: %d entries, cursor %d", len(b.entries), b.cursor)
	}
}

func TestBackupsRestoreNeedsSessionBackup(t *testing.T) {
	data := t.TempDir()
	writeBackup(t, manualBackupDir(data), "20261020-091500")
	b := pressBackups(NewBackups("serial", data, ""), "down", "enter")
	if b.confirm != "" || b.status == "" {
		t.Fatal("restore offered without a session backup")
	}
}

func TestRestoreRejectsBrokenFile(t *testing.T) {
	path := writeBackup(t, t.TempDir(), "20261020-091500") // no DISPLAY_CFG
	msg := restore("serial", path)().(appliedMsg)
	if msg.err == nil {
		t.Fatal("broken backup accepted")
	}
}

func TestSessionBackupProtected(t *testing.T) {
	data := t.TempDir()
	current := writeBackup(t, sessionBackupDir(data), "20261025-100000")
	b := pressBackups(NewBackups("serial", data, current), "down", "d")
	if b.confirm != "" || b.status == "" {
		t.Fatal("delete offered for this session's backup")
	}
	if _, err := os.Stat(current); err != nil {
		t.Fatal("session backup gone")
	}
	if b = pressBackups(b, "enter"); b.confirm != "restore" {
		t.Fatal("restore of this session's backup should be allowed")
	}
}
