package safety

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cyber-pnl/git-purge/pkg/analyzer"
	giter "github.com/cyber-pnl/git-purge/pkg/git"
)

// DefaultProtected are branches never deletable by default.
var DefaultProtected = []string{"main", "master", "dev"}

// Options configures the safety engine.
type Options struct {
	// KeepDays excludes branches modified within this many days (0 = disable).
	KeepDays int
	// CustomPatterns are additional protected branch patterns (Glob style).
	CustomPatterns []string
	// BaseBranch is the reference branch for classification.
	BaseBranch string
	// Force skips the protection check for non-protected branches (unused; kept explicit).
	Force bool
}

// Engine enforces safety rules before any deletion.
type Engine struct {
	git    *giter.Client
	opts   Options
	logDir string
}

// New creates a safety engine rooted at the given git directory.
func New(g *giter.Client, opts Options) *Engine {
	return &Engine{
		git:    g,
		opts:   opts,
		logDir: filepath.Join(g.Dir(), ".git-purge"),
	}
}

// SetLogDir overrides where deletion logs are written (for tests).
func (e *Engine) SetLogDir(dir string) {
	e.logDir = dir
}

// IsProtected returns true if the branch must never be deleted.
func (e *Engine) IsProtected(branch string, isCurrent bool) bool {
	if isCurrent {
		return true
	}
	for _, p := range DefaultProtected {
		if branch == p {
			return true
		}
	}
	for _, p := range e.opts.CustomPatterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if matched, _ := filepath.Match(p, branch); matched {
			return true
		}
	}
	return false
}

// RecentlyModified returns true if the branch was committed within keep-days.
// keepDays == 0 disables the check (always false).
func (e *Engine) RecentlyModified(branch giter.Branch, keepDays int) (bool, error) {
	if keepDays <= 0 {
		return false, nil
	}
	if branch.LastCommitDate == "" {
		return false, nil
	}
	t, err := time.Parse(time.RFC3339, branch.LastCommitDate)
	if err != nil {
		return false, fmt.Errorf("parse commit date %q: %w", branch.LastCommitDate, err)
	}
	cutoff := time.Now().AddDate(0, 0, -keepDays)
	return t.After(cutoff), nil
}

// CanDelete returns the reason a branch cannot be deleted, or "" if it can.
// It runs the full protection chain regardless of classification.
func (e *Engine) CanDelete(branch giter.Branch, keepDays int) (string, error) {
	if e.IsProtected(branch.Name, branch.IsCurrent) {
		return fmt.Sprintf("protected branch %q", branch.Name), nil
	}
	recent, err := e.RecentlyModified(branch, keepDays)
	if err != nil {
		return "", err
	}
	if recent {
		return fmt.Sprintf("modified within the last %d days", keepDays), nil
	}
	return "", nil
}

// DryRun simulates a deletion without touching git.
func (e *Engine) DryRun(branch giter.Branch) error {
	return e.recordLog(branch.Name, branch.Hash, "dry-run")
}

// LogRecording captures a deletion record.
type LogRecording struct {
	Branch    string
	SHA       string
	Mode      string
	Timestamp time.Time
}

// recordLog writes the branch SHA to the local log before deletion.
func (e *Engine) recordLog(branch, sha, mode string) error {
	if err := os.MkdirAll(e.logDir, 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	host := e.logFileFor(branch)
	entry := fmt.Sprintf("%s %s %s %s\n", time.Now().Format(time.RFC3339), mode, branch, sha)
	f, err := os.OpenFile(host, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("write log: %w", err)
	}
	return nil
}

func (e *Engine) logFileFor(branch string) string {
	safe := strings.NewReplacer("/", "_", "\\", "_", "*", "_").Replace(branch)
	return filepath.Join(e.logDir, safe+".log")
}

// CurrentLog returns the log contents for a branch (test helper).
func (e *Engine) CurrentLog(branch string) string {
	data, _ := os.ReadFile(e.logFileFor(branch))
	return string(data)
}

// DeleteSafe records the SHA then deletes the branch.
// It refuses (without side effects) if the branch is protected or current.
func (e *Engine) DeleteSafe(branch giter.Branch, keepDays int, force bool) error {
	reason, err := e.CanDelete(branch, keepDays)
	if err != nil {
		return err
	}
	if reason != "" {
		return fmt.Errorf("refusing to delete: %s", reason)
	}
	if err := e.recordLog(branch.Name, branch.Hash, "deleted"); err != nil {
		return fmt.Errorf("record log before delete: %w", err)
	}
	return e.git.DeleteBranch(branch.Name, force)
}

// Analysis wraps the analyzer to also apply safety filtering.
type Analysis struct {
	Branches   []analyzer.BranchResult
	Candidates []analyzer.BranchResult
}

// Analyze runs classification and safety filtering together.
func (e *Engine) Analyze(an *analyzer.Analyzer) (Analysis, error) {
	results, err := an.AnalyzeAll()
	if err != nil {
		return Analysis{}, err
	}

	var candidates []analyzer.BranchResult
	for i := range results {
		r := &results[i]
		// Hard filter: never consider protected/current branches.
		if r.Status == analyzer.StatusActiveUnmerged {
			continue
		}
		if reason, err := e.CanDelete(r.Branch, e.opts.KeepDays); err != nil {
			return Analysis{}, err
		} else if reason != "" {
			// Mark as protected in results for display.
			r.Status = analyzer.StatusProtected
		}
		if r.Status == analyzer.StatusMerged ||
			r.Status == analyzer.StatusSquashMerged ||
			r.Status == analyzer.StatusGone {
			candidates = append(candidates, *r)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Name < candidates[j].Name
	})

	return Analysis{Branches: results, Candidates: candidates}, nil
}
