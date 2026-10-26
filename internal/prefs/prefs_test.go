package prefs

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

const doc = `<map>
    <string name="{&quot;type&quot;:5,&quot;boltId&quot;:&quot;&quot;}">[9]</string>
    <string name="{&quot;type&quot;:53,&quot;boltId&quot;:&quot;&quot;,&quot;cfgProfileId&quot;:-1}">[7]</string>
    <string name="{&quot;type&quot;:53,&quot;boltId&quot;:&quot;&quot;}">[1,-59,0]</string>
</map>`

func TestBytes(t *testing.T) {
	got, err := Bytes([]byte(doc), 53)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{0x01, 0xc5, 0x00}; !bytes.Equal(got, want) {
		t.Fatalf("got % x, want % x", got, want)
	}
	if _, err := Bytes([]byte(doc), 99); err == nil {
		t.Fatal("expected error for missing field")
	}
}

func TestSetBytes(t *testing.T) {
	out, err := SetBytes([]byte(doc), 53, []byte{0xff, 0x02})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := Bytes(out, 53)
	if !bytes.Equal(got, []byte{0xff, 0x02}) {
		t.Fatalf("got % x", got)
	}
	if other, _ := Bytes(out, 5); !bytes.Equal(other, []byte{9}) {
		t.Fatalf("field 5 changed: % x", other)
	}
}

// TestRoundTripBackup checks that reading and writing back DISPLAY_CFG
// (field 53) leaves config backups under data/ unchanged, if any.
func TestRoundTripBackup(t *testing.T) {
	paths, _ := filepath.Glob("../../data/backups/*/BCfgManager-Bolt-*.xml")
	if len(paths) == 0 {
		t.Skip("no backups in data/backups")
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			in, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			data, err := Bytes(in, 53)
			if err != nil {
				t.Fatal(err)
			}
			out, err := SetBytes(in, 53, data)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(in, out) {
				t.Fatal("round trip changed the file")
			}
		})
	}
}

func entry(field int, value string) string {
	return `<string name="{&quot;type&quot;:` + strconv.Itoa(field) + `,&quot;boltId&quot;:&quot;&quot;}">` + value
}

func TestBytesEdgeCases(t *testing.T) {
	if got, err := Bytes([]byte(entry(7, "[]</string>")), 7); err != nil || got != nil {
		t.Fatalf("empty array: got % x, %v", got, err)
	}
	for name, doc := range map[string]string{
		"out of range": entry(7, "[128]</string>"),
		"not a number": entry(7, "[1,x]</string>"),
		"unterminated": entry(7, "[1,2"),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Bytes([]byte(doc), 7); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestSetBytesSignedRoundTrip(t *testing.T) {
	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
	}
	out, err := SetBytes([]byte(entry(7, "[]</string>")), 7, all)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := Bytes(out, 7); !bytes.Equal(got, all) {
		t.Fatalf("round trip lost bytes")
	}
}

func TestBool(t *testing.T) {
	doc := []byte(`<map>
    <boolean name="{&quot;type&quot;:18,&quot;boltId&quot;:&quot;&quot;,&quot;cfgProfileId&quot;:-1}" value="true" />
    <boolean name="{&quot;type&quot;:18,&quot;boltId&quot;:&quot;&quot;}" value="false" />
</map>`)
	if v, err := Bool(doc, 18); err != nil || v {
		t.Fatalf("got %v, %v; want false (the active value, not the profile's)", v, err)
	}
	out, err := SetBool(doc, 18, true)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := Bool(out, 18); !v {
		t.Fatal("SetBool did not change the active value")
	}
	if !bytes.Contains(out, []byte(`-1}" value="true"`)) || bytes.Count(out, []byte("true")) != 2 {
		t.Fatalf("profile value changed:\n%s", out)
	}
	if _, err := Bool(doc, 99); err == nil {
		t.Fatal("expected error for missing field")
	}
}
