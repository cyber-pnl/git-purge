package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jbdanho/git-purge/pkg/analyzer"
)

func newListCmd(base, protect string, keepDays int, verbose bool) *cobra.Command {
	return &cobra.Command{
		Use:           "list",
		Short:         "List branches classified by status",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := runOptions{base: base, protect: protect, keepDays: keepDays, verbose: verbose}
			_, an, engine, err := setup(opts, "")
			if err != nil {
				return err
			}
			analysis, err := engine.Analyze(an)
			if err != nil {
				return fmt.Errorf("analyze: %w", err)
			}
			printClassification(analysis.Branches)
			return nil
		},
	}
}

func printClassification(branches []analyzer.BranchResult) {
	printHeader := func() {
		fmt.Printf("%-20s %s\n", "STATUS", "BRANCH")
	}
	printHeader()
	for _, b := range branches {
		current := ""
		if b.IsCurrent {
			current = " (current)"
		}
		fmt.Printf("%-20s %s%s\n", b.Status.String(), b.Name, current)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "version",
		Short:         "Print version",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("git-purge", Version)
		},
	}
}
