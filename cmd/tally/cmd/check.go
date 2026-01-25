package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/tinovyatkin/tally/internal/dockerfile"
	"github.com/tinovyatkin/tally/internal/lint"
)

func checkCommand() *cli.Command {
	return &cli.Command{
		Name:      "check",
		Usage:     "Check Dockerfile(s) for issues",
		ArgsUsage: "[DOCKERFILE...]",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "max-lines",
				Aliases: []string{"l"},
				Usage:   "Maximum number of lines allowed in a Dockerfile (0 = unlimited)",
				Value:   0,
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Output format: text, json",
				Value:   "text",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			maxLines := cmd.Int("max-lines")
			format := cmd.String("format")
			files := cmd.Args().Slice()

			if len(files) == 0 {
				// Default to Dockerfile in current directory
				files = []string{"Dockerfile"}
			}

			var allResults []lint.FileResult

			for _, file := range files {
				// Parse the Dockerfile
				parseResult, err := dockerfile.ParseFile(ctx, file)
				if err != nil {
					return fmt.Errorf("failed to parse %s: %w", file, err)
				}

				// Run linting rules
				var issues []lint.Issue

				// Check max-lines rule if configured
				if maxLines > 0 {
					if issue := lint.CheckMaxLines(parseResult, maxLines); issue != nil {
						issues = append(issues, *issue)
					}
				}

				allResults = append(allResults, lint.FileResult{
					File:   file,
					Lines:  parseResult.TotalLines,
					Issues: issues,
				})
			}

			// Output results
			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(allResults); err != nil {
					return fmt.Errorf("failed to encode JSON: %w", err)
				}
			default:
				hasIssues := false
				for _, result := range allResults {
					if len(result.Issues) > 0 {
						hasIssues = true
						for _, issue := range result.Issues {
							fmt.Printf("%s:%d: %s (%s)\n", result.File, issue.Line, issue.Message, issue.Rule)
						}
					}
				}
				if hasIssues {
					os.Exit(1)
				}
			}

			return nil
		},
	}
}
