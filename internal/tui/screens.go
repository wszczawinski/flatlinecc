package tui

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/cfg"
	"github.com/wszczawinski/flatlinecc/internal/fields"
	"github.com/wszczawinski/flatlinecc/internal/prefs"
	"github.com/wszczawinski/flatlinecc/internal/wahoo"
)

// pageOrder lists page types shown first, in this order; the rest follow in file order.
var pageOrder = []uint8{0, 2, 39, 3} // workout, climb, current climb, map

type screensMsg struct {
	doc     []byte
	display *cfg.Display
	err     error
}

// Screens lists the data pages from the live config; enter opens a page's fields.
type Screens struct {
	serial  string
	doc     []byte
	display *cfg.Display
	order   []int // display.Pages indexes in list order
	err     error
	cursor  int  // selected row in order
	open    bool // showing the selected page's fields
	slot    int  // selected field on the open page
}

func NewScreens(serial string) Screens {
	return Screens{serial: serial}
}

func (s Screens) Init() tea.Cmd {
	return func() tea.Msg {
		doc, err := wahoo.PullConfig(s.serial)
		if err != nil {
			return screensMsg{err: err}
		}
		data, err := prefs.Bytes(doc, cfg.DisplayCfgID)
		if err != nil {
			return screensMsg{err: err}
		}
		d, err := cfg.ParseDisplay(data)
		return screensMsg{doc, d, err}
	}
}

func (s Screens) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case screensMsg:
		s.doc, s.display, s.err = msg.doc, msg.display, msg.err
		if s.display != nil {
			s.order = listOrder(s.display.Pages)
		}
	case tea.KeyMsg:
		if s.display == nil {
			if msg.String() == "esc" {
				return s, back
			}
			return s, nil
		}
		if s.open {
			return s.updatePage(msg)
		}
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
		}
	}
	return s, nil
}

func (s Screens) updatePage(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		s.open = false
	case "up", "k":
		if s.slot > 0 {
			s.slot--
		}
	case "down", "j":
		if s.slot < len(s.page().Fields)-1 {
			s.slot++
		}
	}
	return s, nil
}

func (s Screens) View() string {
	switch {
	case s.err != nil:
		return frame(crumbs("Screens"), errorStyle.Render(s.err.Error()), "esc back")
	case s.display == nil:
		return frame(crumbs("Screens"), "Reading pages…", "esc back")
	case s.open:
		p := s.page()
		lines := make([]string, len(p.Fields))
		for i, f := range p.Fields {
			lines[i] = listLine(fmt.Sprintf("%2d. %s", i+1, fields.Name(f)), i == s.slot)
		}
		return frame(crumbs("Screens", fields.PageName(p.Type())), strings.Join(lines, "\n"), "↑/↓ move · esc back")
	}
	lines := make([]string, len(s.order))
	for row, i := range s.order {
		p := s.display.Pages[i]
		line := fmt.Sprintf("%d. %-16s %2d fields", row+1, fields.PageName(p.Type()), len(p.Fields))
		lines[row] = listLine(line, row == s.cursor)
	}
	return frame(crumbs("Screens"), strings.Join(lines, "\n"), "↑/↓ move · enter open · esc back")
}

// page returns the selected page.
func (s Screens) page() cfg.Page {
	return s.display.Pages[s.order[s.cursor]]
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
