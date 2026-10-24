package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wszczawinski/flatlinecc/internal/config"
	"github.com/wszczawinski/flatlinecc/internal/tui"
)

func main() {
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
