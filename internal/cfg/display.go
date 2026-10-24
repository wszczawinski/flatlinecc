package cfg

import (
	"encoding/binary"
	"fmt"
	"math"
)

// DisplayCfgID is the DISPLAY_CFG field holding the page layout.
const DisplayCfgID = 53

const (
	displayHeaderSize = 6
	pageHeadSize      = 11
)

// Display is a decoded DISPLAY_CFG. Bytes with unknown meaning are kept as-is.
type Display struct {
	Header []byte
	Pages  []Page
}

// Page is one data page: an 11-byte head (partly decoded) and its field types.
type Page struct {
	Head   []byte
	Fields []uint16
}

// ID returns the page ID from the head.
func (p Page) ID() uint16 { return binary.LittleEndian.Uint16(p.Head[1:3]) }

// Type returns the page type (workout, map, ...), see the fields package for names.
func (p Page) Type() uint8 { return p.Head[5] }

// ParseDisplay decodes DISPLAY_CFG data. Header and Head slices point into b.
func ParseDisplay(b []byte) (*Display, error) {
	r := reader{b: b}
	d := &Display{Header: r.bytes(displayHeaderSize)}
	if r.err != nil {
		return nil, fmt.Errorf("header: %w", r.err)
	}
	for len(r.b) > 0 {
		p := Page{Head: r.bytes(pageHeadSize)}
		for range r.u8() {
			p.Fields = append(p.Fields, r.u16())
		}
		if r.err != nil {
			return nil, fmt.Errorf("page %d: %w", len(d.Pages), r.err)
		}
		d.Pages = append(d.Pages, p)
	}
	return d, nil
}

// Bytes encodes the layout. A page holds at most 255 fields.
func (d *Display) Bytes() ([]byte, error) {
	b := append([]byte{}, d.Header...)
	for i, p := range d.Pages {
		if len(p.Fields) > math.MaxUint8 {
			return nil, fmt.Errorf("page %d: %d fields, max %d", i, len(p.Fields), math.MaxUint8)
		}
		b = append(b, p.Head...)
		b = append(b, uint8(len(p.Fields)))
		for _, field := range p.Fields {
			b = binary.LittleEndian.AppendUint16(b, field)
		}
	}
	return b, nil
}
