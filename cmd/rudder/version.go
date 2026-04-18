package main

import (
	"github.com/spf13/cobra"

	rudder "gitlab.com/dcresp0/rudder/pkg/rudder"
)

func NewVersionCmd(app *App) *cobra.Command {
	var short bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			if short {
				cmd.Println(rudder.Version)
			} else {
				cmd.Println(rudder.BuildInfo())
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&short, "short", false, "print only the version string")
	return cmd
}
