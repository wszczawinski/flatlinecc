package tui

import "path/filepath"

// Folders inside the data folder. Each is created on first write.

func ridesDir(dataDir string) string {
	return filepath.Join(dataDir, "rides")
}

func routesDir(dataDir string) string {
	return filepath.Join(dataDir, "routes")
}

// manualBackupDir holds backups made from the Backup / restore screen.
// They are never removed automatically.
func manualBackupDir(dataDir string) string {
	return filepath.Join(dataDir, "backups", "manual")
}

// sessionBackupDir holds the config as it was when the session started.
// The backup is removed on a clean quit and kept otherwise.
func sessionBackupDir(dataDir string) string {
	return filepath.Join(dataDir, "backups", "session")
}
