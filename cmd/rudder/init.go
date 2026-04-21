package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dcrespo1/rudder/internal/config"
	"github.com/dcrespo1/rudder/internal/kubeconfig"
	"github.com/dcrespo1/rudder/internal/ui"
)

func NewInitCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Interactive first-time setup wizard",
		Long: `Interactive first-time setup wizard.

Scans for existing kubeconfigs, prompts you to register environments,
and writes ~/.rudder/config.yaml.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.EnsureConfigDir(app.ConfigDir); err != nil {
				return err
			}

			configPath := filepath.Join(app.ConfigDir, "config.yaml")

			// Warn if config already exists
			if _, err := os.Stat(configPath); err == nil {
				fmt.Fprintf(app.Stdout, "Config already exists at %s\n", configPath)
				fmt.Fprint(app.Stdout, "Overwrite? [y/N] ")
				scanner := bufio.NewScanner(os.Stdin)
				if !scanner.Scan() || strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
					fmt.Fprintln(app.Stdout, "Keeping existing config. Use `rudder config add` to add environments.")
					return nil
				}
			}

			ui.PrintBanner(app.Stdout, app.Theme)
			fmt.Fprintln(app.Stdout, "Welcome to Rudder! Let's set up your environments.")
			fmt.Fprintln(app.Stdout)

			// Discover existing kubeconfigs
			discovered := kubeconfig.DiscoverKubeconfigs()
			if len(discovered) > 0 {
				fmt.Fprintln(app.Stdout, "Found the following kubeconfigs:")
				for i, p := range discovered {
					fmt.Fprintf(app.Stdout, "  %d. %s\n", i+1, p)
				}
				fmt.Fprintln(app.Stdout)
			} else {
				fmt.Fprintln(app.Stdout, "No kubeconfigs found in default locations.")
			fmt.Fprintln(app.Stdout)
			}

			scanner := bufio.NewScanner(os.Stdin)
			var environments []config.Environment

			// Offer to register each discovered kubeconfig
			for _, kubeconfigPath := range discovered {
				fmt.Fprintf(app.Stdout, "Register %s? [Y/n] ", kubeconfigPath)
				if !scanner.Scan() {
					break
				}
				answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
				if answer == "n" || answer == "no" {
					continue
				}

				env, err := promptEnvForKubeconfig(scanner, app.Stdout, kubeconfigPath)
				if err != nil {
					fmt.Fprintf(app.Stderr, "skipping %s: %v\n", kubeconfigPath, err)
					continue
				}
				environments = append(environments, env)
				fmt.Fprintln(app.Stdout)
			}

			// Offer to add more manually
			for {
				fmt.Fprint(app.Stdout, "Add another environment manually? [y/N] ")
				if !scanner.Scan() {
					break
				}
				if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
					break
				}
				env, err := promptNewEnvironment(app)
				if err != nil {
					return err
				}
				errs := config.ValidateEnvironment(&env)
				if len(errs) > 0 {
					for _, e := range errs {
						fmt.Fprintf(app.Stderr, "  validation: %s\n", e)
					}
					fmt.Fprintln(app.Stderr, "  Skipping invalid environment.")
					continue
				}
				environments = append(environments, env)
				fmt.Fprintln(app.Stdout)
			}

			cfg := &config.RudderConfig{
				Version:      "1",
				Environments: environments,
			}

			if err := config.SaveConfig(app.ConfigDir, cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			fmt.Fprintln(app.Stdout)
			ui.PrintSuccess(app.Stdout, app.Theme, fmt.Sprintf("config written to %s", configPath))

			if len(environments) > 0 {
				fmt.Fprintln(app.Stdout, "\nRun `rudder use` to select your active environment.")
			} else {
				fmt.Fprintln(app.Stdout, "\nNo environments registered. Run `rudder config add` to add one.")
			}

			return nil
		},
	}
}

// promptEnvForKubeconfig prompts the user for environment details for a discovered kubeconfig.
func promptEnvForKubeconfig(scanner *bufio.Scanner, out io.Writer, kubeconfigPath string) (config.Environment, error) {
	// Suggest a name from the filename
	base := filepath.Base(kubeconfigPath)
	suggestedName := strings.TrimSuffix(strings.TrimSuffix(base, ".yaml"), ".yml")
	if suggestedName == "config" {
		suggestedName = "default"
	}

	prompt := func(label, defaultVal string) string {
		if defaultVal != "" {
			fmt.Fprintf(out, "  %s [%s]: ", label, defaultVal)
		} else {
			fmt.Fprintf(out, "  %s: ", label)
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

	env := config.Environment{
		Kubeconfig: kubeconfigPath,
	}

	env.Name = prompt("Name", suggestedName)
	env.Context = prompt("Context (leave blank for current-context)", "")
	env.Namespace = prompt("Default namespace", "")
	env.Description = prompt("Description", "")

	tagsStr := prompt("Tags (comma-separated)", "")
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			if tag := strings.TrimSpace(t); tag != "" {
				env.Tags = append(env.Tags, tag)
			}
		}
	}

	return env, nil
}
