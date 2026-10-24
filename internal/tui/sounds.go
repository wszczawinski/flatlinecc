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

// soundFields are the sound toggles, as cfg field IDs (see docs/format.md).
// Phone, message and email alerts are left out: they need Bluetooth.
var soundFields = []struct {
	id    int
	label string
}{
	{18, "Mute all sounds"},
	{8, "Workout buzzer"},
	{9, "Notification buzzer"},
	{10, "Navigation buzzer"},
	{45, "Planned workout buzzer"},
	{37, "Mute segments"},
	{11, "Mario tune"},
}

type soundsMsg struct {
	values []bool
	err    error
}

// Sounds edits the sound toggles in the live config.
type Sounds struct {
	serial   string
	backedUp bool   // a session backup exists; apply is refused without one
	orig     []bool // values as read from the device
	values   []bool
	err      error
	status   string
	cursor   int
	confirm  bool
	applying bool
}

// NewSounds returns the Sounds screen for serial; backedUp enables apply.
func NewSounds(serial string, backedUp bool) Sounds {
	return Sounds{serial: serial, backedUp: backedUp}
}

func (s Sounds) Init() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), adbTimeout)
		defer cancel()
		doc, err := wahoo.PullConfig(ctx, s.serial)
		if err != nil {
			return soundsMsg{err: err}
		}
		values := make([]bool, len(soundFields))
		for i, f := range soundFields {
			if values[i], err = prefs.Bool(doc, f.id); err != nil {
				return soundsMsg{err: err}
			}
		}
		return soundsMsg{values: values}
	}
}

// Busy reports whether an apply is writing to the device.
func (s Sounds) Busy() bool {
	return s.applying
}

func (s Sounds) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case soundsMsg:
		s.err, s.orig, s.values = msg.err, msg.values, slices.Clone(msg.values)
	case appliedMsg:
		s.applying = false
		var cmd tea.Cmd
		if s.status, cmd = applyResult(msg); cmd != nil {
			return s, cmd
		}
		return s, s.Init()
	case tea.KeyMsg:
		if s.values == nil {
			if msg.String() == "esc" {
				return s, back
			}
			return s, nil
		}
		switch {
		case s.applying:
			return s, nil
		case s.confirm:
			return s.updateConfirm(msg)
		}
		s.status = ""
		switch msg.String() {
		case "esc":
			return s, back
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(s.values)-1 {
				s.cursor++
			}
		case "enter", " ":
			s.values[s.cursor] = !s.values[s.cursor]
		case "a":
			switch {
			case slices.Equal(s.values, s.orig):
				s.status = mutedStyle.Render("No changes")
			case !s.backedUp:
				s.status = errorStyle.Render("No session backup, apply disabled")
			default:
				s.confirm = true
			}
		}
	}
	return s, nil
}

func (s Sounds) updateConfirm(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc", "n":
		s.confirm = false
	case "y":
		s.confirm, s.applying = false, true
		return s, s.apply()
	}
	return s, nil
}

// apply writes the changed toggles into the live config.
func (s Sounds) apply() tea.Cmd {
	values := slices.Clone(s.values)
	return applyEdit(s.serial, func(doc []byte) ([]byte, error) {
		var err error
		for i, f := range soundFields {
			if doc, err = prefs.SetBool(doc, f.id, values[i]); err != nil {
				return nil, err
			}
		}
		return doc, nil
	})
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
	lines := make([]string, len(soundFields))
	for i, f := range soundFields {
		mark := ""
		if s.values[i] != s.orig[i] {
			mark = " *"
		}
		lines[i] = listLine(fmt.Sprintf("%-24s %-3s%s", f.label, onOff(s.values[i]), mark), i == s.cursor)
	}
	body := strings.Join(lines, "\n")
	if s.status != "" {
		body += "\n\n" + s.status
	}
	return frame(crumbs("Sounds"), body, "↑/↓ move · enter toggle · a apply · esc back")
}

// changes describes each toggle that differs from the device.
func (s Sounds) changes() []string {
	var out []string
	for i, f := range soundFields {
		if s.values[i] != s.orig[i] {
			out = append(out, fmt.Sprintf("%s  %s → %s", f.label, onOff(s.orig[i]), onOff(s.values[i])))
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
