package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	accent = lipgloss.AdaptiveColor{Light: "#00875F", Dark: "#00FF9F"}
	muted  = lipgloss.AdaptiveColor{Light: "#8A8A8A", Dark: "#626262"}

	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(accent).
			Padding(0, 1).
			MarginBottom(1)

	itemStyle     = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	helpStyle     = lipgloss.NewStyle().Foreground(muted).MarginTop(1)
	labelStyle    = lipgloss.NewStyle().Bold(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F"))
	artStyle      = lipgloss.NewStyle().Foreground(accent)
	mutedStyle    = lipgloss.NewStyle().Foreground(muted)
	statusStyle   = lipgloss.NewStyle().PaddingLeft(2).PaddingTop(1)
)

// crumbs builds a breadcrumb title starting at the app name.
func crumbs(path ...string) string {
	return strings.Join(append([]string{"flatlinecc"}, path...), " › ")
}

// frame lays out a screen: title bar, body, help line.
func frame(title, body, help string) string {
	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(title),
		body,
		helpStyle.Render(help),
	))
}
