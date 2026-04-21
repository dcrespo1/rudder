package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// TableModel wraps the bubbles table with Rudder's theme.
type TableModel struct {
	inner table.Model
	theme Theme
}

// NewTable creates a themed table with the given columns and rows.
func NewTable(columns []table.Column, rows []table.Row, theme Theme) TableModel {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.Muted).
		BorderBottom(true).
		Foreground(theme.Primary).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(theme.Primary).
		Background(lipgloss.AdaptiveColor{Light: "#e5e7eb", Dark: "#1e2030"}).
		Bold(false)

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithStyles(s),
	)

	return TableModel{inner: t, theme: theme}
}

// View renders the table with ANSI styling.
func (m TableModel) View() string {
	return m.inner.View()
}

// PlainView renders the table as tab-separated plain text (no ANSI).
func (m TableModel) PlainView() string {
	cols := m.inner.Columns()
	var sb strings.Builder

	// Header
	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.Title
	}
	sb.WriteString(strings.Join(headers, "\t"))
	sb.WriteByte('\n')

	// Rows
	for _, row := range m.inner.Rows() {
		sb.WriteString(strings.Join(row, "\t"))
		sb.WriteByte('\n')
	}
	return sb.String()
}
