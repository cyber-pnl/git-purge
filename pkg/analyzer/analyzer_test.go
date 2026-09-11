package analyzer

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	giter "github.com/jbdanho/git-purge/pkg/git"
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

func TestAnalyzeAll_ProtectedAndCurrent(t *testing.T) {
	c, dir := setupRepo(t)
	commit(t, dir, "a.txt", "base")
	base := defaultBase(t, c)

	results, err := New(c, base).AnalyzeAll()
	if err != nil {
		t.Fatalf("AnalyzeAll: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(results))
	}
	if results[0].Status != StatusProtected {
		t.Errorf("expected PROTECTED, got %s", results[0].Status)
	}
}

func TestAnalyzeAll_Merged(t *testing.T) {
	c, dir := setupRepo(t)
	commit(t, dir, "a.txt", "base")
	base := defaultBase(t, c)

	run(dir, "checkout", "-b", "feature")
	commit(t, dir, "b.txt", "feature work")
	run(dir, "checkout", base)
	run(dir, "merge", "feature", "--no-ff", "-m", "merge feature")

	results, err := New(c, base).AnalyzeAll()
	if err != nil {
		t.Fatalf("AnalyzeAll: %v", err)
	}
	found := false
	for _, r := range results {
		if r.Name == "feature" {
			found = true
			if r.Status != StatusMerged {
				t.Errorf("expected MERGED, got %s", r.Status)
			}
		}
	}
	if !found {
		t.Error("feature branch not found")
	}
}

func TestAnalyzeAll_SquashMerged(t *testing.T) {
	c, dir := setupRepo(t)
	commit(t, dir, "a.txt", "base")
	base := defaultBase(t, c)

	// Create feature branch with real work.
	run(dir, "checkout", "-b", "feature")
	commit(t, dir, "b.txt", "feature work")
	commit(t, dir, "c.txt", "more work")

	// Squash-merge into base: delete feature changes and re-apply as a single commit.
	run(dir, "checkout", base)
	run(dir, "merge", "--squash", "feature")
	run(dir, "commit", "-m", "squash merge feature")

	results, err := New(c, base).AnalyzeAll()
	if err != nil {
		t.Fatalf("AnalyzeAll: %v", err)
	}
	found := false
	for _, r := range results {
		if r.Name == "feature" {
			found = true
			if r.Status != StatusSquashMerged {
				t.Errorf("expected SQUASH_MERGED, got %s", r.Status)
			}
		}
	}
	if !found {
		t.Error("feature branch not found")
	}
}

func TestAnalyzeAll_ActiveUnmerged(t *testing.T) {
	c, dir := setupRepo(t)
	commit(t, dir, "a.txt", "base")
	base := defaultBase(t, c)

	run(dir, "checkout", "-b", "feature")
	commit(t, dir, "b.txt", "feature work")
	run(dir, "checkout", base)

	// base stays untouched → feature is unmerged.

	results, err := New(c, base).AnalyzeAll()
	if err != nil {
		t.Fatalf("AnalyzeAll: %v", err)
	}
	found := false
	for _, r := range results {
		if r.Name == "feature" {
			found = true
			if r.Status != StatusActiveUnmerged {
				t.Errorf("expected ACTIVE_UNMERGED, got %s", r.Status)
			}
		}
	}
	if !found {
		t.Error("feature branch not found")
	}
}

func TestAnalyzeAll_Gone(t *testing.T) {
	// Simulate a gone branch by pointing branch config at a non-existent
	// remote-tracking ref, the same way git shows "[gone]".
	c, dir := setupRepo(t)
	commit(t, dir, "a.txt", "base")
	base := defaultBase(t, c)

	run(dir, "checkout", "-b", "feature")
	commit(t, dir, "b.txt", "work")
	run(dir, "checkout", base)

	run(dir, "config", "branch.feature.remote", "origin")
	run(dir, "config", "branch.feature.merge", "refs/heads/feature")
	run(dir, "remote", "add", "origin", "https://example.com/repo.git")

	results, err := New(c, base).AnalyzeAll()
	if err != nil {
		t.Fatalf("AnalyzeAll: %v", err)
	}
	found := false
	for _, r := range results {
		if r.Name == "feature" {
			found = true
			if r.Status != StatusGone {
				t.Errorf("expected GONE, got %s", r.Status)
			}
		}
	}
	if !found {
		t.Error("feature branch not found")
	}
}

func TestCandidates(t *testing.T) {
	c, dir := setupRepo(t)
	commit(t, dir, "a.txt", "base")
	base := defaultBase(t, c)

	// Merged branch.
	run(dir, "checkout", "-b", "merged-feature")
	commit(t, dir, "m.txt", "merged work")
	run(dir, "checkout", base)
	run(dir, "merge", "merged-feature", "--no-ff", "-m", "merge")

	// Active branch.
	run(dir, "checkout", "-b", "active-feature")
	commit(t, dir, "n.txt", "active work")
	run(dir, "checkout", base)

	candidates, err := New(c, base).Candidates()
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Name != "merged-feature" {
		t.Errorf("expected merged-feature, got %s", candidates[0].Name)
	}
}

func TestStatusString(t *testing.T) {
	cases := []struct {
		s    Status
		want string
	}{
		{StatusMerged, "MERGED"},
		{StatusSquashMerged, "SQUASH_MERGED"},
		{StatusGone, "GONE"},
		{StatusProtected, "PROTECTED"},
		{StatusActiveUnmerged, "ACTIVE_UNMERGED"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.s.String(); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestHasUpstream(t *testing.T) {
	if HasUpstream(giter.Branch{Upstream: "origin/main"}) != true {
		t.Error("expected true for upstream set")
	}
	if HasUpstream(giter.Branch{Upstream: ""}) != false {
		t.Error("expected false for no upstream")
	}
}

func TestIsProtectedByDefault(t *testing.T) {
	a := New(setupDummyClient(), "main")
	cases := []string{"main", "master", "dev", "release/1.0"}
	expected := []bool{true, true, true, false}
	for i, name := range cases {
		if got := a.isProtectedByDefault(name); got != expected[i] {
			t.Errorf("isProtectedByDefault(%q) = %v, want %v", name, got, expected[i])
		}
	}
}

func setupDummyClient() *giter.Client {
	return giter.NewClient(".")
}
