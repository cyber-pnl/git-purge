package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jbdanho/git-purge/pkg/analyzer"
	giter "github.com/jbdanho/git-purge/pkg/git"
	"github.com/jbdanho/git-purge/pkg/safety"
)

// Version is the build version, overridden at link time.
var Version = "dev"

// Execute runs the root command and exits on error.
func Execute() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		dryRun   bool
		force    bool
		yes      bool
		keepDays int
		protect  string
		verbose  bool
		base     string
	)

	root := &cobra.Command{
		Use:   "git-purge",
		Short: "Safely clean up dead local Git branches",
		Long: "git-purge removes merged, squash-merged, and gone local branches\n" +
			"without ever deleting unmerged work.",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPurge(runOptions{
				dryRun:   dryRun,
				force:    force,
				yes:      yes,
				keepDays: keepDays,
				protect:  protect,
				verbose:  verbose,
				base:     base,
			})
		},
	}

	root.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "simulate without deleting anything")
	root.PersistentFlags().IntVar(&keepDays, "keep-days", 0, "keep branches modified within N days")
	root.PersistentFlags().StringVar(&protect, "protect", "", "comma-separated protected branch patterns")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose output")
	root.PersistentFlags().StringVar(&base, "base", "", "base branch for merge detection (default: current main/master)")
	root.PersistentFlags().BoolVar(&force, "force", false, "skip confirmation (non-interactive)")
	root.PersistentFlags().BoolVar(&yes, "yes", false, "auto-confirm all prompts")

	root.AddCommand(newListCmd(base, protect, keepDays, verbose))
	root.AddCommand(newVersionCmd())
	return root
}

type runOptions struct {
	dryRun   bool
	force    bool
	yes      bool
	keepDays int
	protect  string
	verbose  bool
	base     string
}

// setup returns a configured git client, analyzer, and safety engine.
func setup(opts runOptions, inputDir string) (*giter.Client, *analyzer.Analyzer, *safety.Engine, error) {
	dir := inputDir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("get working directory: %w", err)
		}
	}
	g := giter.NewClient(dir)

	inside, err := g.IsInsideWorkTree()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("not a git repository: %w", err)
	}
	if !inside {
		return nil, nil, nil, fmt.Errorf("not inside a git working tree (%s)", dir)
	}

	base := opts.base
	if base == "" {
		// Check for an empty repository (no commits yet) first.
		hasCommits, err := g.HasCommits()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("inspect repository: %w", err)
		}
		if !hasCommits {
			return nil, nil, nil, fmt.Errorf("repository has no commits yet; nothing to purge")
		}
		current, err := g.CurrentBranch()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("detect current branch: %w", err)
		}
		base = current
	}

	engine := safety.New(g, safety.Options{
		KeepDays:       opts.keepDays,
		CustomPatterns: parsePatterns(opts.protect),
		BaseBranch:     base,
	})
	an := analyzer.New(g, base)
	return g, an, engine, nil
}

func parsePatterns(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
