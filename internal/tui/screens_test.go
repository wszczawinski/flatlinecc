package tui

import (
	"slices"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/cfg"
)

func page(typ uint8, fields ...uint16) cfg.Page {
	head := make([]byte, 11)
	head[5] = typ
	return cfg.Page{Head: head, Fields: fields}
}

// loaded returns Screens as if the device sent these pages.
func loaded(pages ...cfg.Page) Screens {
	m, _ := NewScreens("serial", true).Update(screensMsg{display: &cfg.Display{Pages: pages}})
	return m.(Screens)
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func press(s Screens, keys ...string) Screens {
	for _, k := range keys {
		m, _ := s.Update(key(k))
		s = m.(Screens)
	}
	return s
}

func TestListOrder(t *testing.T) {
	pages := []cfg.Page{page(1), page(3), page(0), page(8), page(39), page(2)}
	// workout, elevation, climb, map first, then the rest in file order
	if got, want := listOrder(pages), []int{2, 5, 4, 1, 0, 3}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestPickStagesChange(t *testing.T) {
	s := loaded(page(0, 158, 10))
	if len(s.changes()) != 0 || s.mark(0, -1) != "" {
		t.Fatal("fresh load shows changes")
	}

	// open workout, go to slot 2, open picker, filter to Time of Day (30), pick
	s = press(s, "enter", "down", "enter")
	for _, r := range "time of day" {
		s = press(s, string(r))
	}
	s = press(s, "enter")

	if got := s.display.Pages[0].Fields; !slices.Equal(got, []uint16{158, 30}) {
		t.Fatalf("fields %v", got)
	}
	if s.mark(0, 1) != " *" || s.mark(0, 0) != "" || s.mark(0, -1) != " *" {
		t.Fatal("marks wrong")
	}
	if got := s.changes(); len(got) != 1 {
		t.Fatalf("changes %v", got)
	}
	if !slices.Equal(s.orig[0], []uint16{158, 10}) {
		t.Fatalf("original changed: %v", s.orig[0])
	}
}

func TestPickerEscKeepsField(t *testing.T) {
	s := press(loaded(page(0, 158)), "enter", "enter", "x", "esc")
	if s.picking || s.display.Pages[0].Fields[0] != 158 {
		t.Fatal("esc changed the field")
	}
}

func TestApplyNeedsChangesAndBackup(t *testing.T) {
	s := press(loaded(page(0, 158)), "a")
	if s.confirm {
		t.Fatal("confirm opened without changes")
	}

	s = loaded(page(0, 158))
	s.backedUp = false
	s.display.Pages[0].Fields[0] = 10
	if s = press(s, "a"); s.confirm {
		t.Fatal("confirm opened without a session backup")
	}

	s.backedUp = true
	if s = press(s, "a"); !s.confirm {
		t.Fatal("confirm did not open")
	}
	if s = press(s, "esc"); s.confirm || s.Busy() {
		t.Fatal("esc did not cancel")
	}
}

func TestPickerFilter(t *testing.T) {
	p := NewPicker(10, 0)
	if id, _ := p.Selected(); id != 10 {
		t.Fatalf("cursor on %d, want current field 10", id)
	}
	for _, r := range "425" {
		p, _ = p.Update(key(string(r)))
	}
	if !slices.Equal(p.matches, []uint16{425}) {
		t.Fatalf("filter by ID: %v", p.matches)
	}
	for _, r := range "zzz" {
		p, _ = p.Update(key(string(r)))
	}
	if _, ok := p.Selected(); ok {
		t.Fatal("selected with no matches")
	}
}
