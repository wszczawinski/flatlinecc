package wahoo

import (
	"slices"
	"strings"
	"testing"
)

func TestParseHashes(t *testing.T) {
	a := strings.Repeat("a", 64)
	out := a + "  /sdcard/routes/Morning loop.gpx\r\n" +
		"short  /sdcard/routes/x.gpx\n" +
		"\n"
	got := parseHashes(out)
	if len(got) != 1 || got["Morning loop.gpx"] != a {
		t.Fatalf("got %v", got)
	}
}

func TestParseFiles(t *testing.T) {
	out := "421376 /sdcard/exports/2026-09-30-120334-ELEMNT ROAM 1234-7-0.fit\r\n" +
		"38912 /sdcard/exports/2026-09-30-115948-ELEMNT ROAM 1234-6-0.fit\n" +
		"\n" +
		"garbage\n"
	want := []File{
		{"2026-09-30-120334-ELEMNT ROAM 1234-7-0.fit", 421376},
		{"2026-09-30-115948-ELEMNT ROAM 1234-6-0.fit", 38912},
	}
	if got := parseFiles(out); !slices.Equal(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if got := parseFiles(""); got != nil {
		t.Fatalf("empty: got %+v", got)
	}
}
