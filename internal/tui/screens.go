package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/datafields"
	"github.com/wszczawinski/flatlinecc/internal/display"
	"github.com/wszczawinski/flatlinecc/internal/prefs"
	"github.com/wszczawinski/flatlinecc/internal/wahoo"
)

// pageOrder lists page types shown first, in this order; the rest follow in file order.
var pageOrder = []uint8{0, 2, 39, 3} // workout, elevation, climb, map

type screensMsg struct {
	base   []byte
	layout *display.Layout
	err    error
}

// Screens lists the data pages from the live config; enter opens a page's fields.
type Screens struct {
	applyState
	serial  string
	base    []byte // DISPLAY_CFG as read from the device
	layout  *display.Layout
	orig    [][]uint16 // fields per page as read from the device
	order   []int      // layout.Pages indexes in list order
	err     error
	cursor  int  // selected row in order
	open    bool // showing the selected page's fields
	slot    int  // selected field on the open page
	picking bool // choosing a field for the selected slot
	adding  bool // the picked field goes in a new slot below the selected one
	picker  Picker
}

// NewScreens returns the Screens editor for serial; backedUp enables apply.
func NewScreens(serial string, backedUp bool) Screens {
	return Screens{serial: serial, applyState: applyState{backedUp: backedUp}}
}

func (s Screens) Init() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), adbTimeout)
		defer cancel()
		doc, err := wahoo.PullConfig(ctx, s.serial)
		if err != nil {
			return screensMsg{err: err}
		}
		data, err := prefs.Bytes(doc, display.ConfigField)
		if err != nil {
			return screensMsg{err: err}
		}
		l, err := display.Parse(data)
		return screensMsg{data, l, err}
	}
}

func (s Screens) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, ok := s.applyState.update(msg, s.apply, s.Init); ok {
		return s, cmd
	}
	switch msg := msg.(type) {
	case screensMsg:
		s.base, s.layout, s.err, s.orig = msg.base, msg.layout, msg.err, nil
		if s.layout != nil {
			s.order = listOrder(s.layout.Pages)
			for _, p := range s.layout.Pages {
				s.orig = append(s.orig, slices.Clone(p.Fields))
			}
		}
	case tea.KeyMsg:
		if s.layout == nil {
			if msg.String() == "esc" {
				return s, back
			}
			return s, nil
		}
		switch {
		case s.picking:
			return s.updatePicker(msg)
		case s.open:
			return s.updatePage(msg)
		}
		s.status = ""
		switch msg.String() {
		case "esc":
			return s, back
		case "up", "k", "down", "j":
			s.cursor = moveCursor(s.cursor, len(s.order), msg.String())
		case "enter":
			s.open, s.slot = true, 0
		case "a":
			s.start(len(s.changes()) > 0)
		}
	default:
		if s.picking {
			var cmd tea.Cmd
			s.picker, cmd = s.picker.Update(msg)
			return s, cmd
		}
	}
	return s, nil
}

// apply writes the edited layout into the live config, unless the pages
// changed on the device since Screens opened.
func (s Screens) apply() tea.Cmd {
	data, err := s.layout.Bytes()
	return applyEdit(s.serial, func(doc []byte) ([]byte, error) {
		if err != nil {
			return nil, err
		}
		current, err := prefs.Bytes(doc, display.ConfigField)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(current, s.base) {
			return nil, errors.New("pages changed on the device, reopen Screens")
		}
		return prefs.SetBytes(doc, display.ConfigField, data)
	})
}

func (s Screens) updatePicker(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		s.picking = false
		return s, nil
	case "enter":
		if id, ok := s.picker.Selected(); ok {
			p := s.page()
			if s.adding {
				s.slot = min(s.slot+1, len(p.Fields))
				p.Fields = slices.Insert(p.Fields, s.slot, id)
			} else {
				p.Fields[s.slot] = id
			}
		}
		s.picking = false
		return s, nil
	}
	var cmd tea.Cmd
	s.picker, cmd = s.picker.Update(key)
	return s, cmd
}

// minFields is the fewest fields a page type may keep; other pages need 1.
var minFields = map[uint8]int{0: 3, 2: 3, 39: 3, 3: 2} // workout, elevation, climb, map

// maxFields is the most fields a page type may hold, from the stock layout;
// more per page is untested. Unknown types keep what the device has.
var maxFields = map[uint8]int{
	0:  10, // workout
	1:  9,  // lap
	2:  6,  // elevation
	3:  3,  // map
	4:  4,  // KICKR
	8:  7,  // segment
	15: 6,  // planned workout
	39: 5,  // climb
}

// fieldLimits returns how many fields page i may hold.
func (s Screens) fieldLimits(i int) (lo, hi int) {
	t := s.layout.Pages[i].Type()
	lo, hi = 1, len(s.orig[i])
	if n, ok := minFields[t]; ok {
		lo = n
	}
	if n, ok := maxFields[t]; ok {
		hi = n
	}
	return lo, hi
}

// openPicker opens the field picker on the selected slot. When adding, the
// pick goes in a new slot below it.
func (s Screens) openPicker(adding bool) (Screens, tea.Cmd) {
	p := s.page()
	s.picking, s.adding, s.picker = true, adding, NewPicker(p.Fields[s.slot], p.Type())
	return s, textinput.Blink
}

