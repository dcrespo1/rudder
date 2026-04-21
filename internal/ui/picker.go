package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gitlab.com/dcresp0/rudder/internal/cluster"
	"gitlab.com/dcresp0/rudder/internal/config"
)

// PingResultMsg carries an async ping result into the bubbletea model.
type PingResultMsg struct {
	EnvName string
	Status  cluster.Status
}

// PickerModel is the bubbletea model for the fuzzy environment picker.
type PickerModel struct {
	envs      []config.Environment
	theme     Theme
	active    string // currently persisted active env name
	filter    string
	cursor    int
	statuses  map[string]cluster.Status
	tooSmall  bool
	done      bool
	selected  *config.Environment
	width     int
	height    int
}

// NewPickerModel creates a PickerModel for the given environments.
// active is the currently persisted active environment name (may be empty).
func NewPickerModel(envs []config.Environment, theme Theme, active string) PickerModel {
	statuses := make(map[string]cluster.Status, len(envs))
	for _, e := range envs {
		statuses[e.Name] = cluster.StatusProbing
	}
	return PickerModel{
		envs:     envs,
		theme:    theme,
		active:   active,
		statuses: statuses,
	}
}

// Init starts async ping probes for all environments.
func (m PickerModel) Init() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.envs))
	for i, env := range m.envs {
		env := env
		cmds[i] = func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			kubeconfigPath := config.ExpandPath(env.Kubeconfig)
			status := cluster.Ping(ctx, kubeconfigPath, env.Context)
			return PingResultMsg{EnvName: env.Name, Status: status}
		}
	}
	return tea.Batch(cmds...)
}

// Update handles all incoming messages.
func (m PickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tooSmall = msg.Width < 40 || msg.Height < 5

	case PingResultMsg:
		m.statuses[msg.EnvName] = msg.Status

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.done = true
			return m, tea.Quit

		case "enter":
			filtered := m.filtered()
			if len(filtered) > 0 && m.cursor < len(filtered) {
				env := filtered[m.cursor]
				m.selected = &env
			}
			m.done = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			filtered := m.filtered()
			if m.cursor < len(filtered)-1 {
				m.cursor++
			}

		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.cursor = 0
			}

		default:
			if len(msg.String()) == 1 {
				m.filter += msg.String()
				m.cursor = 0
			}
		}
	}

	return m, nil
}

// View renders the picker UI.
func (m PickerModel) View() string {
	if m.tooSmall {
		return lipgloss.NewStyle().Foreground(m.theme.Warning).Render("terminal too small — resize to use picker")
	}

	var sb strings.Builder

	// Search bar
	filterLine := lipgloss.NewStyle().Foreground(m.theme.Muted).Render("  filter: ") +
		m.filter + "█"
	sb.WriteString(filterLine)
	sb.WriteByte('\n')
	sb.WriteString(lipgloss.NewStyle().Foreground(m.theme.Muted).Render(strings.Repeat("─", 54)))
	sb.WriteByte('\n')

	filtered := m.filtered()
	if len(filtered) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(m.theme.Muted).Render("  no environments match"))
		sb.WriteByte('\n')
		return sb.String()
	}

	for i, env := range filtered {
		cursor := "  "
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(m.theme.Primary).Render("> ")
		}

		statusIcon := statusIcon(m.statuses[env.Name], m.theme)
		activeMarker := " "
		if env.Name == m.active {
			activeMarker = lipgloss.NewStyle().Foreground(m.theme.Success).Render("*")
		}

		tags := ""
		if len(env.Tags) > 0 {
			tags = lipgloss.NewStyle().Foreground(m.theme.Muted).Render("[" + strings.Join(env.Tags, ", ") + "]")
		}

		desc := ""
		if env.Description != "" {
			desc = lipgloss.NewStyle().Foreground(m.theme.Muted).Render("  " + env.Description)
		}

		line := fmt.Sprintf("%s%s%s %-20s %s%s",
			cursor, activeMarker, statusIcon,
			lipgloss.NewStyle().Bold(i == m.cursor).Render(env.Name),
			tags, desc,
		)
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	sb.WriteString(lipgloss.NewStyle().Foreground(m.theme.Muted).Render("\n  ↑/↓ navigate  enter select  esc cancel"))
	return sb.String()
}

// PlainView returns a plain-text list of environments (no ANSI). Used when TUI is disabled.
func (m PickerModel) PlainView() string {
	var sb strings.Builder
	for i, env := range m.envs {
		active := " "
		if env.Name == m.active {
			active = "*"
		}
		sb.WriteString(fmt.Sprintf("%d. %s%-20s  %s\n", i+1, active, env.Name, env.Description))
	}
	return sb.String()
}

// SelectedEnv returns the environment chosen by the user, or nil if cancelled.
func (m PickerModel) SelectedEnv() *config.Environment {
	return m.selected
}

// filtered returns the environments matching the current filter string.
func (m PickerModel) filtered() []config.Environment {
	if m.filter == "" {
		return m.envs
	}
	q := strings.ToLower(m.filter)
	var result []config.Environment
	for _, env := range m.envs {
		if strings.Contains(strings.ToLower(env.Name), q) ||
			strings.Contains(strings.ToLower(env.Description), q) ||
			tagsContain(env.Tags, q) {
			result = append(result, env)
		}
	}
	return result
}

func tagsContain(tags []string, q string) bool {
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return false
}

func statusIcon(s cluster.Status, theme Theme) string {
	switch s {
	case cluster.StatusReachable:
		return lipgloss.NewStyle().Foreground(theme.Success).Render("●")
	case cluster.StatusUnreachable:
		return lipgloss.NewStyle().Foreground(theme.Danger).Render("●")
	case cluster.StatusProbing:
		return lipgloss.NewStyle().Foreground(theme.Muted).Render("◌")
	default:
		return lipgloss.NewStyle().Foreground(theme.Muted).Render("○")
	}
}
