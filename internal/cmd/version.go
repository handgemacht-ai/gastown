package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/style"
)

// Version information - set at build time via ldflags
var (
	Version   = "0.1.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(style.Bold.Render("gt") + " - Gas Town CLI")
		fmt.Printf("Version:  %s\n", Version)
		fmt.Printf("Commit:   %s\n", style.Dim.Render(GitCommit))
		fmt.Printf("Built:    %s\n", style.Dim.Render(BuildTime))
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
