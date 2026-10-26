// Package wahoo reads and applies the Wahoo app's live config (see docs/format.md).
package wahoo

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wszczawinski/flatlinecc/internal/adb"
)

const (
	// App is the Wahoo app package on the device.
	App = "com.wahoofitness.bolt"
	// ConfigPath is the app's live config (SharedPreferences).
	ConfigPath = prefsDir + "/BCfgManager-Bolt.xml"

	prefsDir = "/data/data/com.wahoofitness.bolt/shared_prefs"
)

// PullConfig reads the live config.
func PullConfig(ctx context.Context, serial string) ([]byte, error) {
	return adb.Pull(ctx, serial, ConfigPath)
}

// BackupConfig saves the live config to backupDir and returns the backup's path.
func BackupConfig(ctx context.Context, serial, backupDir string) (string, error) {
	current, err := adb.Pull(ctx, serial, ConfigPath)
	if err != nil {
		return "", err
	}
	return saveBackup(current, backupDir)
}

func saveBackup(doc []byte, backupDir string) (string, error) {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	backup := filepath.Join(backupDir, "BCfgManager-Bolt-"+time.Now().Format("20060102-150405")+".xml")
	return backup, os.WriteFile(backup, doc, 0o644)
}

// ApplyConfig writes doc in place of the live config and restarts the app so
// it loads it. Back up first with BackupConfig.
func ApplyConfig(ctx context.Context, serial string, doc []byte) error {
	// The folder always belongs to the app. The file may not: push creates it
	// as root, and an apply cut off before the chown leaves it that way.
	owner, err := adb.Shell(ctx, serial, "stat", "-c", "%u:%g", prefsDir)
	if err != nil {
		return err
	}

	if err := adb.Push(ctx, serial, doc, ConfigPath); err != nil {
		return err
	}
	if _, err := adb.Shell(ctx, serial, "chown", strings.TrimSpace(owner), ConfigPath); err != nil {
		return err
	}
	if _, err := adb.Shell(ctx, serial, "chmod", "660", ConfigPath); err != nil {
		return err
	}

	// Verify before the restart: the app rewrites the file when it starts.
	written, err := adb.Pull(ctx, serial, ConfigPath)
	if err != nil {
		return err
	}
	if !bytes.Equal(written, doc) {
		return errors.New("config changed on the device before restart, try again")
	}

	// The launcher restarts the app right away and it reads the new config.
	_, err = adb.Shell(ctx, serial, "am", "force-stop", App)
	return err
}
