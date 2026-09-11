package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jbdanho/git-purge/pkg/analyzer"
	giter "github.com/jbdanho/git-purge/pkg/git"
	"github.com/jbdanho/git-purge/pkg/safety"
)

func setupRepo(t *testing.T) (client *giter.Client, dir string) {
	t.Helper()
	dir = t.TempDir()
	cmd := exec.Command("git", "init", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %s: %v", string(out), err)
	}
	_ = exec.Command("git", append([]string{"-C", dir}, "config", "user.email", "test@test.com")...).Run()
	_ = exec.Command("git", append([]string{"-C", dir}, "config", "user.name", "Test")...).Run()
	client = giter.NewClient(dir)
	return client, dir
}

func run(dir string, args ...string) {
	_ = exec.Command("git", append([]string{"-C", dir}, args...)...).Run()
}

func commit(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run(dir, "add", name)
	run(dir, "commit", "-m", "add "+name)
}

func defaultBase(t *testing.T, c *giter.Client) string {
	t.Helper()
	branches, err := c.ListLocalBranches()
	if err != nil || len(branches) == 0 {
		t.Fatal("no branches")
	}
	return branches[0].Name
}

func branchExists(c *giter.Client, name string) bool {
	branches, err := c.ListLocalBranches()
	if err != nil {
		return false
	}
	for _, b := range branches {
		if b.Name == name {
			return true
		}
	}
	return false
}

func TestIntegration_FullPipeline(t *testing.T) {
	c, dir := setupRepo(t)

	// Base commit on default branch.
	commit(t, dir, "base.txt", "initial content")
	base := defaultBase(t, c)

	// --- Merged branch ---
	run(dir, "checkout", "-b", "feature/merged")
	commit(t, dir, "merged.txt", "merged work")
	run(dir, "checkout", base)
	run(dir, "merge", "feature/merged", "--no-ff", "-m", "merge feature/merged")

	// --- Squash-merged branch ---
	run(dir, "checkout", "-b", "feature/squash")
	commit(t, dir, "squash_a.txt", "squash work 1")
	commit(t, dir, "squash_b.txt", "squash work 2")
	run(dir, "checkout", base)
	run(dir, "merge", "--squash", "feature/squash")
	run(dir, "commit", "-m", "squash merge feature/squash")

	// --- Active (unmerged) branch ---
	run(dir, "checkout", "-b", "feature/active")
	commit(t, dir, "active.txt", "active work")
	run(dir, "checkout", base)

	// --- Back to base for analysis ---
	run(dir, "checkout", base)

	// Run Analyze + DeleteSafe pipeline through the safety engine.
	an := analyzer.New(c, base)
	e := safety.New(c, safety.Options{BaseBranch: base})
	e.SetLogDir(filepath.Join(dir, ".git-purge"))

	analysis, err := e.Analyze(an)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	// --- Verify classification ---

	// Merged branch should be a candidate.
	var foundMerged bool
	for _, r := range analysis.Candidates {
		if r.Name == "feature/merged" {
			foundMerged = true
			if r.Status != analyzer.StatusMerged {
				t.Errorf("feature/merged: want MERGED, got %s", r.Status)
			}
		}
	}
	if !foundMerged {
		t.Error("feature/merged should be a candidate")
	}

	// Squash-merged branch should be a candidate.
	var foundSquash bool
	for _, r := range analysis.Candidates {
		if r.Name == "feature/squash" {
			foundSquash = true
			if r.Status != analyzer.StatusSquashMerged {
				t.Errorf("feature/squash: want SQUASH_MERGED, got %s", r.Status)
			}
		}
	}
	if !foundSquash {
		t.Error("feature/squash should be a candidate")
	}

	// Active branch should NOT be a candidate.
	for _, r := range analysis.Candidates {
		if r.Name == "feature/active" {
			t.Error("feature/active should not be a candidate")
		}
	}

	// Protected branch (main/master) should NOT be a candidate.
	for _, r := range analysis.Candidates {
		if r.Name == "main" || r.Name == "master" {
			t.Errorf("%s should not be a candidate", r.Name)
		}
	}

	// --- DeleteSafe on all candidates ---
	// Use force=true because squash-merged branches aren't "fully merged" in
	// git's own sense — our analyzer already confirmed safety.
	for _, cand := range analysis.Candidates {
		if err := e.DeleteSafe(cand.Branch, 0, true); err != nil {
			t.Fatalf("DeleteSafe(%s): %v", cand.Name, err)
		}
	}

	// --- Verify branches are actually deleted ---
	if branchExists(c, "feature/merged") {
		t.Error("feature/merged should have been deleted")
	}
	if branchExists(c, "feature/squash") {
		t.Error("feature/squash should have been deleted")
	}

	// Active branch must still exist.
	if !branchExists(c, "feature/active") {
		t.Error("feature/active should still exist")
	}

	// Default branch must still exist.
	if !branchExists(c, base) {
		t.Errorf("default branch %s should still exist", base)
	}

	// --- Verify log files contain SHA records ---
	mergedLog := e.CurrentLog("feature/merged")
	if !strings.Contains(mergedLog, "deleted") {
		t.Errorf("feature/merged log should contain 'deleted', got: %q", mergedLog)
	}
	// Log should also contain a SHA (40-char hex).
	if len(mergedLog) < 40 {
		t.Errorf("feature/merged log should contain SHA, got: %q", mergedLog)
	}

	squashLog := e.CurrentLog("feature/squash")
	if !strings.Contains(squashLog, "deleted") {
		t.Errorf("feature/squash log should contain 'deleted', got: %q", squashLog)
	}
	if len(squashLog) < 40 {
		t.Errorf("feature/squash log should contain SHA, got: %q", squashLog)
	}
}
