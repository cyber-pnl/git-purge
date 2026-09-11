package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Branch represents a local Git branch with metadata.
type Branch struct {
	Name           string
	Hash           string
	Upstream       string
	UpstreamGone   bool
	IsCurrent      bool
	LastCommitDate string
}

// Client wraps Git operations via exec.Command.
type Client struct {
	dir string
}

// NewClient returns a Git client operating in the given directory.
func NewClient(dir string) *Client {
	return &Client{dir: dir}
}

// Dir returns the working directory bound to this client.
func (c *Client) Dir() string {
	return c.dir
}

func (c *Client) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = c.dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(stderr.String()), err)
	}
	return stdout.String(), nil
}

// ListLocalBranches returns all local branches with metadata.
func (c *Client) ListLocalBranches() ([]Branch, error) {
	// Get current branch
	current, err := c.CurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("get current branch: %w", err)
	}

	// Get branches with tracking info
	out, err := c.run("branch", "-vv", "--format=%(refname:short)|%(objectname:short)|%(upstream:short)|%(upstream:track,nobracket)|%(committerdate:iso8601-strict)")
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	var branches []Branch
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 5 {
			continue
		}
		b := Branch{
			Name:           parts[0],
			Hash:           parts[1],
			Upstream:       parts[2],
			UpstreamGone:   parts[3] == "gone",
			IsCurrent:      parts[0] == current,
			LastCommitDate: parts[4],
		}
		branches = append(branches, b)
	}
	return branches, nil
}

// HasCommits returns true if the repository has at least one commit/HEAD.
func (c *Client) HasCommits() (bool, error) {
	_, err := c.run("rev-parse", "--verify", "--quiet", "HEAD")
	if err != nil {
		return false, nil
	}
	return true, nil
}

// CurrentBranch returns the name of the currently checked out branch.
func (c *Client) CurrentBranch() (string, error) {
	out, err := c.run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("get current branch: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// IsMerged returns true if branch is fully merged into baseBranch.
func (c *Client) IsMerged(branch, baseBranch string) (bool, error) {
	_, err := c.run("merge-base", "--is-ancestor", branch, baseBranch)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("check merge status: %w", err)
	}
	return true, nil
}

// MergeBase returns the merge-base commit between two refs.
func (c *Client) MergeBase(a, b string) (string, error) {
	out, err := c.run("merge-base", a, b)
	if err != nil {
		return "", fmt.Errorf("merge-base %s %s: %w", a, b, err)
	}
	return strings.TrimSpace(out), nil
}

// DiffTree returns the diff output of a commit versus its parent.
func (c *Client) DiffTree(commit string) (string, error) {
	out, err := c.run("diff-tree", "-r", "--name-only", commit)
	if err != nil {
		return "", fmt.Errorf("diff-tree %s: %w", commit, err)
	}
	return out, nil
}

// DiffTreeStat returns a stat summary of changes in a commit.
func (c *Client) DiffTreeStat(commit string) (string, error) {
	out, err := c.run("diff-tree", "-r", "--stat", commit)
	if err != nil {
		return "", fmt.Errorf("diff-tree stat %s: %w", commit, err)
	}
	return out, nil
}

// DeleteBranch deletes the given branch. If force is true, uses -D; otherwise -d.
func (c *Client) DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := c.run("branch", flag, name)
	if err != nil {
		return fmt.Errorf("delete branch %s: %w", name, err)
	}
	return nil
}

// ReflogSHA returns the SHA recorded in reflog for a branch, or empty if none.
func (c *Client) ReflogSHA(branch string) (string, error) {
	out, err := c.run("reflog", "show", "--format=%H", "-n", "1", branch)
	if err != nil {
		// Branch might not exist in reflog
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

// IsInsideWorkTree returns true if the path is inside a Git work tree.
func (c *Client) IsInsideWorkTree() (bool, error) {
	_, err := c.run("rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false, nil
	}
	return true, nil
}

// IsWorkTreeClean returns true if the working tree has no uncommitted changes.
func (c *Client) IsWorkTreeClean() (bool, error) {
	out, err := c.run("status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("check work tree status: %w", err)
	}
	return strings.TrimSpace(out) == "", nil
}

// Log returns the commit log between two refs.
func (c *Client) Log(from, to string) (string, error) {
	args := []string{"log", "--oneline"}
	if from != "" && to != "" {
		args = append(args, from+".."+to)
	} else if to != "" {
		args = append(args, to)
	}
	out, err := c.run(args...)
	if err != nil {
		return "", fmt.Errorf("git log: %w", err)
	}
	return out, nil
}

// CatFileBlob returns the content of a blob object.
func (c *Client) CatFileBlob(sha string) (string, error) {
	out, err := c.run("cat-file", "-p", sha)
	if err != nil {
		return "", fmt.Errorf("cat-file %s: %w", sha, err)
	}
	return out, nil
}

// TreeHash returns the tree hash for a given commit.
func (c *Client) TreeHash(commit string) (string, error) {
	out, err := c.run("rev-parse", commit+"^{tree}")
	if err != nil {
		return "", fmt.Errorf("tree hash %s: %w", commit, err)
	}
	return strings.TrimSpace(out), nil
}

// LogCommits returns a list of commit SHAs between two refs.
func (c *Client) LogCommits(from, to string) ([]string, error) {
	args := []string{"log", "--format=%H"}
	if from != "" && to != "" {
		args = append(args, from+".."+to)
	} else if to != "" {
		args = append(args, to)
	}
	out, err := c.run(args...)
	if err != nil {
		return nil, fmt.Errorf("git log commits: %w", err)
	}
	var commits []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			commits = append(commits, line)
		}
	}
	return commits, nil
}
