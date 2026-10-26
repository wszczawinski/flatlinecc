package display

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/wszczawinski/flatlinecc/internal/prefs"
)

func TestRoundTrip(t *testing.T) {
	in := []byte{
		0x01, 0xc5, 0x00, 0x07, 0x00, 0x07, // header
		0x01, 0x1b, 0x00, 0x00, 0x00, 0x01, 0x05, 0x01, 0x00, 0xff, 0xff, // page head, type 1
		0x02, 0x9b, 0x00, 0x0b, 0x00, // 2 fields: 155, 11
		0x01, 0x0d, 0x00, 0x03, 0x00, 0x03, 0x01, 0x01, 0x00, 0xff, 0xff, // page head, type 3
		0x01, 0x15, 0x00, // 1 field: 21
	}
	l, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Pages) != 2 || l.Pages[0].ID() != 0x1b || l.Pages[1].Type() != 3 || l.Pages[0].Fields[0] != 155 {
		t.Fatalf("unexpected parse: %+v", l)
	}
	out, err := l.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(in, out) {
		t.Fatalf("round trip mismatch\n in: % x\nout: % x", in, out)
	}
}

// TestRoundTripBackup checks the layout in config backups under data/, if any.
func TestRoundTripBackup(t *testing.T) {
	paths, _ := filepath.Glob("../../data/backups/*/BCfgManager-Bolt-*.xml")
	if len(paths) == 0 {
		t.Skip("no backups in data/backups")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			doc, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			data, err := prefs.Bytes(doc, ConfigField)
			if err != nil {
				t.Fatal(err)
			}
			l, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			out, err := l.Bytes()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(data, out) {
				t.Fatalf("round trip mismatch: %d bytes in, %d out", len(data), len(out))
			}
			t.Logf("header % x", l.Header)
			for _, p := range l.Pages {
				t.Logf("page %#04x type %2d head % x fields %v", p.ID(), p.Type(), p.Head, p.Fields)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := map[string][]byte{
		"short header":   {0x01, 0xc5},
		"short head":     {0x01, 0xc5, 0x00, 0x07, 0x00, 0x07, 0x01, 0x1b},
		"missing fields": {0x01, 0xc5, 0x00, 0x07, 0x00, 0x07, 0x01, 0x1b, 0x00, 0x00, 0x00, 0x01, 0x05, 0x01, 0x00, 0xff, 0xff, 0x02, 0x9b},
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(in); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestBytesTooManyFields(t *testing.T) {
	l := &Layout{
		Header: make([]byte, headerSize),
		Pages:  []Page{{Head: make([]byte, pageHeadSize), Fields: make([]uint16, 256)}},
	}
	if _, err := l.Bytes(); err == nil {
		t.Fatal("expected error for 256 fields")
	}
}

// FuzzParse checks that anything Parse accepts encodes back to the same bytes.
func FuzzParse(f *testing.F) {
	f.Add([]byte{
		0x01, 0xc5, 0x00, 0x07, 0x00, 0x07,
		0x01, 0x1b, 0x00, 0x00, 0x00, 0x01, 0x05, 0x01, 0x00, 0xff, 0xff,
		0x01, 0x9b, 0x00,
	})
	f.Fuzz(func(t *testing.T, in []byte) {
		l, err := Parse(in)
		if err != nil {
			return
		}
		out, err := l.Bytes()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(in, out) {
			t.Fatalf("round trip mismatch\n in: % x\nout: % x", in, out)
		}
	})
}
