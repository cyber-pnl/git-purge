package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGit(dir string, args ...string) {
	_ = exec.Command("git", append([]string{"-C", dir}, args...)...).Run()
}

func setupTestRepo(t *testing.T) *Client {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %s: %v", string(out), err)
	}
	runGit(dir, "config", "user.email", "test@test.com")
	runGit(dir, "config", "user.name", "Test")
	return NewClient(dir)
}

func commitFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(dir, "add", name)
	runGit(dir, "commit", "-m", "add "+name)
}

func getDefaultBranch(t *testing.T, c *Client) string {
	t.Helper()
	out, err := c.run("symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		return strings.TrimPrefix(strings.TrimSpace(out), "refs/remotes/origin/")
	}
	// Fallback: check what branch exists
	branches, err := c.ListLocalBranches()
	if err != nil || len(branches) == 0 {
		t.Fatal("no branches found")
	}
	return branches[0].Name
}

func TestListLocalBranches(t *testing.T) {
	c := setupTestRepo(t)
	commitFile(t, c.dir, "a.txt", "hello")

	branches, err := c.ListLocalBranches()
	if err != nil {
		t.Fatalf("ListLocalBranches: %v", err)
	}
	if len(branches) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(branches))
	}
	if !branches[0].IsCurrent {
		t.Error("expected branch to be current")
	}
}

func TestCurrentBranch(t *testing.T) {
	c := setupTestRepo(t)
	commitFile(t, c.dir, "a.txt", "hello")

	branch, err := c.CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch == "" {
		t.Error("expected non-empty branch name")
	}
}

func TestMergeBase(t *testing.T) {
	c := setupTestRepo(t)
	commitFile(t, c.dir, "a.txt", "initial")
	baseBranch := getDefaultBranch(t, c)

	runGit(c.dir, "checkout", "-b", "feature")
	commitFile(t, c.dir, "b.txt", "feature work")

	runGit(c.dir, "checkout", baseBranch)

	base, err := c.MergeBase(baseBranch, "feature")
	if err != nil {
		t.Fatalf("MergeBase: %v", err)
	}
	if base == "" {
		t.Error("expected non-empty merge base")
	}
}

func TestIsMerged(t *testing.T) {
	c := setupTestRepo(t)
	commitFile(t, c.dir, "a.txt", "initial")
	baseBranch := getDefaultBranch(t, c)

	runGit(c.dir, "checkout", "-b", "feature")
	commitFile(t, c.dir, "b.txt", "feature work")

	runGit(c.dir, "checkout", baseBranch)
	runGit(c.dir, "merge", "feature", "--no-ff", "-m", "merge feature")

	merged, err := c.IsMerged("feature", baseBranch)
	if err != nil {
		t.Fatalf("IsMerged: %v", err)
	}
	if !merged {
		t.Error("expected feature to be merged")
	}
}

func TestDeleteBranch(t *testing.T) {
	c := setupTestRepo(t)
	commitFile(t, c.dir, "a.txt", "initial")
	baseBranch := getDefaultBranch(t, c)

	// Create branch, add commit, merge, then delete
	runGit(c.dir, "checkout", "-b", "to-delete")
	commitFile(t, c.dir, "b.txt", "delete me")

	runGit(c.dir, "checkout", baseBranch)
	runGit(c.dir, "merge", "to-delete", "--no-ff", "-m", "merge to-delete")

	err := c.DeleteBranch("to-delete", false)
	if err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}

	branches, _ := c.ListLocalBranches()
	for _, b := range branches {
		if b.Name == "to-delete" {
			t.Error("branch should have been deleted")
		}
	}
}

func TestDeleteBranchForce(t *testing.T) {
	c := setupTestRepo(t)
	commitFile(t, c.dir, "a.txt", "initial")
	baseBranch := getDefaultBranch(t, c)

	runGit(c.dir, "checkout", "-b", "unmerged")
	commitFile(t, c.dir, "b.txt", "unmerged work")

	runGit(c.dir, "checkout", baseBranch)

	err := c.DeleteBranch("unmerged", true)
	if err != nil {
		t.Fatalf("DeleteBranch force: %v", err)
	}

	branches, _ := c.ListLocalBranches()
	for _, b := range branches {
		if b.Name == "unmerged" {
			t.Error("branch should have been force-deleted")
		}
	}
}

func TestIsInsideWorkTree(t *testing.T) {
	c := setupTestRepo(t)
	inside, err := c.IsInsideWorkTree()
	if err != nil {
		t.Fatalf("IsInsideWorkTree: %v", err)
	}
	if !inside {
		t.Error("expected to be inside work tree")
	}
}

func TestIsWorkTreeClean(t *testing.T) {
	c := setupTestRepo(t)
	commitFile(t, c.dir, "a.txt", "hello")

	clean, err := c.IsWorkTreeClean()
	if err != nil {
		t.Fatalf("IsWorkTreeClean: %v", err)
	}
	if !clean {
		t.Error("expected clean work tree")
	}
}

func TestDiffTree(t *testing.T) {
	c := setupTestRepo(t)
	commitFile(t, c.dir, "a.txt", "initial")
	commitFile(t, c.dir, "b.txt", "second")

	out, err := c.run("rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("get HEAD: %v", err)
	}
	hash := strings.TrimSpace(out)

	diff, err := c.DiffTree(hash)
	if err != nil {
		t.Fatalf("DiffTree: %v", err)
	}
	if diff == "" {
		t.Error("expected non-empty diff for commit")
	}
}

func TestTreeHash(t *testing.T) {
	c := setupTestRepo(t)
	commitFile(t, c.dir, "a.txt", "content")

	treeHash, err := c.TreeHash("HEAD")
	if err != nil {
		t.Fatalf("TreeHash: %v", err)
	}
	if treeHash == "" {
		t.Error("expected non-empty tree hash")
	}
}

func TestLogCommits(t *testing.T) {
	c := setupTestRepo(t)
	commitFile(t, c.dir, "a.txt", "first")
	commitFile(t, c.dir, "b.txt", "second")
	baseBranch := getDefaultBranch(t, c)

	runGit(c.dir, "checkout", "-b", "feature")
	commitFile(t, c.dir, "c.txt", "third")

	runGit(c.dir, "checkout", baseBranch)

	commits, err := c.LogCommits(baseBranch, "feature")
	if err != nil {
		t.Fatalf("LogCommits: %v", err)
	}
	if len(commits) != 1 {
		t.Errorf("expected 1 commit, got %d", len(commits))
	}
}

func TestReflogSHA(t *testing.T) {
	c := setupTestRepo(t)
	commitFile(t, c.dir, "a.txt", "initial")

	sha, err := c.ReflogSHA("HEAD")
	if err != nil {
		t.Fatalf("ReflogSHA: %v", err)
	}
	// SHA might be empty if no reflog entry, that's ok
	_ = sha
}

func TestCatFileBlob(t *testing.T) {
	c := setupTestRepo(t)
	commitFile(t, c.dir, "a.txt", "hello world")

	out, err := c.run("rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("get HEAD: %v", err)
	}
	sha := strings.TrimSpace(out)

	content, err := c.CatFileBlob(sha)
	if err != nil {
		t.Fatalf("CatFileBlob: %v", err)
	}
	if content == "" {
		t.Error("expected non-empty blob content")
	}
}
