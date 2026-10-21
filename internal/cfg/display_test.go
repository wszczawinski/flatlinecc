package cfg

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDisplayRoundTrip(t *testing.T) {
	in := []byte{
		0x01, 0xc5, 0x00, 0x07, 0x00, 0x07, // header
		0x01, 0x1b, 0x00, 0x00, 0x00, 0x01, 0x05, 0x01, 0x00, 0xff, 0xff, // page head, type 1
		0x02, 0x9b, 0x00, 0x0b, 0x00, // 2 fields: 155, 11
		0x01, 0x0d, 0x00, 0x03, 0x00, 0x03, 0x01, 0x01, 0x00, 0xff, 0xff, // page head, type 3
		0x01, 0x15, 0x00, // 1 field: 21
	}
	d, err := ParseDisplay(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Pages) != 2 || d.Pages[0].ID() != 0x1b || d.Pages[1].Type() != 3 || d.Pages[0].Fields[0] != 155 {
		t.Fatalf("unexpected parse: %+v", d)
	}
	if out := d.Bytes(); !bytes.Equal(in, out) {
		t.Fatalf("round trip mismatch\n in: % x\nout: % x", in, out)
	}
}

// TestDisplayRoundTripBackup checks DISPLAY_CFG in real device files, if present.
func TestDisplayRoundTripBackup(t *testing.T) {
	paths, _ := filepath.Glob("../../data/backups/elemnt_config/bolt-*.cfg")
	if len(paths) == 0 {
		t.Skip("no backup in data/backups/elemnt_config")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			in, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			f, err := Parse(in)
			if err != nil {
				t.Fatal(err)
			}
			if name := f.Field(52); name != nil {
				t.Logf("profile name %q", name.Data)
			}
			field := f.Field(DisplayCfgID)
			if field == nil {
				t.Skip("no DISPLAY_CFG field")
			}
			d, err := ParseDisplay(field.Data)
			if err != nil {
				t.Fatal(err)
			}
			if out := d.Bytes(); !bytes.Equal(field.Data, out) {
				t.Fatalf("round trip mismatch: %d bytes in, %d out", len(field.Data), len(out))
			}
			t.Logf("header % x", d.Header)
			for _, p := range d.Pages {
				t.Logf("page %#04x type %2d head % x fields %v", p.ID(), p.Type(), p.Head, p.Fields)
			}
		})
	}
}
