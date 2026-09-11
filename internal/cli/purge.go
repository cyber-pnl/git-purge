package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cyber-pnl/git-purge/pkg/safety"
	"github.com/cyber-pnl/git-purge/pkg/ui"
)

// runPurge is the entrypoint for the default (purge) command.
func runPurge(opts runOptions) error {
	_, an, engine, err := setup(opts, "")
	if err != nil {
		return err
	}

	analysis, err := engine.Analyze(an)
	if err != nil {
		return fmt.Errorf("analyze: %w", err)
	}

	if opts.verbose {
		fmt.Printf("Analyzed %d branch(es), found %d candidate(s).\n",
			len(analysis.Branches), len(analysis.Candidates))
	}

	if len(analysis.Candidates) == 0 {
		fmt.Println("No stale branches to clean. All clean!")
		return nil
	}

	if opts.dryRun {
		printDryRun(analysis)
		return nil
	}
	if opts.yes && opts.force {
		return deleteAll(engine, analysis)
	}
	return runInteractive(engine, analysis)
}

func printDryRun(analysis safety.Analysis) {
	fmt.Println("Dry run — nothing will be deleted. Candidates:")
	for _, c := range analysis.Candidates {
		fmt.Printf("  %-20s %s\n", c.Status.String(), c.Name)
	}
	fmt.Printf("\n%d branch(es) would be deleted.\n", len(analysis.Candidates))
}

func deleteAll(engine *safety.Engine, analysis safety.Analysis) error {
	kept := 0
	for _, c := range analysis.Candidates {
		if err := engine.DeleteSafe(c.Branch, 0, true); err != nil {
			fmt.Printf("skip %-20s %v\n", c.Name, err)
			kept++
			continue
		}
		fmt.Printf("deleted %s\n", c.Name)
	}
	if kept > 0 {
		return fmt.Errorf("%d branch(es) could not be deleted", kept)
	}
	return nil
}

// runInteractive launches the Bubbletea TUI for branch selection.
func runInteractive(engine *safety.Engine, analysis safety.Analysis) error {
	model := ui.New(engine, analysis.Candidates)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run TUI: %w", err)
	}
	return nil
}
