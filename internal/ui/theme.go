package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Theme holds all adaptive lipgloss styles for the application.
// Constructed once at startup in PersistentPreRunE — never inside bubbletea Update/View.
type Theme struct {
	Primary   lipgloss.AdaptiveColor
	Muted     lipgloss.AdaptiveColor
	Success   lipgloss.AdaptiveColor
	Warning   lipgloss.AdaptiveColor
	Danger    lipgloss.AdaptiveColor
	Active    lipgloss.Style
	Header    lipgloss.Style
	Separator lipgloss.Style
	Badge     lipgloss.Style
}

// NewTheme constructs the adaptive theme by detecting terminal background luminance.
func NewTheme() Theme {
	dark := termenv.HasDarkBackground()
	_ = dark // used implicitly via AdaptiveColor

	primary := lipgloss.AdaptiveColor{Light: "#1a1a2e", Dark: "#7dcfff"}
	muted := lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#565f89"}
	success := lipgloss.AdaptiveColor{Light: "#16a34a", Dark: "#9ece6a"}
	warning := lipgloss.AdaptiveColor{Light: "#d97706", Dark: "#e0af68"}
	danger := lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#f7768e"}

	return Theme{
		Primary: primary,
		Muted:   muted,
		Success: success,
		Warning: warning,
		Danger:  danger,
		Active: lipgloss.NewStyle().
			Foreground(success).
			Bold(true),
		Header: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true),
		Separator: lipgloss.NewStyle().
			Foreground(muted),
		Badge: lipgloss.NewStyle().
			Foreground(primary).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(muted).
			Padding(0, 1),
	}
}
