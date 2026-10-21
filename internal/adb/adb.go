// Package adb talks to the device by running the adb command.
package adb

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var ErrNotInstalled = errors.New("adb not found in PATH")

const timeout = 5 * time.Second

type Device struct {
	Serial string
	State  string // device, unauthorized, offline, ...
	Model  string
}

func run(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "adb", args...).Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", ErrNotInstalled
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("adb %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

// Devices lists attached devices (adb devices -l).
func Devices() ([]Device, error) {
	out, err := run("devices", "-l")
	if err != nil {
		return nil, err
	}
	var devices []Device
	for line := range strings.SplitSeq(out, "\n") {
		if strings.HasPrefix(line, "List of devices") || strings.HasPrefix(line, "*") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		d := Device{Serial: fields[0], State: fields[1]}
		for _, f := range fields[2:] {
			if model, ok := strings.CutPrefix(f, "model:"); ok {
				d.Model = strings.ReplaceAll(model, "_", " ")
			}
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// PackageVersion returns an installed app's versionName (adb shell dumpsys package).
func PackageVersion(serial, pkg string) (string, error) {
	out, err := run("-s", serial, "shell", "dumpsys", "package", pkg)
	if err != nil {
		return "", err
	}
	for line := range strings.SplitSeq(out, "\n") {
		if version, ok := strings.CutPrefix(strings.TrimSpace(line), "versionName="); ok {
			return version, nil
		}
	}
	return "", nil
}

// Props returns all system properties of a device (adb shell getprop).
func Props(serial string) (map[string]string, error) {
	out, err := run("-s", serial, "shell", "getprop")
	if err != nil {
		return nil, err
	}
	props := map[string]string{}
	for line := range strings.SplitSeq(out, "\n") {
		// [ro.product.model]: [ROAM]
		key, value, ok := strings.Cut(strings.TrimSpace(line), "]: [")
		if !ok {
			continue
		}
		props[strings.TrimPrefix(key, "[")] = strings.TrimSuffix(value, "]")
	}
	return props, nil
}
