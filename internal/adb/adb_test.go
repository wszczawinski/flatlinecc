package adb

import (
	"maps"
	"slices"
	"testing"
)

func TestParseDevices(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want []Device
	}{
		{"empty", "List of devices attached\n\n", nil},
		{
			"daemon start and two devices",
			"* daemon not running; starting now at tcp:5037\n* daemon started successfully\n" +
				"List of devices attached\n" +
				"ABC123         device usb:1-1 product:roam1p1 model:ELEMNT_ROAM1_1 device:roam1p1 transport_id:1\n" +
				"XYZ            unauthorized usb:1-2 transport_id:2\n",
			[]Device{
				{Serial: "ABC123", State: "device", Model: "ELEMNT ROAM1 1"},
				{Serial: "XYZ", State: "unauthorized"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseDevices(tt.out); !slices.Equal(got, tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	out := "Packages:\n  Package [com.example] (1a2b):\n    versionCode=42\n    versionName=1.2.3\n"
	if got := parseVersion(out); got != "1.2.3" {
		t.Fatalf("got %q", got)
	}
	if got := parseVersion("nothing here"); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestParseProps(t *testing.T) {
	out := "[ro.product.model]: [ELEMNT ROAM]\n[empty]: []\ngarbage line\n"
	want := map[string]string{"ro.product.model": "ELEMNT ROAM", "empty": ""}
	if got := parseProps(out); !maps.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
