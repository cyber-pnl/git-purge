package analyzer

import (
	"fmt"
	"strings"

	giter "github.com/jbdanho/git-purge/pkg/git"
)

// Status represents the classification of a branch.
type Status int

const (
	// StatusUnknown is the initial state before classification.
	StatusUnknown Status = iota
	// StatusMerged means the branch is fully merged into base.
	StatusMerged
	// StatusSquashMerged means the branch was squash-merged into base.
	StatusSquashMerged
	// StatusGone means the upstream tracking branch was deleted.
	StatusGone
	// StatusProtected means the branch is protected and never deletable.
	StatusProtected
	// StatusActiveUnmerged means the branch has unmerged work and must be kept.
	StatusActiveUnmerged
)

func (s Status) String() string {
	switch s {
	case StatusMerged:
		return "MERGED"
	case StatusSquashMerged:
		return "SQUASH_MERGED"
	case StatusGone:
		return "GONE"
	case StatusProtected:
		return "PROTECTED"
	case StatusActiveUnmerged:
		return "ACTIVE_UNMERGED"
	default:
		return "UNKNOWN"
	}
}

// BranchResult combines branch info with its classification.
type BranchResult struct {
	giter.Branch
	Status Status
}

// Analyzer classifies branches against a base branch.
type Analyzer struct {
	git              *giter.Client
	baseBranch       string
	defaultProtected []string
}

// New creates a new Analyzer for the given base branch.
func New(g *giter.Client, baseBranch string) *Analyzer {
	if baseBranch == "" {
		baseBranch = "main"
	}
	return &Analyzer{
		git:              g,
		baseBranch:       baseBranch,
		defaultProtected: []string{"main", "master", "dev"},
	}
}

// Classify determines the status of a single branch.
func (a *Analyzer) Classify(branch giter.Branch) (Status, error) {
	// Protected branches: current branch, default protected names.
	if branch.IsCurrent || a.isProtectedByDefault(branch.Name) {
		return StatusProtected, nil
	}

	// Gone upstream.
	if branch.UpstreamGone {
		return StatusGone, nil
	}

	// Merged (classic fast-forward or merge commit).
	merged, err := a.git.IsMerged(branch.Name, a.baseBranch)
	if err != nil {
		// On doubt, treat as not merged — safer.
		return StatusActiveUnmerged, fmt.Errorf("check merge status: %w", err)
	}
	if merged {
		return StatusMerged, nil
	}

	// Squash-merged detection.
	squashed, err := a.isSquashMerged(branch.Name)
	if err != nil {
		return StatusActiveUnmerged, err
	}
	if squashed {
		return StatusSquashMerged, nil
	}

	// Everything else needs to be kept.
	if err := a.verifyBaseBranch(); err != nil {
		return StatusActiveUnmerged, fmt.Errorf("verify base branch: %w", err)
	}
	return StatusActiveUnmerged, nil
}

func (a *Analyzer) isProtectedByDefault(name string) bool {
	for _, p := range a.defaultProtected {
		if name == p {
			return true
		}
	}
	return false
}

func (a *Analyzer) verifyBaseBranch() error {
	branches, err := a.git.ListLocalBranches()
	if err != nil {
		return err
	}
	for _, b := range branches {
		if b.Name == a.baseBranch {
			return nil
		}
	}
	return fmt.Errorf("base branch %q does not exist locally", a.baseBranch)
}

// isSquashMerged detects if a branch's changes were squash-merged into base.
//
// Algorithm:
//  1. Find merge-base between base and branch.
//  2. Compute the tree of branch tip.
//  3. Walk commits on base after merge-base; if any commit has the same
//     tree as the branch tip, the branch was squash-merged.
func (a *Analyzer) isSquashMerged(branch string) (bool, error) {
	mergeBase, err := a.git.MergeBase(a.baseBranch, branch)
	if err != nil {
		return false, fmt.Errorf("merge-base: %w", err)
	}

	// Tree of branch tip.
	branchTree, err := a.git.TreeHash(branch)
	if err != nil {
		return false, fmt.Errorf("branch tree: %w", err)
	}

	// Enumerate commits on base since merge-base.
	commits, err := a.git.LogCommits(mergeBase, a.baseBranch)
	if err != nil {
		return false, fmt.Errorf("list candidate commits: %w", err)
	}

	for _, commit := range commits {
		tree, err := a.git.TreeHash(commit)
		if err != nil {
			continue
		}
		if tree == branchTree {
			return true, nil
		}
	}

	// Also handle the edge case where the branch has no commits beyond
	// merge-base but the trees still differ (impossible in practice).
	if mergeBase != "" {
		baseTree, err := a.git.TreeHash(mergeBase)
		if err == nil && baseTree == branchTree {
			return true, nil
		}
	}

	return false, nil
}

// AnalyzeAll classifies every local branch and returns the results.
// Protected branches (by default list or the current HEAD) are filtered.
func (a *Analyzer) AnalyzeAll() ([]BranchResult, error) {
	branches, err := a.git.ListLocalBranches()
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	results := make([]BranchResult, 0, len(branches))
	for _, b := range branches {
		status, err := a.Classify(b)
		if err != nil {
			// On any error, be safe: active unmerged.
			status = StatusActiveUnmerged
		}
		results = append(results, BranchResult{Branch: b, Status: status})
	}
	return results, nil
}

// Candidates returns only branches eligible for deletion.
func (a *Analyzer) Candidates() ([]BranchResult, error) {
	results, err := a.AnalyzeAll()
	if err != nil {
		return nil, err
	}
	var candidates []BranchResult
	for _, r := range results {
		switch r.Status {
		case StatusMerged, StatusSquashMerged, StatusGone:
			candidates = append(candidates, r)
		}
	}
	return candidates, nil
}

// HasUpstream returns true if the branch has a remote tracking ref.
func HasUpstream(branch giter.Branch) bool {
	return strings.TrimSpace(branch.Upstream) != ""
}
