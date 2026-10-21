// Package cfg reads and writes ELEMNT .cfg files (see docs/format.md).
package cfg

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Field struct {
	ID   uint32
	Data []byte
}

type File struct {
	Version   uint8 // 0: u16 field IDs, 1: u32 field IDs
	ProfileID uint16
	Fields    []Field
	Trailer   []byte // bytes after the last field, kept as-is
}

// Parse decodes a .cfg file. Field data slices point into b.
func Parse(b []byte) (*File, error) {
	r := reader{b: b}
	f := &File{Version: r.u8(), ProfileID: r.u16()}
	count := r.u16()
	if r.err != nil {
		return nil, fmt.Errorf("header: %w", r.err)
	}
	if f.Version > 1 {
		return nil, fmt.Errorf("unsupported version %d", f.Version)
	}
	for i := range count {
		var id uint32
		if f.Version == 0 {
			id = uint32(r.u16())
		} else {
			id = r.u32()
		}
		data := r.bytes(int(r.u16()))
		if r.err != nil {
			return nil, fmt.Errorf("field %d of %d: %w", i, count, r.err)
		}
		f.Fields = append(f.Fields, Field{ID: id, Data: data})
	}
	f.Trailer = r.b
	return f, nil
}

// Field returns the field with the given ID, or nil.
func (f *File) Field(id uint32) *Field {
	for i := range f.Fields {
		if f.Fields[i].ID == id {
			return &f.Fields[i]
		}
	}
	return nil
}

// Bytes encodes the file.
func (f *File) Bytes() []byte {
	le := binary.LittleEndian
	b := []byte{f.Version}
	b = le.AppendUint16(b, f.ProfileID)
	b = le.AppendUint16(b, uint16(len(f.Fields)))
	for _, field := range f.Fields {
		if f.Version == 0 {
			b = le.AppendUint16(b, uint16(field.ID))
		} else {
			b = le.AppendUint32(b, field.ID)
		}
		b = le.AppendUint16(b, uint16(len(field.Data)))
		b = append(b, field.Data...)
	}
	return append(b, f.Trailer...)
}

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

func (r *reader) u32() uint32 {
	if b := r.bytes(4); b != nil {
		return binary.LittleEndian.Uint32(b)
	}
	return 0
}
