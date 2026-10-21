package prefs

import (
	"bytes"
	"os"
	"testing"

	"github.com/wszczawinski/flatlinecc/internal/cfg"
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

// TestRealFile checks DISPLAY_CFG in a pulled config, if present.
func TestRealFile(t *testing.T) {
	in, err := os.ReadFile("../../data/BCfgManager-Bolt.orig.xml")
	if err != nil {
		t.Skip("no data/BCfgManager-Bolt.orig.xml")
	}
	data, err := Bytes(in, cfg.DisplayCfgID)
	if err != nil {
		t.Fatal(err)
	}
	d, err := cfg.ParseDisplay(data)
	if err != nil {
		t.Fatal(err)
	}
	out, err := SetBytes(in, cfg.DisplayCfgID, d.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(in, out) {
		t.Fatal("round trip changed the file")
	}
	t.Logf("%d pages", len(d.Pages))
}
