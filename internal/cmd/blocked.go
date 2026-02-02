package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/constants"
	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/rig"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/workspace"
)

var blockedJSON bool
var blockedRig string

var blockedCmd = &cobra.Command{
	Use:     "blocked",
	GroupID: GroupWork,
	Short:   "Show blocked work across town",
	Long: `Display all blocked work items across the town and all rigs.

Aggregates blocked issues from:
- Town beads (hq-* items: convoys, cross-rig coordination)
- Each rig's beads (project-level issues, MRs)

Blocked items have unresolved dependencies preventing them from being worked.
Results are sorted by priority (highest first) then by source.

Examples:
  gt blocked              # Show all blocked work
  gt blocked --json       # Output as JSON
  gt blocked --rig=gastown  # Show only one rig`,
	RunE: runBlocked,
}

func init() {
	blockedCmd.Flags().BoolVar(&blockedJSON, "json", false, "Output as JSON")
	blockedCmd.Flags().StringVar(&blockedRig, "rig", "", "Filter to a specific rig")
	rootCmd.AddCommand(blockedCmd)
}

// BlockedSource represents blocked items from a single source (town or rig).
type BlockedSource struct {
	Name   string         `json:"name"`
	Issues []*beads.Issue `json:"issues"`
	Error  string         `json:"error,omitempty"`
}

// BlockedResult is the aggregated result of gt blocked.
type BlockedResult struct {
	Sources  []BlockedSource `json:"sources"`
	Summary  BlockedSummary  `json:"summary"`
	TownRoot string          `json:"town_root,omitempty"`
}

// BlockedSummary provides counts for the blocked report.
type BlockedSummary struct {
	Total    int            `json:"total"`
	BySource map[string]int `json:"by_source"`
	P0Count  int            `json:"p0_count"`
	P1Count  int            `json:"p1_count"`
	P2Count  int            `json:"p2_count"`
	P3Count  int            `json:"p3_count"`
	P4Count  int            `json:"p4_count"`
}

func runBlocked(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	rigsConfigPath := constants.MayorRigsPath(townRoot)
	rigsConfig, err := config.LoadRigsConfig(rigsConfigPath)
	if err != nil {
		rigsConfig = &config.RigsConfig{Rigs: make(map[string]config.RigEntry)}
	}

	g := git.NewGit(townRoot)
	mgr := rig.NewManager(townRoot, rigsConfig, g)
	rigs, err := mgr.DiscoverRigs()
	if err != nil {
		return fmt.Errorf("discovering rigs: %w", err)
	}

	if blockedRig != "" {
		var filtered []*rig.Rig
		for _, r := range rigs {
			if r.Name == blockedRig {
				filtered = append(filtered, r)
				break
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("rig not found: %s", blockedRig)
		}
		rigs = filtered
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	sources := make([]BlockedSource, 0, len(rigs)+1)

	if blockedRig == "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			townBeadsPath := beads.GetTownBeadsPath(townRoot)
			townBeads := beads.New(townBeadsPath)
			issues, err := townBeads.Blocked()

			mu.Lock()
			defer mu.Unlock()
			src := BlockedSource{Name: "town"}
			if err != nil {
				src.Error = err.Error()
			} else {
				formulaNames := getFormulaNames(townBeadsPath)
				filtered := filterFormulaScaffolds(issues, formulaNames)
				wispIDs := getWispIDs(townBeadsPath)
				filtered = filterWisps(filtered, wispIDs)
				src.Issues = filterWispsByID(filtered)
			}
			sources = append(sources, src)
		}()
	}

	for _, r := range rigs {
		wg.Add(1)
		go func(r *rig.Rig) {
			defer wg.Done()
			rigBeads := beads.New(r.BeadsPath())
			issues, err := rigBeads.Blocked()

			mu.Lock()
			defer mu.Unlock()
			src := BlockedSource{Name: r.Name}
			if err != nil {
				src.Error = err.Error()
			} else {
				formulaNames := getFormulaNames(r.BeadsPath())
				filtered := filterFormulaScaffolds(issues, formulaNames)
				wispIDs := getWispIDs(r.BeadsPath())
				filtered = filterWisps(filtered, wispIDs)
				src.Issues = filterWispsByID(filtered)
			}
			sources = append(sources, src)
		}(r)
	}

	wg.Wait()

	sort.Slice(sources, func(i, j int) bool {
		if sources[i].Name == "town" {
			return true
		}
		if sources[j].Name == "town" {
			return false
		}
		return sources[i].Name < sources[j].Name
	})

	for i := range sources {
		sort.Slice(sources[i].Issues, func(a, b int) bool {
			return sources[i].Issues[a].Priority < sources[i].Issues[b].Priority
		})
	}

	summary := BlockedSummary{
		BySource: make(map[string]int),
	}
	for _, src := range sources {
		count := len(src.Issues)
		summary.Total += count
		summary.BySource[src.Name] = count
		for _, issue := range src.Issues {
			switch issue.Priority {
			case 0:
				summary.P0Count++
			case 1:
				summary.P1Count++
			case 2:
				summary.P2Count++
			case 3:
				summary.P3Count++
			case 4:
				summary.P4Count++
			}
		}
	}

	result := BlockedResult{
		Sources:  sources,
		Summary:  summary,
		TownRoot: townRoot,
	}

	if blockedJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	return printBlockedHuman(result)
}

