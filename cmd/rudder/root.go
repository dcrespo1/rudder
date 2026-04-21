package main

import (
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/dcrespo1/rudder/internal/config"
	"github.com/dcrespo1/rudder/internal/kubectl"
	"github.com/dcrespo1/rudder/internal/ui"
)

// App holds all shared application state passed into commands via closure.
type App struct {
	Config      *config.RudderConfig
	State       *config.State
	Theme       ui.Theme
	Log         *slog.Logger
	KubectlPath string
	NoColor     bool
	NoTUI       bool
	ConfigDir   string
	// Stdout and Stderr are the output writers used by all commands.
	// Defaults to os.Stdout/os.Stderr; overridden in tests via cmd.OutOrStdout().
	Stdout io.Writer
	Stderr io.Writer
}

// NewApp returns a zero-value App with output defaulting to os.Stdout/os.Stderr.
func NewApp() *App {
	return &App{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

var rootCmd *cobra.Command

// Execute is the entrypoint called from main.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	app := NewApp()
	rootCmd = newCommandTree(app)
}

// newCommandTree builds and returns the full cobra command tree for the given app.
// Exposed for testing: tests call this directly with a pre-configured App instead
// of relying on the package-level singleton created by init().
func newCommandTree(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:           "rudder",
		Short:         "Multi-cluster kubectl wrapper",
		Long:          "Rudder is a high-performance, multi-cluster kubectl wrapper for managing Kubernetes environments.",
		SilenceUsage:  true, // don't print usage on RunE errors
		SilenceErrors: true, // we print errors ourselves in main.go
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Commands that need no config or kubectl resolution
			name := cmd.Name()
			if name == "version" || name == "completion" || name == "help" {
				app.Stdout = cmd.OutOrStdout()
				app.Stderr = cmd.ErrOrStderr()
				app.Theme = ui.NewTheme()
				app.Log = slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), nil))
				return nil
			}

			// RUDDER_CI implies both NO_COLOR and NO_TUI
			if os.Getenv("RUDDER_CI") == "true" {
				app.NoColor = true
				app.NoTUI = true
			}
			if os.Getenv("RUDDER_NO_COLOR") != "" {
				app.NoColor = true
			}
			if os.Getenv("RUDDER_NO_TUI") != "" {
				app.NoTUI = true
			}

			// Resolve config dir: flag > env var > default
			if app.ConfigDir == "" {
				if envDir := os.Getenv("RUDDER_CONFIG"); envDir != "" {
					app.ConfigDir = envDir
				} else {
					app.ConfigDir = config.DefaultConfigDir()
				}
			}

			// Logger
			level := slog.LevelInfo
			switch os.Getenv("RUDDER_LOG_LEVEL") {
			case "debug":
				level = slog.LevelDebug
			case "warn":
				level = slog.LevelWarn
			case "error":
				level = slog.LevelError
			}
			app.Log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

			// Wire output writers from cobra so tests can capture output
			app.Stdout = cmd.OutOrStdout()
			app.Stderr = cmd.ErrOrStderr()

			// Theme — constructed once here, never inside Update/View
			app.Theme = ui.NewTheme()

			// Load config (returns empty config if file doesn't exist — first-run safe)
			cfg, err := config.LoadConfig(app.ConfigDir)
			if err != nil {
				return err
			}
			app.Config = cfg

			// Load state (returns nil if file doesn't exist)
			state, err := config.LoadState(app.ConfigDir)
			if err != nil {
				return err
			}
			app.State = state

			// Resolve kubectl path (only needed for exec)
			if name == "exec" {
				kubectlPath, err := kubectl.Resolve(cfg.KubectlPath)
				if err != nil {
					return err
				}
				app.KubectlPath = kubectlPath
			}

			return nil
		},
	}

	// Persistent flags
	root.PersistentFlags().StringVar(&app.ConfigDir, "config-dir", "", "override config directory (default: ~/.rudder)")
	root.PersistentFlags().BoolVar(&app.NoColor, "no-color", false, "disable color output")
	root.PersistentFlags().BoolVar(&app.NoTUI, "no-tui", false, "disable interactive TUI")
	root.PersistentFlags().String("log-level", "info", "log verbosity: debug, info, warn, error")

	// Register subcommands
	root.AddCommand(NewInitCmd(app))
	root.AddCommand(NewEnvsCmd(app))
	root.AddCommand(NewUseCmd(app))
	root.AddCommand(NewConfigCmd(app))
	root.AddCommand(NewExecCmd(app))
	root.AddCommand(NewVersionCmd(app))

	return root
}
