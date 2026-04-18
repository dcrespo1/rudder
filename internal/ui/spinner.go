package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerModel wraps the bubbles spinner with Rudder's theme.
type SpinnerModel struct {
	inner spinner.Model
	theme Theme
	label string
}

// NewSpinner creates a themed spinner with the given label.
func NewSpinner(theme Theme, label string) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.Primary)
	return SpinnerModel{inner: s, theme: theme, label: label}
}

func (m SpinnerModel) Init() tea.Cmd {
	return m.inner.Tick
}

func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.inner, cmd = m.inner.Update(msg)
	return m, cmd
}

func (m SpinnerModel) View() string {
	return m.inner.View() + " " + m.label
}

// PlainView returns a plain-text fallback (no ANSI codes).
func (m SpinnerModel) PlainView() string {
	return m.label + "..."
}