func printBlockedHuman(result BlockedResult) error {
	if result.Summary.Total == 0 {
		fmt.Println("No blocked work across town.")
		return nil
	}

	fmt.Printf("%s Blocked work across town:\n\n", style.Bold.Render("\U0001F6AB"))

	for _, src := range result.Sources {
		if src.Error != "" {
			fmt.Printf("%s %s\n", style.Dim.Render(src.Name+"/"), style.Warning.Render("(error: "+src.Error+")"))
			continue
		}

		count := len(src.Issues)
		if count == 0 {
			continue
		}

		itemWord := "items"
		if count == 1 {
			itemWord = "item"
		}
		fmt.Printf("%s (%d %s)\n", style.Bold.Render(src.Name+"/"), count, itemWord)
		for _, issue := range src.Issues {
			priorityStr := fmt.Sprintf("P%d", issue.Priority)
			var priorityStyled string
			switch issue.Priority {
			case 0:
				priorityStyled = style.Error.Render(priorityStr)
			case 1:
				priorityStyled = style.Error.Render(priorityStr)
			case 2:
				priorityStyled = style.Warning.Render(priorityStr)
			default:
				priorityStyled = style.Dim.Render(priorityStr)
			}

			title := issue.Title
			if len(title) > 60 {
				title = title[:57] + "..."
			}

			blockedByStr := ""
			if len(issue.BlockedBy) > 0 {
				blockedByStr = " " + style.Dim.Render("(blocked by: "+strings.Join(issue.BlockedBy, ", ")+")")
			}

			fmt.Printf("  [%s] %s %s%s\n", priorityStyled, style.Dim.Render(issue.ID), title, blockedByStr)
		}
		fmt.Println()
	}

	parts := []string{}
	if result.Summary.P0Count > 0 {
		parts = append(parts, fmt.Sprintf("%d P0", result.Summary.P0Count))
	}
	if result.Summary.P1Count > 0 {
		parts = append(parts, fmt.Sprintf("%d P1", result.Summary.P1Count))
	}
	if result.Summary.P2Count > 0 {
		parts = append(parts, fmt.Sprintf("%d P2", result.Summary.P2Count))
	}
	if result.Summary.P3Count > 0 {
		parts = append(parts, fmt.Sprintf("%d P3", result.Summary.P3Count))
	}
	if result.Summary.P4Count > 0 {
		parts = append(parts, fmt.Sprintf("%d P4", result.Summary.P4Count))
	}

	if len(parts) > 0 {
		fmt.Printf("Total: %d items blocked (%s)\n", result.Summary.Total, strings.Join(parts, ", "))
	} else {
		fmt.Printf("Total: %d items blocked\n", result.Summary.Total)
	}

	return nil
}

// filterWispsByID removes issues whose ID contains "-wisp-".
// This is a defense-in-depth filter for bd blocked which may not filter wisps server-side.
func filterWispsByID(issues []*beads.Issue) []*beads.Issue {
	filtered := make([]*beads.Issue, 0, len(issues))
	for _, issue := range issues {
		if !strings.Contains(issue.ID, "-wisp-") {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}
