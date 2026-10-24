// Package adb talks to the device by running the adb command.
package adb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ErrNotInstalled means the adb command isn't in PATH.
var ErrNotInstalled = errors.New("adb not found in PATH")

// Device is one line of `adb devices -l`.
type Device struct {
	Serial string
	State  string // device, unauthorized, offline, ...
	Model  string
}

func run(ctx context.Context, args ...string) (string, error) {
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

// Shell runs a command on the device and returns its output.
func Shell(ctx context.Context, serial string, args ...string) (string, error) {
	return run(ctx, append([]string{"-s", serial, "shell"}, args...)...)
}

// Pull reads a file from the device.
func Pull(ctx context.Context, serial, remote string) ([]byte, error) {
	tmp, err := os.CreateTemp("", "flatlinecc-*")
	if err != nil {
		return nil, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	if _, err := run(ctx, "-s", serial, "pull", remote, tmp.Name()); err != nil {
		return nil, err
	}
	return os.ReadFile(tmp.Name())
}

// Push writes data to a file on the device.
func Push(ctx context.Context, serial string, data []byte, remote string) error {
	tmp, err := os.CreateTemp("", "flatlinecc-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	_, err = run(ctx, "-s", serial, "push", tmp.Name(), remote)
	return err
}

// Devices lists attached devices (adb devices -l).
func Devices(ctx context.Context) ([]Device, error) {
	out, err := run(ctx, "devices", "-l")
	if err != nil {
		return nil, err
	}
	return parseDevices(out), nil
}

// PackageVersion returns an installed app's versionName (adb shell dumpsys package).
func PackageVersion(ctx context.Context, serial, pkg string) (string, error) {
	out, err := Shell(ctx, serial, "dumpsys", "package", pkg)
	if err != nil {
		return "", err
	}
	return parseVersion(out), nil
}

// Props returns all system properties of a device (adb shell getprop).
func Props(ctx context.Context, serial string) (map[string]string, error) {
	out, err := Shell(ctx, serial, "getprop")
	if err != nil {
		return nil, err
	}
	return parseProps(out), nil
}

func parseDevices(out string) []Device {
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
	return devices
}

func parseVersion(out string) string {
	for line := range strings.SplitSeq(out, "\n") {
		if version, ok := strings.CutPrefix(strings.TrimSpace(line), "versionName="); ok {
			return version
		}
	}
	return ""
}

func parseProps(out string) map[string]string {
	props := map[string]string{}
	for line := range strings.SplitSeq(out, "\n") {
		// [ro.product.model]: [ROAM]
		key, value, ok := strings.Cut(strings.TrimSpace(line), "]: [")
		if !ok {
			continue
		}
		props[strings.TrimPrefix(key, "[")] = strings.TrimSuffix(value, "]")
	}
	return props
}
