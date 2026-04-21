package ui

import (
	"io"

	"github.com/charmbracelet/lipgloss"
)

// PrintBanner writes the Rudder branded header to w.
func PrintBanner(w io.Writer, theme Theme) {
	banner := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		Render("rudder") +
		lipgloss.NewStyle().
			Foreground(theme.Muted).
			Render("  multi-cluster kubectl wrapper")

	Println(w, banner)
	Println(w, lipgloss.NewStyle().Foreground(theme.Muted).Render("─────────────────────────────"))
}
