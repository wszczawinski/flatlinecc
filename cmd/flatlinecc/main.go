package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/config"
	"github.com/wszczawinski/flatlinecc/internal/tui"
)

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("flatlinecc", version())
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	m, err := tea.NewProgram(tui.New(cfg), tea.WithAltScreen()).Run()
	if err != nil {
		// Keep the session backup: something went wrong.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	app := m.(tui.App)
	if path := app.SessionBackup(); path != "" && !app.KeepBackup() {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "remove session backup:", err)
		}
	}
}

// version is the module version Go stamps from the git tag at build time,
// e.g. v0.4.0, or a pseudo-version between tags.
func version() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "unknown"
}
