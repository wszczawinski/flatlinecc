package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/adb"
)

const (
	pollInterval = 2 * time.Second
	adbTimeout   = 10 * time.Second // one read from the device
	applyTimeout = 30 * time.Second // a whole apply: pull, push, verify, restart
	rideTimeout  = 2 * time.Minute  // one .fit file
	routeTimeout = time.Minute      // one .gpx file
)

type devicesMsg struct {
	devices []adb.Device
	err     error
}

type pollMsg struct{}

func checkDevices() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), adbTimeout)
	defer cancel()
	devices, err := adb.Devices(ctx)
	return devicesMsg{devices, err}
}

func poll() tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg { return pollMsg{} })
}

func statusLine(d *adb.Device) string {
	name := d.Model
	if name == "" {
		name = d.Serial
	}
	return selectedStyle.Render("● " + name + " connected")
}
