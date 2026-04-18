package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Println writes a line to w.
func Println(w io.Writer, args ...interface{}) {
	fmt.Fprintln(w, args...)
}

// Printf writes a formatted string to w.
func Printf(w io.Writer, format string, args ...interface{}) {
	fmt.Fprintf(w, format, args...)
}

// PrintError writes a styled error message to w.
func PrintError(w io.Writer, err error) {
	fmt.Fprintf(w, "error: %s\n", err.Error())
}

// PrintSuccess writes a styled success message to w.
func PrintSuccess(w io.Writer, theme Theme, msg string) {
	line := lipgloss.NewStyle().Foreground(theme.Success).Render("✓") + " " + msg
	fmt.Fprintln(w, line)
}

// PrintWarning writes a styled warning message to w.
func PrintWarning(w io.Writer, theme Theme, msg string) {
	line := lipgloss.NewStyle().Foreground(theme.Warning).Render("⚠") + " " + msg
	fmt.Fprintln(w, line)
}

// PrintClusterHeader writes the styled cluster separator header used in exec --all output.
func PrintClusterHeader(w io.Writer, theme Theme, envName string) {
	const totalWidth = 54
	label := " " + envName + " "
	dashes := strings.Repeat("─", max(0, totalWidth-len(label)-3))
	line := theme.Separator.Render("─── " + label + dashes)
	fmt.Fprintln(w, line)
}

