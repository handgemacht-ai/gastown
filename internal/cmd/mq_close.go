package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/style"
)

var (
	mqCloseReason      string
	mqCloseCloseSource bool
)

var mqCloseCmd = &cobra.Command{
	Use:   "close <rig> <mr-id-or-branch>",
	Short: "Close a merge request",
	Long: `Close a merge request with a reason.

Closes the MR bead and optionally its source issue.
Typically used after a successful merge to record the outcome in beads.

Examples:
  gt mq close greenplace gp-mr-abc123
  gt mq close greenplace gp-mr-abc123 --reason=merged
  gt mq close greenplace gp-mr-abc123 --reason=superseded --no-close-source`,
	Args: cobra.ExactArgs(2),
	RunE: runMQClose,
}

func runMQClose(cmd *cobra.Command, args []string) error {
	rigName := args[0]
	mrIDOrBranch := args[1]

	mgr, _, _, err := getRefineryManager(rigName)
	if err != nil {
		return err
	}

	result, err := mgr.CloseMR(mrIDOrBranch, mqCloseReason, mqCloseCloseSource)
	if err != nil {
		return fmt.Errorf("closing MR: %w", err)
	}

	fmt.Printf("%s Closed: %s\n", style.Bold.Render("✓"), result.ID)
	fmt.Printf("  Branch: %s\n", result.Branch)
	fmt.Printf("  Worker: %s\n", result.Worker)
	fmt.Printf("  Reason: %s\n", mqCloseReason)

	if mqCloseCloseSource && result.IssueID != "" {
		fmt.Printf("  Issue:  %s %s\n", result.IssueID, style.Dim.Render("(closed)"))
	} else if result.IssueID != "" {
		fmt.Printf("  Issue:  %s %s\n", result.IssueID, style.Dim.Render("(not closed)"))
	}

	return nil
}
