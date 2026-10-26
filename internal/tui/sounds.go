package tui

import (
	"context"
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/prefs"
	"github.com/wszczawinski/flatlinecc/internal/wahoo"
)

type soundsMsg struct {
	values []bool
	err    error
}

// Sounds edits the sound toggles in the live config.
type Sounds struct {
	applyState
	serial string
	orig   []bool // values as read from the device
	values []bool
	err    error
	cursor int
}

// NewSounds returns the Sounds screen for serial; backedUp enables apply.
func NewSounds(serial string, backedUp bool) Sounds {
	return Sounds{serial: serial, applyState: applyState{backedUp: backedUp}}
}

func (s Sounds) Init() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), adbTimeout)
		defer cancel()
		doc, err := wahoo.PullConfig(ctx, s.serial)
		if err != nil {
			return soundsMsg{err: err}
		}
		values := make([]bool, len(wahoo.Sounds))
		for i, f := range wahoo.Sounds {
			if values[i], err = prefs.Bool(doc, f.ID); err != nil {
				return soundsMsg{err: err}
			}
		}
		return soundsMsg{values: values}
	}
}

func (s Sounds) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, ok := s.applyState.update(msg, s.apply, s.Init); ok {
		return s, cmd
	}
	switch msg := msg.(type) {
	case soundsMsg:
		s.err, s.orig, s.values = msg.err, msg.values, slices.Clone(msg.values)
	case tea.KeyMsg:
		if s.values == nil {
			if msg.String() == "esc" {
				return s, back
			}
			return s, nil
		}
		s.status = ""
		switch msg.String() {
		case "esc":
			return s, back
		case "up", "k", "down", "j":
			s.cursor = moveCursor(s.cursor, len(s.values), msg.String())
		case "enter", " ":
			s.values[s.cursor] = !s.values[s.cursor]
		case "a":
			s.start(!slices.Equal(s.values, s.orig))
		}
	}
	return s, nil
}

// apply writes the changed toggles into the live config. Unchanged ones are
// left alone, so settings changed on the device meanwhile are kept.
func (s Sounds) apply() tea.Cmd {
	s.values = slices.Clone(s.values) // later toggles must not leak into this apply
	return applyEdit(s.serial, s.edit)
}

func (s Sounds) edit(doc []byte) ([]byte, error) {
	var err error
	for i, f := range wahoo.Sounds {
		if s.values[i] == s.orig[i] {
			continue
		}
		if doc, err = prefs.SetBool(doc, f.ID, s.values[i]); err != nil {
			return nil, err
		}
	}
	return doc, nil
}

func (s Sounds) View() string {
	switch {
	case s.err != nil:
		return frame(crumbs("Sounds"), errorStyle.Render(s.err.Error()), "esc back")
	case s.values == nil:
		return frame(crumbs("Sounds"), "Reading settings…", "esc back")
	case s.applying:
		return frame(crumbs("Sounds", "Apply"), "Applying…", "")
	case s.confirm:
		body := strings.Join(s.changes(), "\n") + "\n\n" + mutedStyle.Render("The Wahoo app restarts to load the new settings.")
		return frame(crumbs("Sounds", "Apply"), body, "y apply · esc cancel")
	}
	lines := make([]string, len(wahoo.Sounds))
	for i, f := range wahoo.Sounds {
		mark := ""
		if s.values[i] != s.orig[i] {
			mark = " *"
		}
		lines[i] = listLine(fmt.Sprintf("%-24s %-3s%s", f.Name, onOff(s.values[i]), mark), i == s.cursor)
	}
	return frame(crumbs("Sounds"), withStatus(lines, s.status), "↑/↓ move · enter toggle · a apply · esc back")
}

// changes describes each toggle that differs from the device.
func (s Sounds) changes() []string {
	var out []string
	for i, f := range wahoo.Sounds {
		if s.values[i] != s.orig[i] {
			out = append(out, fmt.Sprintf("%s  %s → %s", f.Name, onOff(s.orig[i]), onOff(s.values[i])))
		}
	}
	return out
}

func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}
