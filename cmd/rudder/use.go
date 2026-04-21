package main

import (
	"bufio"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"gitlab.com/dcresp0/rudder/internal/config"
	"gitlab.com/dcresp0/rudder/internal/ui"
)

func NewUseCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "use [env]",
		Short: "Switch the active environment",
		Long: `Switch the active environment.

With no argument, opens a fuzzy picker with live cluster reachability probes.
With an env name, switches directly without the picker.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.EnsureConfigDir(app.ConfigDir); err != nil {
				return err
			}

			if len(args) == 1 {
				return runUseDirect(app, args[0])
			}
			return runUsePicker(app)
		},
	}
}

// runUseDirect switches to the named environment without opening the picker.
func runUseDirect(app *App, name string) error {
	_, ok := app.Config.FindEnv(name)
	if !ok {
		return fmt.Errorf("environment %q not found — run `rudder envs` to list available environments", name)
	}

	if err := config.SaveState(app.ConfigDir, &config.State{ActiveEnvironment: name}); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	ui.PrintSuccess(app.Stdout, app.Theme, fmt.Sprintf("switched to environment %q", name))
	return nil
}

// runUsePicker opens the TUI fuzzy picker or falls back to plain mode.
func runUsePicker(app *App) error {
	if len(app.Config.Environments) == 0 {
		ui.PrintWarning(app.Stdout, app.Theme, "no environments registered — run `rudder init` or `rudder config add`")
		return nil
	}

	// Check TTY on os.Stdout (the real terminal), not app.Stdout which may be a test buffer
	if app.NoTUI || !term.IsTerminal(int(os.Stdout.Fd())) {
		return runUsePlain(app)
	}

	activeEnv := ""
	if app.State != nil {
		activeEnv = app.State.ActiveEnvironment
	}

	m := ui.NewPickerModel(app.Config.Environments, app.Theme, activeEnv)
	p := tea.NewProgram(m, tea.WithAltScreen())

	defer func() {
		if r := recover(); r != nil {
			p.ReleaseTerminal()
			panic(r) // re-panic with terminal restored
		}
	}()

	result, err := p.Run()
	if err != nil {
		app.Log.Debug("bubbletea error, falling back to plain mode", "err", err)
		return runUsePlain(app)
	}

	picked, ok := result.(ui.PickerModel)
	if !ok {
		return runUsePlain(app)
	}

	selected := picked.SelectedEnv()
	if selected == nil {
		// User cancelled
		return nil
	}

	return saveAndAnnounce(app, selected.Name)
}

// runUsePlain prompts for an environment name using plain stdin/stdout.
func runUsePlain(app *App) error {
	fmt.Fprintln(app.Stdout, "Available environments:")
	for i, e := range app.Config.Environments {
		active := " "
		if app.State != nil && e.Name == app.State.ActiveEnvironment {
			active = "*"
		}
		fmt.Fprintf(app.Stdout, "  %d. %s%s\n", i+1, active, e.Name)
	}
	fmt.Fprint(app.Stdout, "\nEnter environment name: ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return nil
	}
	name := scanner.Text()
	if name == "" {
		return nil
	}

	_, ok := app.Config.FindEnv(name)
	if !ok {
		return fmt.Errorf("environment %q not found", name)
	}

	return saveAndAnnounce(app, name)
}

func saveAndAnnounce(app *App, name string) error {
	if err := config.SaveState(app.ConfigDir, &config.State{ActiveEnvironment: name}); err != nil {
		return fmt.Errorf("save state: %w", err)
	}
	ui.PrintSuccess(app.Stdout, app.Theme, fmt.Sprintf("switched to environment %q", name))
	return nil
}
