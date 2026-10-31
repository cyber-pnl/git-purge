package safety

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	branch := giter.Branch{LastCommitDate: time.Now().AddDate(0, 0, -1).Format(time.RFC3339)}
	recent, err := e.RecentlyModified(branch, 14)
	if err != nil {
		t.Fatalf("RecentlyModified: %v", err)
	}

	old := giter.Branch{LastCommitDate: time.Now().AddDate(0, 0, -30).Format(time.RFC3339)}
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

func TestRecordLogAddsGitignoreEntry(t *testing.T) {
	c, dir := setupRepo(t)
	ignorePath := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(ignorePath, []byte("bin/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New(c, Options{})
	e.SetLogDir(filepath.Join(dir, ".git-purge"))
	for _, name := range []string{"feature/foo", "feature/bar"} {
		if err := e.DryRun(giter.Branch{Name: name, Hash: "abc123"}); err != nil {
			t.Fatalf("DryRun(%s): %v", name, err)
		}
	}

	data, err := os.ReadFile(ignorePath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.HasPrefix(content, "bin/\n") {
		t.Errorf("existing .gitignore content lost: %q", content)
	}
	if got := strings.Count(content, ".git-purge/"); got != 1 {
		t.Errorf("expected .git-purge/ exactly once, got %d in %q", got, content)
	}
}

func TestRecordLogCreatesGitignore(t *testing.T) {
	c, dir := setupRepo(t)
	e := New(c, Options{})
	e.SetLogDir(filepath.Join(dir, ".git-purge"))

	if err := e.DryRun(giter.Branch{Name: "feature/foo", Hash: "abc123"}); err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(data), ".git-purge/") {
		t.Errorf("expected .git-purge/ in .gitignore, got %q", string(data))
	}
}

func TestRecordLogSkipsExistingGitignoreEntry(t *testing.T) {
	c, dir := setupRepo(t)
	ignorePath := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(ignorePath, []byte(".git-purge\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New(c, Options{})
	e.SetLogDir(filepath.Join(dir, ".git-purge"))
	if err := e.DryRun(giter.Branch{Name: "feature/foo", Hash: "abc123"}); err != nil {
		t.Fatalf("DryRun: %v", err)
	}

	data, err := os.ReadFile(ignorePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != ".git-purge\n" {
		t.Errorf("existing entry should be kept unchanged, got %q", string(data))
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

	targetResult := analyzer.BranchResult{Branch: target, Status: analyzer.StatusMerged}
	res, err := e.DeleteSafe(&targetResult, 0, false)
	if err != nil {
		t.Fatalf("DeleteSafe: %v", err)
	}
	if !res.LocalDeleted {
		t.Error("expected local branch to be deleted")
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
	mainBranch := analyzer.BranchResult{
		Branch: giter.Branch{Name: "main", Hash: "abc"},
		Status: analyzer.StatusProtected,
	}
	_, err := e.DeleteSafe(&mainBranch, 0, false)
	if err == nil {
		t.Fatal("expected error for protected branch, got nil")
	}
	if !strings.Contains(err.Error(), "protected") {
		t.Errorf("expected protection error, got %v", err)
	}
}

func initBareRemote(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if out, err := exec.Command("git", "init", "--bare", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %s: %v", string(out), err)
	}
	return dir
}

func findCandidate(t *testing.T, analysis Analysis, name string) analyzer.BranchResult {
	t.Helper()
	for _, r := range analysis.Candidates {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("%s should be a candidate", name)
	return analyzer.BranchResult{}
}

func TestDeleteSafe_Remote(t *testing.T) {
	c, dir := setupRepo(t)
	commit(t, dir, "a.txt", "base")
	base := defaultBase(t, c)

	run(dir, "remote", "add", "origin", initBareRemote(t))
	run(dir, "checkout", "-b", "feature/merged")
	commit(t, dir, "b.txt", "work")
	run(dir, "push", "-u", "origin", "feature/merged")
	run(dir, "checkout", base)
	run(dir, "merge", "feature/merged", "--no-ff", "-m", "merge")

	e := New(c, Options{DeleteRemote: true})
	e.SetLogDir(filepath.Join(dir, ".git-purge"))
	analysis, err := e.Analyze(analyzer.New(c, base))
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	cand := findCandidate(t, analysis, "feature/merged")

	res, err := e.DeleteSafe(&cand, 0, false)
	if err != nil {
		t.Fatalf("DeleteSafe: %v", err)
	}
	if !res.LocalDeleted {
		t.Error("expected local branch to be deleted")
	}
	if !res.RemoteDeleted {
		t.Errorf("expected remote branch to be deleted, skipped: %q", res.RemoteSkipped)
	}

	sha, err := c.RemoteBranchSHA("origin", "feature/merged")
	if err != nil {
		t.Fatalf("RemoteBranchSHA: %v", err)
	}
	if sha != "" {
		t.Errorf("remote branch should be gone, got %q", sha)
	}
}

func TestDeleteSafe_RemoteAbsent(t *testing.T) {
	c, dir := setupRepo(t)
	commit(t, dir, "a.txt", "base")
	base := defaultBase(t, c)

	run(dir, "remote", "add", "origin", initBareRemote(t))
	run(dir, "checkout", "-b", "feature/merged")
	commit(t, dir, "b.txt", "work")
	run(dir, "checkout", base)
	run(dir, "merge", "feature/merged", "--no-ff", "-m", "merge")

	e := New(c, Options{DeleteRemote: true})
	e.SetLogDir(filepath.Join(dir, ".git-purge"))
	analysis, err := e.Analyze(analyzer.New(c, base))
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	cand := findCandidate(t, analysis, "feature/merged")

	res, err := e.DeleteSafe(&cand, 0, false)
	if err != nil {
		t.Fatalf("DeleteSafe: %v", err)
	}
	if !res.LocalDeleted {
		t.Error("expected local branch to be deleted")
	}
	if res.RemoteDeleted {
		t.Error("remote branch did not exist and should not be reported as deleted")
	}
	if !strings.Contains(res.RemoteSkipped, "does not exist") {
		t.Errorf("expected 'does not exist' skip reason, got %q", res.RemoteSkipped)
	}
}

func TestDeleteSafe_RefusesUnmerged(t *testing.T) {
	c, dir := setupRepo(t)
	commit(t, dir, "a.txt", "base")
	base := defaultBase(t, c)

	run(dir, "checkout", "-b", "feature/active")
	commit(t, dir, "b.txt", "active work")
	run(dir, "checkout", base)

	branches, _ := c.ListLocalBranches()
	var target giter.Branch
	for _, b := range branches {
		if b.Name == "feature/active" {
			target = b
		}
	}
	if target.Name == "" {
		t.Fatal("active branch not found")
	}

	e := New(c, Options{DeleteRemote: true})
	e.SetLogDir(filepath.Join(dir, ".git-purge"))
	_, err := e.DeleteSafe(&analyzer.BranchResult{Branch: target, Status: analyzer.StatusActiveUnmerged}, 0, true)
	if err == nil {
		t.Fatal("expected refusal for unmerged branch")
	}
	if !strings.Contains(err.Error(), "not safe") {
		t.Errorf("unexpected error: %v", err)
	}

	branches, _ = c.ListLocalBranches()
	for _, b := range branches {
		if b.Name == "feature/active" {
			return
		}
	}
	t.Error("unmerged branch must not be deleted")
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
