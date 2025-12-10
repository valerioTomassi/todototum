package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// these will be overridden at build time using -ldflags
	version = "0.0.1"
	commit  = "dev"
	date    = "unknown"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

// versionCmd prints todototum version information.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show todototum version information",
	Long:  `Displays the current version, git commit, and build date for todototum.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("todototum %s (commit %s, built %s)\n", version, commit, date)
	},
}
