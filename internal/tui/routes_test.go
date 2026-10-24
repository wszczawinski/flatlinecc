package tui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/wszczawinski/flatlinecc/internal/wahoo"
)

func TestLocalRoutes(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "b.gpx"), make([]byte, 3), 0o644)
	os.WriteFile(filepath.Join(dir, "A.GPX"), make([]byte, 2), 0o644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), nil, 0o644)
	os.WriteFile(filepath.Join(dir, "._b.gpx"), nil, 0o644)
	os.Mkdir(filepath.Join(dir, "sub.gpx"), 0o755)

	got, err := localRoutes(dir)
	want := []wahoo.File{{Name: "A.GPX", Size: 2}, {Name: "b.gpx", Size: 3}}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("got %v, %v; want %v", got, err, want)
	}
	if got, err := localRoutes(filepath.Join(dir, "missing")); err != nil || got != nil {
		t.Fatalf("missing dir: got %v, %v", got, err)
	}
}

func TestMergeRoutes(t *testing.T) {
	local := []wahoo.File{{Name: "same.gpx", Size: 5}, {Name: "edited.gpx", Size: 9}, {Name: "new.gpx", Size: 1}}
	remote := []wahoo.File{{Name: "same.gpx", Size: 5}, {Name: "edited.gpx", Size: 7}, {Name: "device.gpx", Size: 4}}
	got := mergeRoutes(local, remote)

	want := []struct {
		name       string
		push, pull bool
	}{
		{"device.gpx", false, true},
		{"edited.gpx", true, false},
		{"new.gpx", true, false},
		{"same.gpx", false, false},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i, w := range want {
		if got[i].name != w.name || got[i].push() != w.push || got[i].pull() != w.pull {
			t.Errorf("%d: got %+v push=%v pull=%v, want %+v", i, got[i], got[i].push(), got[i].pull(), w)
		}
	}
}

func TestRoutesQueueStopsOnError(t *testing.T) {
	r := NewRoutes("serial", t.TempDir())
	m, _ := r.Update(routesMsg{routes: mergeRoutes(
		[]wahoo.File{{Name: "a.gpx", Size: 1}, {Name: "b.gpx", Size: 1}},
		[]wahoo.File{{Name: "b.gpx", Size: 1}, {Name: "c.gpx", Size: 1}},
	)})
	r = m.(Routes)
	m, cmd := r.Update(key("s"))
	r = m.(Routes)
	var names []string
	for _, rt := range r.queue {
		names = append(names, rt.name)
	}
	if !slices.Equal(names, []string{"a.gpx", "c.gpx"}) || cmd == nil {
		t.Fatalf("queue %v", names)
	}
	m, _ = r.Update(routeSyncedMsg{})
	r = m.(Routes)
	m, cmd = r.Update(routeSyncedMsg{err: os.ErrPermission})
	r = m.(Routes)
	if len(r.queue) != 0 || r.synced != 1 || r.status == "" || cmd == nil {
		t.Fatalf("after failure: queue %v, synced %d, reload %v", r.queue, r.synced, cmd != nil)
	}
}
