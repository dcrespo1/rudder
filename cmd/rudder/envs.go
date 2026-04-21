package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"gitlab.com/dcresp0/rudder/internal/config"
	"gitlab.com/dcresp0/rudder/internal/ui"
)

func NewEnvsCmd(app *App) *cobra.Command {
	var tags []string
	var output string

	cmd := &cobra.Command{
		Use:   "envs",
		Short: "List all registered environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			envs := app.Config.FilterByTags(tags)

			activeEnv := ""
			if app.State != nil {
				activeEnv = app.State.ActiveEnvironment
			}
			// RUDDER_ENV overrides persisted state for display
			if e := os.Getenv("RUDDER_ENV"); e != "" {
				activeEnv = e
			}

			switch output {
			case "json":
				return printEnvsJSON(cmd, envs, activeEnv)
			default:
				return printEnvsTable(cmd, envs, activeEnv, app.Theme, app.NoTUI)
			}
		},
	}

	cmd.Flags().StringSliceVar(&tags, "tags", nil, "filter by tags (comma-separated)")
	cmd.Flags().StringVarP(&output, "output", "o", "table", "output format: table, json")
	return cmd
}

type envJSON struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Kubeconfig  string   `json:"kubeconfig"`
	Context     string   `json:"context,omitempty"`
	Namespace   string   `json:"namespace,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Active      bool     `json:"active"`
}

func printEnvsJSON(cmd *cobra.Command, envs []config.Environment, activeEnv string) error {
	out := make([]envJSON, len(envs))
	for i, e := range envs {
		out[i] = envJSON{
			Name:        e.Name,
			Description: e.Description,
			Kubeconfig:  e.Kubeconfig,
			Context:     e.Context,
			Namespace:   e.Namespace,
			Tags:        e.Tags,
			Active:      e.Name == activeEnv,
		}
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	cmd.Println(string(data))
	return nil
}

func printEnvsTable(cmd *cobra.Command, envs []config.Environment, activeEnv string, theme ui.Theme, plain bool) error {
	if len(envs) == 0 {
		ui.PrintWarning(cmd.OutOrStdout(), theme, "no environments registered — run `rudder init` or `rudder config add`")
		return nil
	}

	columns := []table.Column{
		{Title: "  ", Width: 2},
		{Title: "NAME", Width: 20},
		{Title: "CONTEXT", Width: 22},
		{Title: "NAMESPACE", Width: 12},
		{Title: "TAGS", Width: 24},
	}

	rows := make([]table.Row, len(envs))
	for i, e := range envs {
		active := " "
		if e.Name == activeEnv {
			active = "●"
		}
		ctx := e.Context
		if ctx == "" {
			ctx = "(current-context)"
		}
		ns := e.Namespace
		if ns == "" {
			ns = "-"
		}
		rows[i] = table.Row{active, e.Name, ctx, ns, strings.Join(e.Tags, ", ")}
	}

	if plain {
		// Plain text output — no ANSI
		t := ui.NewTable(columns, rows, theme)
		cmd.Print(t.PlainView())
		return nil
	}

	// Styled table
	t := ui.NewTable(columns, rows, theme)

	// Highlight active row manually
	activeStyle := lipgloss.NewStyle().Foreground(theme.Success)
	var sb strings.Builder
	lines := strings.Split(t.View(), "\n")
	for _, line := range lines {
		if len(line) > 2 && strings.Contains(line, "●") {
			sb.WriteString(activeStyle.Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteByte('\n')
	}
	cmd.Print(sb.String())
	return nil
}
