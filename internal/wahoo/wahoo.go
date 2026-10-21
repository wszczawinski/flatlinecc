// Package wahoo reads and applies the Wahoo app's live config (see docs/format.md).
package wahoo

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wszczawinski/flatlinecc/internal/adb"
)

const (
	App        = "com.wahoofitness.bolt"
	ConfigPath = "/data/data/com.wahoofitness.bolt/shared_prefs/BCfgManager-Bolt.xml"
)

// PullConfig reads the live config.
func PullConfig(serial string) ([]byte, error) {
	return adb.Pull(serial, ConfigPath)
}

// ApplyConfig saves the current config to backupDir, writes doc in its place
// and restarts the app so it loads it. It returns the backup's path.
func ApplyConfig(serial string, doc []byte, backupDir string) (string, error) {
	current, err := adb.Pull(serial, ConfigPath)
	if err != nil {
		return "", err
	}
	owner, err := adb.Shell(serial, "stat", "-c", "%u:%g", ConfigPath)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	backup := filepath.Join(backupDir, "BCfgManager-Bolt-"+time.Now().Format("20060102-150405")+".xml")
	if err := os.WriteFile(backup, current, 0o644); err != nil {
		return "", err
	}

	if err := adb.Push(serial, doc, ConfigPath); err != nil {
		return backup, err
	}
	if _, err := adb.Shell(serial, "chown", strings.TrimSpace(owner), ConfigPath); err != nil {
		return backup, err
	}
	if _, err := adb.Shell(serial, "chmod", "660", ConfigPath); err != nil {
		return backup, err
	}

	// Verify before the restart: the app rewrites the file when it starts.
	written, err := adb.Pull(serial, ConfigPath)
	if err != nil {
		return backup, err
	}
	if !bytes.Equal(written, doc) {
		return backup, errors.New("config changed on the device before restart, try again")
	}

	// The launcher restarts the app right away and it reads the new config.
	_, err = adb.Shell(serial, "am", "force-stop", App)
	return backup, err
}
