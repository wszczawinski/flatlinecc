package tui

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/cfg"
	"github.com/wszczawinski/flatlinecc/internal/prefs"
	"github.com/wszczawinski/flatlinecc/internal/wahoo"
)

// manualBackupDir holds backups made from the Backup / restore screen. They are never removed automatically.
func manualBackupDir(dataDir string) string {
	return filepath.Join(dataDir, "backups", "manual")
}

type backupEntry struct {
	path    string
	label   string // backup time, from the file name
	session bool
}

type manualBackupMsg struct{ err error }

// Backups lists config backups: back up now, restore or delete one.
type Backups struct {
	serial  string
	dataDir string
	session string // this session's backup; restore is refused without one
	entries []backupEntry
	err     error
	status  string
	cursor  int // 0 is "Back up now", then entries
	confirm string
	working bool // backing up or restoring
}

// NewBackups returns the Backup / restore screen. session is this session's
// backup: it enables restore and can't be deleted here.
func NewBackups(serial, dataDir, session string) Backups {
	b := Backups{serial: serial, dataDir: dataDir, session: session}
	b.entries, b.err = listBackups(dataDir)
	return b
}

func (b Backups) Init() tea.Cmd {
	return nil
}

// Busy reports whether a restore is writing to the device.
func (b Backups) Busy() bool {
	return b.working
}

func (b Backups) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case backupMsg:
		if msg.err == nil {
			b.session = msg.path
			b.entries, b.err = listBackups(b.dataDir)
		}
	case manualBackupMsg:
		b.working = false
		b.status = selectedStyle.Render("Backup saved")
		if msg.err != nil {
			b.status = errorStyle.Render("Backup failed: " + msg.err.Error())
		}
		b.entries, b.err = listBackups(b.dataDir)
	case appliedMsg:
		b.working = false
		var cmd tea.Cmd
		b.status, cmd = applyResult(msg)
		if msg.err == nil {
			b.status = selectedStyle.Render("Restored")
		}
		return b, cmd
	case tea.KeyMsg:
		switch {
		case b.working:
			return b, nil
		case b.confirm != "":
			return b.updateConfirm(msg)
		}
		b.status = ""
		switch msg.String() {
		case "esc":
			return b, back
		case "up", "k":
			if b.cursor > 0 {
				b.cursor--
			}
		case "down", "j":
			if b.cursor < len(b.entries) {
				b.cursor++
			}
		case "enter":
			if b.cursor == 0 {
				b.working = true
				return b, backupNow(b.serial, b.dataDir)
			}
			if b.session == "" {
				b.status = errorStyle.Render("No session backup, restore disabled")
				return b, nil
			}
			b.confirm = "restore"
		case "d":
			switch {
			case b.cursor == 0:
			case b.entries[b.cursor-1].path == b.session:
				b.status = mutedStyle.Render("This session's backup can't be deleted")
			default:
				b.confirm = "delete"
			}
		}
	}
	return b, nil
}

func (b Backups) updateConfirm(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc", "n":
		b.confirm = ""
	case "y":
		e := b.entries[b.cursor-1]
		action := b.confirm
		b.confirm = ""
		if action == "restore" {
			b.working = true
			return b, restore(b.serial, e.path)
		}
		if err := os.Remove(e.path); err != nil {
			b.status = errorStyle.Render("Delete failed: " + err.Error())
			return b, nil
		}
		b.status = mutedStyle.Render("Deleted " + e.label)
		b.entries, b.err = listBackups(b.dataDir)
		b.cursor = min(b.cursor, len(b.entries))
	}
	return b, nil
}

func backupNow(serial, dataDir string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), adbTimeout)
		defer cancel()
		_, err := wahoo.BackupConfig(ctx, serial, manualBackupDir(dataDir))
		return manualBackupMsg{err}
	}
}

// restore pushes a backup in place of the live config, after checking that
// its page layout parses.
func restore(serial, path string) tea.Cmd {
	return func() tea.Msg {
		doc, err := os.ReadFile(path)
		if err != nil {
			return appliedMsg{err}
		}
		data, err := prefs.Bytes(doc, cfg.DisplayCfgID)
		if err == nil {
			_, err = cfg.ParseDisplay(data)
		}
		if err != nil {
			return appliedMsg{fmt.Errorf("not a usable backup: %w", err)}
		}
		ctx, cancel := context.WithTimeout(context.Background(), applyTimeout)
		defer cancel()
		return appliedMsg{wahoo.ApplyConfig(ctx, serial, doc)}
	}
}

// listBackups returns manual backups, then session ones, newest first.
func listBackups(dataDir string) ([]backupEntry, error) {
	var all []backupEntry
	for _, dir := range []struct {
		path    string
		session bool
	}{{manualBackupDir(dataDir), false}, {sessionBackupDir(dataDir), true}} {
		files, err := os.ReadDir(dir.path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		var entries []backupEntry
		for _, f := range files {
			stamp, ok := strings.CutPrefix(f.Name(), "BCfgManager-Bolt-")
			stamp, ok2 := strings.CutSuffix(stamp, ".xml")
			if !ok || !ok2 || f.IsDir() {
				continue
			}
			label := stamp
			if t, err := time.ParseInLocation("20060102-150405", stamp, time.Local); err == nil {
				label = t.Format("2006-01-02 15:04:05")
			}
			entries = append(entries, backupEntry{filepath.Join(dir.path, f.Name()), label, dir.session})
		}
		slices.Reverse(entries) // ReadDir sorts by name, which is oldest first
		all = append(all, entries...)
	}
	return all, nil
}

func (b Backups) View() string {
	help := "↑/↓ move · enter back up / restore · d delete · esc back"
	switch {
	case b.err != nil:
		return frame(crumbs("Backup / restore"), errorStyle.Render(b.err.Error()), "esc back")
	case b.working && b.cursor == 0:
		return frame(crumbs("Backup / restore"), "Backing up…", "")
	case b.working:
		return frame(crumbs("Backup / restore", "Restore"), "Restoring…", "")
	case b.confirm == "restore":
		body := "Restore the backup from " + b.entries[b.cursor-1].label + "?\n\n" +
			mutedStyle.Render("It replaces the whole config on the device. The Wahoo app restarts.")
		return frame(crumbs("Backup / restore", "Restore"), body, "y restore · esc cancel")
	case b.confirm == "delete":
		body := "Delete the backup from " + b.entries[b.cursor-1].label + "?"
		return frame(crumbs("Backup / restore", "Delete"), body, "y delete · esc cancel")
	}

	lines := []string{listLine("Back up now", b.cursor == 0)}
	section := ""
	for i, e := range b.entries {
		if s := sectionName(e.session); s != section {
			section = s
			lines = append(lines, "", mutedStyle.Render(s))
		}
		label := e.label
		if e.path == b.session {
			label += mutedStyle.Render(" (this session)")
		}
		lines = append(lines, listLine(label, b.cursor == i+1))
	}
	if len(b.entries) == 0 {
		lines = append(lines, "", mutedStyle.Render("No backups yet"))
	}
	if b.status != "" {
		lines = append(lines, "", b.status)
	}
	return frame(crumbs("Backup / restore"), strings.Join(lines, "\n"), help)
}

func sectionName(session bool) string {
	if session {
		return "Session (removed on a clean quit, kept after a crash or failed apply)"
	}
	return "Manual"
}
