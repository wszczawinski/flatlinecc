// Package display reads and writes DISPLAY_CFG, the data page layout
// (see docs/format.md).
package display

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// ConfigField is DISPLAY_CFG's field number in the live config.
const ConfigField = 53

const (
	headerSize   = 6
	pageHeadSize = 11
)

// Layout is a decoded DISPLAY_CFG. Bytes with unknown meaning are kept as-is.
type Layout struct {
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

// Type returns the page type (workout, map, ...), see the datafields package for names.
func (p Page) Type() uint8 { return p.Head[5] }

// Parse decodes DISPLAY_CFG data. Header and Head slices point into b.
func Parse(b []byte) (*Layout, error) {
	r := reader{b: b}
	l := &Layout{Header: r.bytes(headerSize)}
	if r.err != nil {
		return nil, fmt.Errorf("header: %w", r.err)
	}
	for len(r.b) > 0 {
		p := Page{Head: r.bytes(pageHeadSize)}
		for range r.u8() {
			p.Fields = append(p.Fields, r.u16())
		}
		if r.err != nil {
			return nil, fmt.Errorf("page %d: %w", len(l.Pages), r.err)
		}
		l.Pages = append(l.Pages, p)
	}
	return l, nil
}

// Bytes encodes the layout. A page holds at most 255 fields.
func (l *Layout) Bytes() ([]byte, error) {
	b := append([]byte{}, l.Header...)
	for i, p := range l.Pages {
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

// reader reads little-endian values. After the first short read it returns
// zeros and keeps the error.
type reader struct {
	b   []byte
	err error
}

func (r *reader) bytes(n int) []byte {
	if r.err != nil {
		return nil
	}
	if len(r.b) < n {
		r.err = io.ErrUnexpectedEOF
		return nil
	}
	out := r.b[:n]
	r.b = r.b[n:]
	return out
}

func (r *reader) u8() uint8 {
	if b := r.bytes(1); b != nil {
		return b[0]
	}
	return 0
}

func (r *reader) u16() uint16 {
	if b := r.bytes(2); b != nil {
		return binary.LittleEndian.Uint16(b)
	}
	return 0
}
