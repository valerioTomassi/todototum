package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command executed when no subcommand is provided.
var rootCmd = &cobra.Command{
	Use:   "todototum",
	Short: "See the whole TODO picture",
	Long: `todototum scans your codebase for TODO, FIXME, BUG and NOTE comments
across any programming language. It outputs clear summaries to the terminal
or generates reports for later analysis.`,
	// no Run function here; 'scan' will handle execution
}

// Execute runs the CLI. Called from main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Any global (persistent) flags can go here later if needed.
}
