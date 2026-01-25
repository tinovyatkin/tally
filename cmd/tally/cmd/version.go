package cmd

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/tinovyatkin/tally/internal/version"
)

func versionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Print version information",
		Action: func(_ context.Context, _ *cli.Command) error {
			fmt.Printf("tally version %s\n", version.Version())
			return nil
		},
	}
}
