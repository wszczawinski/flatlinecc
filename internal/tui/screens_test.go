package tui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/cfg"
	"github.com/wszczawinski/flatlinecc/internal/prefs"
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
	if got := s.changes(); len(got) == 0 {
		t.Fatal("no changes listed")
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

func TestPickerKeepsHiddenCurrentField(t *testing.T) {
	// 11 (Lap Distance) is offered only on the lap page.
	p := NewPicker(11, 0)
	if id, _ := p.Selected(); id != 11 {
		t.Fatalf("cursor on %d, want hidden current field 11", id)
	}
}

func TestEmptyPageEnter(t *testing.T) {
	s := press(loaded(page(0)), "enter", "enter")
	if s.picking {
		t.Fatal("picker opened on a page with no fields")
	}
}

func TestAddDeleteLimits(t *testing.T) {
	// workout: min 3, max 10
	s := loaded(page(0, 158, 10, 30, 32, 2, 5, 40, 425, 45, 41))

	s = press(s, "enter", "n")
	if s.picking || s.status == "" {
		t.Fatal("add allowed past 10 fields")
	}

	s = press(s, "d", "d", "d", "d", "d", "d", "d")
	if got := s.page().Fields; !slices.Equal(got, []uint16{425, 45, 41}) {
		t.Fatalf("after deletes: %v", got)
	}
	if s = press(s, "d"); len(s.page().Fields) != 3 || s.status == "" {
		t.Fatal("delete allowed below the workout minimum")
	}

	// add below slot 1: picker starts on 425, pick it as-is
	s = press(s, "n", "enter")
	if got := s.page().Fields; !slices.Equal(got, []uint16{425, 425, 45, 41}) || s.slot != 1 {
		t.Fatalf("after add: %v, slot %d", got, s.slot)
	}
	if s.mark(0, 0) != " *" {
		t.Fatal("changed slot not marked")
	}
}

func TestMaxIsFixedAfterApply(t *testing.T) {
	// the device already has 9 workout fields after an earlier apply
	s := press(loaded(page(0, 158, 10, 30, 32, 2, 5, 40, 425, 45)), "enter", "n")
	if !s.picking {
		t.Fatal("could not add back to 10")
	}
}

func TestAddEscCancels(t *testing.T) {
	s := loaded(page(0, 158, 10, 30, 32))
	s = press(s, "enter", "d", "n", "esc")
	if len(s.page().Fields) != 3 {
		t.Fatalf("esc still added: %v", s.page().Fields)
	}
}

func TestReorder(t *testing.T) {
	s := press(loaded(page(3, 158, 21)), "enter", "J")
	if got := s.page().Fields; !slices.Equal(got, []uint16{21, 158}) || s.slot != 1 {
		t.Fatalf("after move down: %v, slot %d", got, s.slot)
	}
	s = press(s, "K")
	if got := s.page().Fields; !slices.Equal(got, []uint16{158, 21}) || s.slot != 0 {
		t.Fatalf("after move up: %v, slot %d", got, s.slot)
	}
	if len(s.changes()) != 0 {
		t.Fatal("moving back still shows a change")
	}
}

func TestOtherPagesMinOne(t *testing.T) {
	s := press(loaded(page(8, 238, 239)), "enter", "d")
	if len(s.page().Fields) != 1 {
		t.Fatal("segment page should go down to 1 field")
	}
	if s = press(s, "d"); len(s.page().Fields) != 1 {
		t.Fatal("deleted the last field")
	}
}

func TestMapLimits(t *testing.T) {
	// map: min 2, max 3 even though the device had 2
	s := press(loaded(page(3, 158, 21)), "enter", "n", "enter")
	if len(s.page().Fields) != 3 {
		t.Fatalf("map should take a 3rd field: %v", s.page().Fields)
	}
	if s = press(s, "n"); s.picking {
		t.Fatal("map took a 4th field")
	}
	s = press(s, "d", "d")
	if len(s.page().Fields) != 2 {
		t.Fatalf("map went below 2: %v", s.page().Fields)
	}
}

func loadedSounds(values ...bool) Sounds {
	m, _ := NewSounds("serial", true).Update(soundsMsg{values: values})
	return m.(Sounds)
}

func pressSounds(s Sounds, keys ...string) Sounds {
	for _, k := range keys {
		m, _ := s.Update(key(k))
		s = m.(Sounds)
	}
	return s
}

func TestSoundsToggleAndApplyGuard(t *testing.T) {
	values := make([]bool, len(soundFields))
	s := pressSounds(loadedSounds(values...), "a")
	if s.confirm {
		t.Fatal("confirm opened without changes")
	}
	s = pressSounds(s, "down", "enter")
	if !s.values[1] || s.orig[1] || len(s.changes()) != 1 {
		t.Fatalf("toggle: values %v, changes %v", s.values, s.changes())
	}
	if s = pressSounds(s, "a"); !s.confirm {
		t.Fatal("confirm did not open")
	}
	if s = pressSounds(s, "esc", "enter"); len(s.changes()) != 0 {
		t.Fatal("toggling back still shows a change")
	}
}

func TestSoundsEditOnlyChanged(t *testing.T) {
	values := make([]bool, len(soundFields))
	s := loadedSounds(values...)
	s.values[1] = true // workout buzzer (8)

	// on the device, mute (18) was turned on after the screen opened
	var doc strings.Builder
	doc.WriteString("<map>\n")
	for _, f := range soundFields {
		fmt.Fprintf(&doc, `<boolean name="{&quot;type&quot;:%d,&quot;boltId&quot;:&quot;&quot;}" value="%t" />`+"\n", f.id, f.id == 18)
	}
	out, err := s.edit([]byte(doc.String()))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := prefs.Bool(out, 8); !v {
		t.Fatal("changed toggle not written")
	}
	if v, _ := prefs.Bool(out, 18); !v {
		t.Fatal("device-side mute was overwritten")
	}
}
