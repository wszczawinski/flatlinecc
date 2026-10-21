package tui

import (
	"errors"

	"github.com/charmbracelet/lipgloss"

	"github.com/wszczawinski/flatlinecc/internal/adb"
)

// ROAM v1 with its name on screen.
const deviceArt = `╭──────────────────────────╮
│    ━━  ━━  ━━  ━━  ━━    │
│   ╭──────────────────╮   │
│ ┃ │                  │   │
│   │                  │   │
│ ┃ │                  │   │
│   │                  │   │
│ ┃ │                  │   │
│   │      ELEMNT      │   │
│ ┃ │       ROAM       │   │
│   │                  │   │
│ ┃ │                  │   │
│   │                  │   │
│ ┃ │                  │   │
│   │                  │   │
│ ┃ │                  │   │
│   ╰──────────────────╯   │
│╭──────╮ ╭──────╮ ╭──────╮│
│╰──────╯ ╰──────╯ ╰──────╯│
╰──────────────────────────╯`

// connectView is shown instead of the menu while no device is ready.
func connectView(checked bool, devices []adb.Device, err error) string {
	title, hint := connectMessage(checked, devices, err)
	parts := []string{artStyle.Render(deviceArt), "", labelStyle.Render(title)}
	if hint != "" {
		parts = append(parts, mutedStyle.Render(hint))
	}
	parts = append(parts, helpStyle.Render("q quit"))
	return lipgloss.JoinVertical(lipgloss.Center, parts...)
}

func connectMessage(checked bool, devices []adb.Device, err error) (title, hint string) {
	switch {
	case !checked:
		return "Looking for device…", ""
	case errors.Is(err, adb.ErrNotInstalled):
		return "adb not found", "See README for setup"
	case err != nil:
		return "adb error", err.Error()
	case len(devices) == 0:
		return "Connect your ROAM", "Plug it in with a USB cable"
	}
	if devices[0].State == "unauthorized" {
		return "Device not authorized", "Unplug · power · power · plug back in"
	}
	return "Device is " + devices[0].State, "Try unplugging and plugging it back in"
}
