package datafields

import (
	"slices"
	"testing"
)

func TestForPage(t *testing.T) {
	const workout, lap, elevation, kickr, summit = 0, 1, 2, 4, 39
	tests := []struct {
		name  string
		page  uint8
		id    uint16
		offer bool
	}{
		{"general field everywhere", workout, 10, true},
		{"general field on lap page", lap, 10, true},
		{"lap field on lap page", lap, 11, true},
		{"lap field hidden elsewhere", workout, 11, false},
		{"kickr field on kickr page", kickr, 120, true},
		{"kickr field hidden on lap page", lap, 120, false},
		{"summit field on summit page", summit, 426, true},
		{"summit field hidden on elevation page", elevation, 426, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slices.Contains(ForPage(tt.page), tt.id); got != tt.offer {
				t.Fatalf("ForPage(%d) contains %d = %v, want %v", tt.page, tt.id, got, tt.offer)
			}
		})
	}
}

func TestPageOnlyFieldsAreNamed(t *testing.T) {
	for page, ids := range pageOnly {
		for _, id := range ids {
			if _, ok := names[id]; !ok {
				t.Errorf("page %d lists field %d, which has no name", page, id)
			}
		}
	}
}

func TestNameFallback(t *testing.T) {
	if got := Name(65000); got != "Field 65000" {
		t.Fatalf("got %q", got)
	}
	if got := PageName(200); got != "Page type 200" {
		t.Fatalf("got %q", got)
	}
}
