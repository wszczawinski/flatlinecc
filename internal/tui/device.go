package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/adb"
	"github.com/wszczawinski/flatlinecc/internal/wahoo"
)

const (
	pollInterval = 2 * time.Second
	adbTimeout   = 10 * time.Second // one read from the device
	applyTimeout = 30 * time.Second // a whole apply: pull, push, verify, restart
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

type propsMsg struct {
	props    map[string]string
	firmware string
	err      error
}

var infoRows = []struct{ label, prop string }{
	{"Manufacturer", "ro.product.manufacturer"},
	{"Model", "ro.product.model"},
	{"Serial", "ro.serialno"},
	{"Android", "ro.build.version.release"},
	{"Build", "ro.build.display.id"},
}

// DeviceInfo shows firmware and system properties of the connected device.
type DeviceInfo struct {
	serial   string
	props    map[string]string
	firmware string
	err      error
	loaded   bool
}

// NewDeviceInfo returns the device info screen for serial.
func NewDeviceInfo(serial string) DeviceInfo {
	return DeviceInfo{serial: serial}
}

func (d DeviceInfo) Init() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), adbTimeout)
		defer cancel()
		props, err := adb.Props(ctx, d.serial)
		if err != nil {
			return propsMsg{err: err}
		}
		firmware, err := adb.PackageVersion(ctx, d.serial, wahoo.App) // the app version is the firmware version
		return propsMsg{props, firmware, err}
	}
}

func (d DeviceInfo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case propsMsg:
		d.props, d.firmware, d.err, d.loaded = msg.props, msg.firmware, msg.err, true
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return d, back
		}
	}
	return d, nil
}

func (d DeviceInfo) View() string {
	var body string
	switch {
	case !d.loaded:
		body = "Reading device…"
	case d.err != nil:
		body = errorStyle.Render(d.err.Error())
	default:
		lines := []string{labelStyle.Width(14).Render("Firmware") + d.firmware}
		for _, row := range infoRows {
			lines = append(lines, labelStyle.Width(14).Render(row.label)+d.props[row.prop])
		}
		body = strings.Join(lines, "\n")
	}
	return frame(crumbs("Device info"), body, "esc back")
}
