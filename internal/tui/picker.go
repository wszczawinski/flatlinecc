package tui

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/datafields"
)

const pickerRows = 10

// Picker is a filterable list of fields.
type Picker struct {
	input   textinput.Model
	all     []uint16 // sorted by name
	matches []uint16
	cursor  int
}

// NewPicker lists the fields offered on pageType.
func NewPicker(current uint16, pageType uint8) Picker {
	input := textinput.New()
	input.Placeholder = "type to filter"
	input.Focus()

	all := datafields.ForPage(pageType)
	if !slices.Contains(all, current) {
		// Keep the slot's field pickable even when it's hidden for this page.
		all = append(all, current)
	}
	slices.SortFunc(all, func(a, b uint16) int {
		return cmp.Compare(datafields.Name(a), datafields.Name(b))
	})
	p := Picker{input: input, all: all}
	p.filter()
	p.cursor = max(slices.Index(p.matches, current), 0)
	return p
}

// Selected returns the highlighted field, false if nothing matches.
func (p Picker) Selected() (uint16, bool) {
	if len(p.matches) == 0 {
		return 0, false
	}
	return p.matches[p.cursor], true
}

func (p Picker) Update(msg tea.Msg) (Picker, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up":
			if p.cursor > 0 {
				p.cursor--
			}
			return p, nil
		case "down":
			if p.cursor < len(p.matches)-1 {
				p.cursor++
			}
			return p, nil
		}
	}
	var cmd tea.Cmd
	query := p.input.Value()
	p.input, cmd = p.input.Update(msg)
	if p.input.Value() != query {
		p.filter()
		p.cursor = 0
	}
	return p, cmd
}

// filter keeps fields whose name or ID contains the query.
func (p *Picker) filter() {
	q := strings.ToLower(strings.TrimSpace(p.input.Value()))
	p.matches = nil // fresh slice: older Picker values must keep theirs
	for _, id := range p.all {
		if strings.Contains(strings.ToLower(datafields.Name(id)), q) || strings.Contains(strconv.Itoa(int(id)), q) {
			p.matches = append(p.matches, id)
		}
	}
}

func (p Picker) View() string {
	lines := []string{p.input.View(), ""}
	if len(p.matches) == 0 {
		lines = append(lines, mutedStyle.Render("No matching fields"))
		return strings.Join(lines, "\n")
	}
	start, end := window(p.cursor, len(p.matches), pickerRows)
	for i := start; i < end; i++ {
		lines = append(lines, listLine(datafields.Name(p.matches[i]), i == p.cursor))
	}
	lines = append(lines, "", mutedStyle.Render(fmt.Sprintf("%d/%d", p.cursor+1, len(p.matches))))
	return strings.Join(lines, "\n")
}