func (s Screens) updatePage(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	s.status = ""
	p := s.page()
	if len(p.Fields) == 0 {
		// Nothing to select; a page the device has empty can't grow either.
		if key.String() == "esc" {
			s.open = false
		}
		return s, nil
	}
	lo, hi := s.fieldLimits(s.order[s.cursor])
	switch key.String() {
	case "a":
		s.start(len(s.changes()) > 0)
	case "esc":
		s.open = false
	case "up", "k", "down", "j":
		s.slot = moveCursor(s.slot, len(p.Fields), key.String())
	case "shift+up", "K":
		if s.slot > 0 {
			p.Fields[s.slot-1], p.Fields[s.slot] = p.Fields[s.slot], p.Fields[s.slot-1]
			s.slot--
		}
	case "shift+down", "J":
		if s.slot < len(p.Fields)-1 {
			p.Fields[s.slot+1], p.Fields[s.slot] = p.Fields[s.slot], p.Fields[s.slot+1]
			s.slot++
		}
	case "enter":
		return s.openPicker(false)
	case "n":
		if len(p.Fields) >= hi {
			s.status = mutedStyle.Render(fmt.Sprintf("Max %d fields on this page", hi))
			return s, nil
		}
		return s.openPicker(true)
	case "d":
		if len(p.Fields) <= lo {
			s.status = mutedStyle.Render(fmt.Sprintf("Min %d fields on this page", lo))
			return s, nil
		}
		p.Fields = slices.Delete(p.Fields, s.slot, s.slot+1)
		s.slot = min(s.slot, len(p.Fields)-1)
	}
	return s, nil
}

func (s Screens) View() string {
	switch {
	case s.err != nil:
		return frame(crumbs("Screens"), errorStyle.Render(s.err.Error()), "esc back")
	case s.layout == nil:
		return frame(crumbs("Screens"), "Reading pages…", "esc back")
	case s.applying:
		return frame(crumbs("Screens", "Apply"), "Applying…", "")
	case s.confirm:
		body := strings.Join(s.changes(), "\n") + "\n\n" + mutedStyle.Render("The Wahoo app restarts to load the new layout.")
		return frame(crumbs("Screens", "Apply"), body, "y apply · esc cancel")
	case s.picking:
		slot := fmt.Sprintf("Field %d", s.slot+1)
		if s.adding {
			slot = "New field"
		}
		title := crumbs("Screens", datafields.PageName(s.page().Type()), slot)
		return frame(title, s.picker.View(), "type to filter · ↑/↓ move · enter pick · esc cancel")
	case s.open:
		p := s.page()
		lines := make([]string, len(p.Fields))
		for i, f := range p.Fields {
			lines[i] = listLine(fmt.Sprintf("%2d. %s%s", i+1, datafields.Name(f), s.mark(s.order[s.cursor], i)), i == s.slot)
		}
		help := "↑/↓ move · shift+↑/↓ reorder · enter change\n\nn add · d delete · a apply · esc back"
		return frame(crumbs("Screens", datafields.PageName(p.Type())), withStatus(lines, s.status), help)
	}
	lines := make([]string, len(s.order))
	for row, i := range s.order {
		p := s.layout.Pages[i]
		line := fmt.Sprintf("%d. %-16s %2d fields%s", row+1, datafields.PageName(p.Type()), len(p.Fields), s.mark(i, -1))
		lines[row] = listLine(line, row == s.cursor)
	}
	return frame(crumbs("Screens"), withStatus(lines, s.status), "↑/↓ move · enter open · a apply · esc back")
}

// changes describes each page that differs from the device, in list order.
func (s Screens) changes() []string {
	names := func(ids []uint16) string {
		out := make([]string, len(ids))
		for i, id := range ids {
			out[i] = datafields.Name(id)
		}
		return strings.Join(out, ", ")
	}
	var out []string
	for _, i := range s.order {
		p := s.layout.Pages[i]
		if slices.Equal(p.Fields, s.orig[i]) {
			continue
		}
		out = append(out,
			labelStyle.Render(datafields.PageName(p.Type())),
			mutedStyle.Render("  was  "+names(s.orig[i])),
			"  now  "+names(p.Fields))
	}
	return out
}

// page returns the selected page; edits through it change the staged layout.
func (s Screens) page() *display.Page {
	return &s.layout.Pages[s.order[s.cursor]]
}

// mark returns " *" if the slot (or any slot when slot is -1) on page i differs from the device.
func (s Screens) mark(i, slot int) string {
	cur, orig := s.layout.Pages[i].Fields, s.orig[i]
	changed := slot < 0 && !slices.Equal(cur, orig) ||
		slot >= 0 && (slot >= len(orig) || cur[slot] != orig[slot])
	if changed {
		return " *"
	}
	return ""
}

// listOrder returns page indexes with pageOrder types first.
func listOrder(pages []display.Page) []int {
	rank := func(i int) int {
		if r := slices.Index(pageOrder, pages[i].Type()); r >= 0 {
			return r
		}
		return len(pageOrder)
	}
	order := make([]int, len(pages))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return rank(a) - rank(b) })
	return order
}
