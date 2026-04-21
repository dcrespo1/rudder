package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dcrespo1/rudder/internal/config"
	"github.com/dcrespo1/rudder/internal/ui"
)

func NewConfigCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage the environment registry",
	}

	cmd.AddCommand(newConfigAddCmd(app))
	cmd.AddCommand(newConfigRemoveCmd(app))
	cmd.AddCommand(newConfigRenameCmd(app))
	cmd.AddCommand(newConfigListCmd(app))
	cmd.AddCommand(newConfigEditCmd(app))
	cmd.AddCommand(newConfigValidateCmd(app))

	return cmd
}

func newConfigAddCmd(app *App) *cobra.Command {
	var (
		name        string
		kubeconfig  string
		ctx         string
		namespace   string
		description string
		tags        []string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.EnsureConfigDir(app.ConfigDir); err != nil {
				return err
			}

			var env config.Environment

			if name != "" && kubeconfig != "" {
				// Non-interactive: all required flags provided
				env = config.Environment{
					Name:        name,
					Description: description,
					Kubeconfig:  kubeconfig,
					Context:     ctx,
					Namespace:   namespace,
					Tags:        tags,
				}
			} else {
				// Interactive wizard
				var err error
				env, err = promptNewEnvironment(app)
				if err != nil {
					return err
				}
			}

			// Validate before saving
			errs := config.ValidateEnvironment(&env)
			if len(errs) > 0 {
				for _, e := range errs {
					ui.PrintError(app.Stderr, e)
				}
				return fmt.Errorf("validation failed")
			}

			// Check for duplicate name
			if _, exists := app.Config.FindEnv(env.Name); exists {
				return fmt.Errorf("environment %q already exists — use `rudder config rename` to rename it", env.Name)
			}

			app.Config.Environments = append(app.Config.Environments, env)
			if err := config.SaveConfig(app.ConfigDir, app.Config); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			ui.PrintSuccess(app.Stdout, app.Theme, fmt.Sprintf("added environment %q", env.Name))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "environment name")
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file")
	cmd.Flags().StringVar(&ctx, "context", "", "context name within the kubeconfig (default: current-context)")
	cmd.Flags().StringVar(&namespace, "namespace", "", "default namespace override")
	cmd.Flags().StringVar(&description, "description", "", "human-readable description")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "comma-separated tags")
	return cmd
}

func newConfigRemoveCmd(app *App) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if _, ok := app.Config.FindEnv(name); !ok {
				return fmt.Errorf("environment %q not found", name)
			}

			if !yes {
				fmt.Fprintf(app.Stdout, "Remove environment %q? [y/N] ", name)
				scanner := bufio.NewScanner(os.Stdin)
				if !scanner.Scan() {
					return nil
				}
				if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
					fmt.Fprintln(app.Stdout, "aborted")
					return nil
				}
			}

			newEnvs := make([]config.Environment, 0, len(app.Config.Environments)-1)
			for _, e := range app.Config.Environments {
				if e.Name != name {
					newEnvs = append(newEnvs, e)
				}
			}
			app.Config.Environments = newEnvs

			if err := config.SaveConfig(app.ConfigDir, app.Config); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			// If we just removed the active env, clear state
			if app.State != nil && app.State.ActiveEnvironment == name {
				if err := config.SaveState(app.ConfigDir, &config.State{}); err != nil {
					return fmt.Errorf("clear state: %w", err)
				}
				ui.PrintWarning(app.Stdout, app.Theme, "removed active environment — run `rudder use` to select a new one")
			}

			ui.PrintSuccess(app.Stdout, app.Theme, fmt.Sprintf("removed environment %q", name))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

func newConfigRenameCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Rename an environment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName, newName := args[0], args[1]

			if _, ok := app.Config.FindEnv(oldName); !ok {
				return fmt.Errorf("environment %q not found", oldName)
			}
			if newName != oldName {
				if _, exists := app.Config.FindEnv(newName); exists {
					return fmt.Errorf("environment %q already exists", newName)
				}
			}

			for i := range app.Config.Environments {
				if app.Config.Environments[i].Name == oldName {
					app.Config.Environments[i].Name = newName
					break
				}
			}

			if err := config.SaveConfig(app.ConfigDir, app.Config); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			// Update state if renamed env was active
			if app.State != nil && app.State.ActiveEnvironment == oldName {
				if err := config.SaveState(app.ConfigDir, &config.State{ActiveEnvironment: newName}); err != nil {
					return fmt.Errorf("update state: %w", err)
				}
			}

			ui.PrintSuccess(app.Stdout, app.Theme, fmt.Sprintf("renamed %q → %q", oldName, newName))
			return nil
		},
	}
}

func newConfigListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeEnv := ""
			if app.State != nil {
				activeEnv = app.State.ActiveEnvironment
			}
			return printEnvsTable(cmd, app.Config.Environments, activeEnv, app.Theme, app.NoTUI)
		},
	}
}

func newConfigEditCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open config in $EDITOR",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.EnsureConfigDir(app.ConfigDir); err != nil {
				return err
			}

			configPath := filepath.Join(app.ConfigDir, "config.yaml")

			// Create empty config file if it doesn't exist
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				if err := config.SaveConfig(app.ConfigDir, app.Config); err != nil {
					return fmt.Errorf("create config file: %w", err)
				}
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				editor = "vi"
			}

			c := exec.Command(editor, configPath)
			c.Stdin = os.Stdin
			c.Stdout = app.Stdout
			c.Stderr = app.Stderr
			return c.Run()
		},
	}
}

func newConfigValidateCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate all kubeconfig paths and contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(app.Config.Environments) == 0 {
				ui.PrintWarning(app.Stdout, app.Theme, "no environments to validate")
				return nil
			}

			allValid := true
			for _, env := range app.Config.Environments {
				env := env
				errs := config.ValidateEnvironment(&env)
				if len(errs) == 0 {
					ui.PrintSuccess(app.Stdout, app.Theme, env.Name)
				} else {
					allValid = false
					for _, e := range errs {
						fmt.Fprintf(app.Stderr, "✗ %s.%s: %s\n", env.Name, e.Field, e.Message)
					}
				}
			}

			if !allValid {
				return fmt.Errorf("validation failed for one or more environments")
			}
			return nil
		},
	}
}

// promptNewEnvironment runs an interactive wizard for adding a new environment.
func promptNewEnvironment(app *App) (config.Environment, error) {
	scanner := bufio.NewScanner(os.Stdin)

	prompt := func(label, defaultVal string) string {
		if defaultVal != "" {
			fmt.Fprintf(app.Stdout, "%s [%s]: ", label, defaultVal)
		} else {
			fmt.Fprintf(app.Stdout, "%s: ", label)
		}
		if !scanner.Scan() {
			return defaultVal
		}
		v := strings.TrimSpace(scanner.Text())
		if v == "" {
			return defaultVal
		}
		return v
	}

	var env config.Environment
	env.Name = prompt("Name", "")
	env.Kubeconfig = prompt("Kubeconfig path", "~/.kube/config")
	env.Context = prompt("Context (leave blank for current-context)", "")
	env.Namespace = prompt("Default namespace (leave blank for none)", "")
	env.Description = prompt("Description (optional)", "")

	tagsStr := prompt("Tags (comma-separated, optional)", "")
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			if tag := strings.TrimSpace(t); tag != "" {
				env.Tags = append(env.Tags, tag)
			}
		}
	}

	return env, nil
}
