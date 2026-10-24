package tui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/wszczawinski/flatlinecc/internal/wahoo"
)

func TestSyncedRides(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.fit"), make([]byte, 10), 0o644)
	os.WriteFile(filepath.Join(dir, "b.fit"), make([]byte, 5), 0o644) // partial copy
	got := syncedRides(dir, []wahoo.File{{Name: "a.fit", Size: 10}, {Name: "b.fit", Size: 10}, {Name: "c.fit", Size: 3}})
	if !got["a.fit"] || got["b.fit"] || got["c.fit"] {
		t.Fatalf("got %v, want only a.fit", got)
	}
}

func TestRidesOrderAndQueue(t *testing.T) {
	dir := t.TempDir()
	r := NewRides("serial", dir)
	m, _ := r.Update(ridesMsg{rides: []wahoo.File{
		{Name: "2026-09-24-152538-x.fit", Size: 1},
		{Name: "2026-09-30-120334-x.fit", Size: 1},
		{Name: "2026-09-26-120133-x.fit", Size: 1},
	}})
	r = m.(Rides)
	if r.rides[0].Name != "2026-09-30-120334-x.fit" {
		t.Fatalf("not newest first: %v", r.rides)
	}

	os.MkdirAll(ridesDir(dir), 0o755)
	os.WriteFile(filepath.Join(ridesDir(dir), "2026-09-26-120133-x.fit"), []byte{0}, 0o644)
	r.synced = syncedRides(ridesDir(dir), r.rides)

	m, cmd := r.Update(key("s"))
	r = m.(Rides)
	want := []string{"2026-09-24-152538-x.fit", "2026-09-30-120334-x.fit"}
	if !slices.Equal(r.queue, want) || cmd == nil {
		t.Fatalf("queue %v, want oldest new first %v", r.queue, want)
	}

	// first copy done, second fails: queue stops, done one stays copied
	m, _ = r.Update(rideCopiedMsg{name: want[0]})
	r = m.(Rides)
	m, _ = r.Update(rideCopiedMsg{name: want[1], err: os.ErrPermission})
	r = m.(Rides)
	if len(r.queue) != 0 || !r.synced[want[0]] || r.synced[want[1]] || r.status == "" {
		t.Fatalf("after failure: queue %v, synced %v", r.queue, r.synced)
	}
}

func TestRideTimeAndSize(t *testing.T) {
	if got := rideTime("2026-09-30-120334-ELEMNT ROAM 1234-7-0.fit"); got != "2026-09-30 12:03" {
		t.Fatalf("rideTime: %q", got)
	}
	if got := rideTime("odd.fit"); got != "odd.fit" {
		t.Fatalf("rideTime fallback: %q", got)
	}
	for n, want := range map[int64]string{512: "512 B", 38912: "38 KB", 1258291: "1.2 MB"} {
		if got := humanSize(n); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestWindow(t *testing.T) {
	for _, tt := range []struct{ cursor, n, rows, start, end int }{
		{0, 5, 10, 0, 5},
		{0, 30, 10, 0, 10},
		{15, 30, 10, 10, 20},
		{29, 30, 10, 20, 30},
	} {
		if s, e := window(tt.cursor, tt.n, tt.rows); s != tt.start || e != tt.end {
			t.Errorf("window(%d, %d, %d) = %d, %d; want %d, %d", tt.cursor, tt.n, tt.rows, s, e, tt.start, tt.end)
		}
	}
}
