package cfg

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	in := []byte{
		0x00, 0xff, 0xff, 0x02, 0x00, // version 0, profile 65535, 2 fields
		0x34, 0x00, 0x03, 0x00, 'a', 'b', 0x00, // field 52, 3 bytes
		0x63, 0x00, 0x01, 0x00, 0x05, // field 99, 1 byte
		0x00, 0x00, // trailer
	}
	f, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	if f.ProfileID != 65535 || len(f.Fields) != 2 || f.Fields[1].ID != 99 {
		t.Fatalf("unexpected parse: %+v", f)
	}
	if out := f.Bytes(); !bytes.Equal(in, out) {
		t.Fatalf("round trip mismatch\n in: % x\nout: % x", in, out)
	}
}

func TestParseTruncated(t *testing.T) {
	if _, err := Parse([]byte{0x00, 0xff, 0xff, 0x01, 0x00, 0x34}); err == nil {
		t.Fatal("expected error")
	}
}

// TestRoundTripBackup checks real device files in data/backups, if present.
func TestRoundTripBackup(t *testing.T) {
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
			if out := f.Bytes(); !bytes.Equal(in, out) {
				t.Fatalf("round trip mismatch: %d bytes in, %d out", len(in), len(out))
			}
			t.Logf("%d fields, trailer % x", len(f.Fields), f.Trailer)
		})
	}
}
