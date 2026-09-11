package safety

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
"github.com/cyber-pnl/git-purge/pkg/analyzer"

	giter "github.com/cyber-pnl/git-purge/pkg/git"
)

func setupRepo(t *testing.T) (client *giter.Client, dir string) {
	t.Helper()
	dir = t.TempDir()
	cmd := exec.Command("git", "init", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %s: %v", string(out), err)
	}
	run(dir, "config", "user.email", "test@test.com")
	run(dir, "config", "user.name", "Test")
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

func TestIsProtected(t *testing.T) {
	c, _ := setupRepo(t)
	e := New(c, Options{CustomPatterns: []string{"release/*", "staging"}})

	cases := []struct {
		branch    string
		isCurrent bool
		want      bool
	}{
		{"main", false, true},
		{"master", false, true},
		{"dev", false, true},
		{"HEAD-branch", true, true},
		{"release/1.0", false, true},
		{"release/x/y", false, false},
		{"staging", false, true},
		{"feature/login", false, false},
		{"chore", false, false},
		{"test with space", false, false},
	}
	for _, tc := range cases {
		if got := e.IsProtected(tc.branch, tc.isCurrent); got != tc.want {
			t.Errorf("IsProtected(%q, %v) = %v, want %v", tc.branch, tc.isCurrent, got, tc.want)
		}
	}
}

func TestRecentlyModified(t *testing.T) {
	c, _ := setupRepo(t)
	e := New(c, Options{KeepDays: 14})

	branch := giter.Branch{LastCommitDate: "2026-09-01T10:00:00+02:00"}
	recent, err := e.RecentlyModified(branch, 14)
	if err != nil {
		t.Fatalf("RecentlyModified: %v", err)
	}

	old := giter.Branch{LastCommitDate: "2024-01-01T10:00:00+02:00"}
	notRecent, err := e.RecentlyModified(old, 14)
	if err != nil {
		t.Fatalf("RecentlyModified(old): %v", err)
	}

	disabled, err := e.RecentlyModified(branch, 0)
	if err != nil {
		t.Fatalf("RecentlyModified(disabled): %v", err)
	}

	if !recent {
		t.Error("expected recent branch to be flagged")
	}
	if notRecent {
		t.Error("expected old branch not to be flagged")
	}
	if disabled {
		t.Error("expected keep-days=0 to disable the check")
	}
}

func TestCanDelete(t *testing.T) {
	c, _ := setupRepo(t)
	e := New(c, Options{CustomPatterns: []string{"release/*"}})

	// Protected.
	reason, err := e.CanDelete(giter.Branch{Name: "main"}, 0)
	if err != nil {
		t.Fatalf("CanDelete: %v", err)
	}
	if reason == "" {
		t.Error("expected protection reason for main")
	}

	// Current branch.
	reason, err = e.CanDelete(giter.Branch{Name: "current", IsCurrent: true}, 0)
	if err != nil {
		t.Fatalf("CanDelete: %v", err)
	}
	if reason == "" {
		t.Error("expected protection reason for current branch")
	}

	// Custom pattern.
	reason, _ = e.CanDelete(giter.Branch{Name: "release/1.0"}, 0)
	if reason == "" {
		t.Error("expected protection reason for release/*")
	}

	// Deletable.
	reason, err = e.CanDelete(giter.Branch{Name: "feature/x"}, 0)
	if err != nil {
		t.Fatalf("CanDelete: %v", err)
	}
	if reason != "" {
		t.Errorf("expected no reason, got %q", reason)
	}
}

func TestDryRunLogs(t *testing.T) {
	c, dir := setupRepo(t)
	e := New(c, Options{})
	e.SetLogDir(filepath.Join(dir, ".git-purge"))

	err := e.DryRun(giter.Branch{Name: "feature/foo", Hash: "abc123"})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	log := e.CurrentLog("feature/foo")
	if !strings.Contains(log, "feature/foo") || !strings.Contains(log, "abc123") || !strings.Contains(log, "dry-run") {
		t.Errorf("unexpected log content: %q", log)
	}
}

func TestDeleteSafe(t *testing.T) {
	c, dir := setupRepo(t)
	commit(t, dir, "a.txt", "base")
	base := defaultBase(t, c)

	// Merge a branch so it's deletable.
	run(dir, "checkout", "-b", "feature/merged")
	commit(t, dir, "b.txt", "work")
	run(dir, "checkout", base)
	run(dir, "merge", "feature/merged", "--no-ff", "-m", "merge")

	e := New(c, Options{})
	e.SetLogDir(filepath.Join(dir, ".git-purge"))

	branches, _ := c.ListLocalBranches()
	var target giter.Branch
	for _, b := range branches {
		if b.Name == "feature/merged" {
			target = b
			break
		}
	}
	if target.Name == "" {
		t.Fatal("merged branch not found")
	}

	err := e.DeleteSafe(target, 0, false)
	if err != nil {
		t.Fatalf("DeleteSafe: %v", err)
	}

	// Branch should be gone.
	branches, _ = c.ListLocalBranches()
	for _, b := range branches {
		if b.Name == "feature/merged" {
			t.Error("branch should have been deleted")
		}
	}

	if log := e.CurrentLog("feature/merged"); !strings.Contains(log, "deleted") {
		t.Errorf("expected delete log entry, got %q", log)
	}
}

func TestDeleteSafe_ProtectedRefused(t *testing.T) {
	c, dir := setupRepo(t)
	commit(t, dir, "a.txt", "base")

	e := New(c, Options{})
	mainBranch := giter.Branch{Name: "main", Hash: "abc"}
	err := e.DeleteSafe(mainBranch, 0, false)
	if err == nil {
		t.Fatal("expected error for protected branch, got nil")
	}
	if !strings.Contains(err.Error(), "protected") {
		t.Errorf("expected protection error, got %v", err)
	}
}

func TestAnalyze(t *testing.T) {
	c, dir := setupRepo(t)
	commit(t, dir, "a.txt", "base")
	base := defaultBase(t, c)

	// Merged branch.
	run(dir, "checkout", "-b", "feature/merged")
	commit(t, dir, "m.txt", "merged work")
	run(dir, "checkout", base)
	run(dir, "merge", "feature/merged", "--no-ff", "-m", "merge")

	// Active branch.
	run(dir, "checkout", "-b", "feature/active")
	commit(t, dir, "n.txt", "active work")
	run(dir, "checkout", base)

	an := analyzer.New(c, base)
	e := New(c, Options{})
	analysis, err := e.Analyze(an)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(analysis.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(analysis.Candidates))
	}
	if analysis.Candidates[0].Name != "feature/merged" {
		t.Errorf("expected feature/merged, got %s", analysis.Candidates[0].Name)
	}

	if len(analysis.Branches) != 3 {
		t.Errorf("expected 3 branches in results, got %d", len(analysis.Branches))
	}
	_ = analysis
}
