// Package cmd provides CLI commands for the gt tool.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gt",
	Short: "Gas Town - Multi-agent workspace manager",
	Long: `Gas Town (gt) manages multi-agent workspaces called rigs.

It coordinates agent spawning, work distribution, and communication
across distributed teams of AI agents working on shared codebases.`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
}
