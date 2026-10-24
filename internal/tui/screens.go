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

	"github.com/wszczawinski/flatlinecc/internal/cfg"
	"github.com/wszczawinski/flatlinecc/internal/fields"
	"github.com/wszczawinski/flatlinecc/internal/prefs"
	"github.com/wszczawinski/flatlinecc/internal/wahoo"
)

// pageOrder lists page types shown first, in this order; the rest follow in file order.
var pageOrder = []uint8{0, 2, 39, 3} // workout, elevation, climb, map

type screensMsg struct {
	base    []byte
	display *cfg.Display
	err     error
}

// Screens lists the data pages from the live config; enter opens a page's fields.
type Screens struct {
	serial   string
	backedUp bool   // a session backup exists; apply is refused without one
	base     []byte // DISPLAY_CFG as read from the device
	display  *cfg.Display
	orig     [][]uint16 // fields per page as read from the device
	order    []int      // display.Pages indexes in list order
	err      error
	status   string
	cursor   int  // selected row in order
	open     bool // showing the selected page's fields
	slot     int  // selected field on the open page
	picking  bool // choosing a field for the selected slot
	adding   bool // the picked field goes in a new slot below the selected one
	picker   Picker
	confirm  bool // reviewing changes before apply
	applying bool
}

// NewScreens returns the Screens editor for serial; backedUp enables apply.
func NewScreens(serial string, backedUp bool) Screens {
	return Screens{serial: serial, backedUp: backedUp}
}

func (s Screens) Init() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), adbTimeout)
		defer cancel()
		_, data, err := pullDisplay(ctx, s.serial)
		if err != nil {
			return screensMsg{err: err}
		}
		d, err := cfg.ParseDisplay(data)
		return screensMsg{data, d, err}
	}
}

// pullDisplay reads the live config and its DISPLAY_CFG bytes.
func pullDisplay(ctx context.Context, serial string) (doc, data []byte, err error) {
	doc, err = wahoo.PullConfig(ctx, serial)
	if err != nil {
		return nil, nil, err
	}
	data, err = prefs.Bytes(doc, cfg.DisplayCfgID)
	return doc, data, err
}

// Busy reports whether an apply is writing to the device.
func (s Screens) Busy() bool {
	return s.applying
}

func (s Screens) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case screensMsg:
		s.base, s.display, s.err, s.orig = msg.base, msg.display, msg.err, nil
		if s.display != nil {
			s.order = listOrder(s.display.Pages)
			for _, p := range s.display.Pages {
				s.orig = append(s.orig, slices.Clone(p.Fields))
			}
		}
	case appliedMsg:
		s.applying = false
		var cmd tea.Cmd
		if s.status, cmd = applyResult(msg); cmd != nil {
			return s, cmd
		}
		return s, s.Init()
	case tea.KeyMsg:
		if s.display == nil {
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
		case s.picking:
			return s.updatePicker(msg)
		case s.open:
			return s.updatePage(msg)
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
			if s.cursor < len(s.order)-1 {
				s.cursor++
			}
		case "enter":
			s.open, s.slot = true, 0
		case "a":
			s = s.startApply()
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

// startApply opens the review of changes, or says why it can't.
func (s Screens) startApply() Screens {
	switch {
	case len(s.changes()) == 0:
		s.status = mutedStyle.Render("No changes")
	case !s.backedUp:
		s.status = errorStyle.Render("No session backup, apply disabled")
	default:
		s.confirm = true
	}
	return s
}

func (s Screens) updateConfirm(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc", "n":
		s.confirm = false
	case "y":
		s.confirm, s.applying = false, true
		return s, s.apply()
	}
	return s, nil
}

// apply writes the edited layout into the live config, unless the pages
// changed on the device since Screens opened.
func (s Screens) apply() tea.Cmd {
	data, err := s.display.Bytes()
	return applyEdit(s.serial, func(doc []byte) ([]byte, error) {
		if err != nil {
			return nil, err
		}
		current, err := prefs.Bytes(doc, cfg.DisplayCfgID)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(current, s.base) {
			return nil, errors.New("pages changed on the device, reopen Screens")
		}
		return prefs.SetBytes(doc, cfg.DisplayCfgID, data)
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
	t := s.display.Pages[i].Type()
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
		s = s.startApply()
	case "esc":
		s.open = false
	case "up", "k":
		if s.slot > 0 {
			s.slot--
		}
	case "down", "j":
		if s.slot < len(p.Fields)-1 {
			s.slot++
		}
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
	case s.display == nil:
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
		title := crumbs("Screens", fields.PageName(s.page().Type()), slot)
		return frame(title, s.picker.View(), "type to filter · ↑/↓ move · enter pick · esc cancel")
	case s.open:
		p := s.page()
		lines := make([]string, len(p.Fields))
		for i, f := range p.Fields {
			lines[i] = listLine(fmt.Sprintf("%2d. %s%s", i+1, fields.Name(f), s.mark(s.order[s.cursor], i)), i == s.slot)
		}
		help := "↑/↓ move · shift+↑/↓ reorder · enter change\n\nn add · d delete · a apply · esc back"
		return frame(crumbs("Screens", fields.PageName(p.Type())), s.withStatus(lines), help)
	}
	lines := make([]string, len(s.order))
	for row, i := range s.order {
		p := s.display.Pages[i]
		line := fmt.Sprintf("%d. %-16s %2d fields%s", row+1, fields.PageName(p.Type()), len(p.Fields), s.mark(i, -1))
		lines[row] = listLine(line, row == s.cursor)
	}
	return frame(crumbs("Screens"), s.withStatus(lines), "↑/↓ move · enter open · a apply · esc back")
}

// withStatus joins lines and appends the status message, if any.
func (s Screens) withStatus(lines []string) string {
	body := strings.Join(lines, "\n")
	if s.status != "" {
		body += "\n\n" + s.status
	}
	return body
}

// changes describes each page that differs from the device, in list order.
func (s Screens) changes() []string {
	names := func(ids []uint16) string {
		out := make([]string, len(ids))
		for i, id := range ids {
			out[i] = fields.Name(id)
		}
		return strings.Join(out, ", ")
	}
	var out []string
	for _, i := range s.order {
		p := s.display.Pages[i]
		if slices.Equal(p.Fields, s.orig[i]) {
			continue
		}
		out = append(out,
			labelStyle.Render(fields.PageName(p.Type())),
			mutedStyle.Render("  was  "+names(s.orig[i])),
			"  now  "+names(p.Fields))
	}
	return out
}

// page returns the selected page; edits through it change the staged layout.
func (s Screens) page() *cfg.Page {
	return &s.display.Pages[s.order[s.cursor]]
}

// mark returns " *" if the slot (or any slot when slot is -1) on page i differs from the device.
func (s Screens) mark(i, slot int) string {
	cur, orig := s.display.Pages[i].Fields, s.orig[i]
	changed := slot < 0 && !slices.Equal(cur, orig) ||
		slot >= 0 && (slot >= len(orig) || cur[slot] != orig[slot])
	if changed {
		return " *"
	}
	return ""
}

// listOrder returns page indexes with pageOrder types first.
func listOrder(pages []cfg.Page) []int {
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

func listLine(text string, selected bool) string {
	if selected {
		return selectedStyle.Render("> " + text)
	}
	return itemStyle.Render(text)
}
