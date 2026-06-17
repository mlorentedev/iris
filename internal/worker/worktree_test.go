// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package worker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo creates a throwaway git repo with one commit so `git worktree
// add` has a HEAD to branch from. Author/committer identity is forced via -c so
// the test is independent of the developer's global git config / CI environment.
func initTestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runs := [][]string{
		{"init", "-q", "-b", "main"},
		{"-c", "user.email=ci@iris.test", "-c", "user.name=iris-ci", "commit", "--allow-empty", "-q", "-m", "root"},
	}
	for _, args := range runs {
		if out, err := runGit(context.Background(), repo, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return repo
}

func worktreeCount(t *testing.T, repo string) int {
	t.Helper()
	out, err := runGit(context.Background(), repo, "worktree", "list", "--porcelain")
	if err != nil {
		t.Fatalf("git worktree list: %v\n%s", err, out)
	}
	return strings.Count(string(out), "worktree ")
}

// TestWorktreeLifecycle is AC2 item C: a job gets a fresh, isolated worktree and
// it is removed on completion — even after pi has dirtied it — leaving no leaked
// worktree behind.
func TestWorktreeLifecycle(t *testing.T) {
	repo := initTestRepo(t)
	base := t.TempDir()
	ctx := context.Background()

	if n := worktreeCount(t, repo); n != 1 {
		t.Fatalf("precondition: worktree count = %d, want 1 (main only)", n)
	}

	wt, err := NewWorktree(ctx, repo, base, "abc123def")
	if err != nil {
		t.Fatalf("NewWorktree: %v", err)
	}
	if _, err := os.Stat(wt.Dir); err != nil {
		t.Fatalf("worktree dir not created: %v", err)
	}
	if n := worktreeCount(t, repo); n != 2 {
		t.Errorf("after add: worktree count = %d, want 2", n)
	}

	// Simulate pi editing files in the worktree — Remove must still succeed.
	if err := os.WriteFile(filepath.Join(wt.Dir, "EDITED.txt"), []byte("pi was here\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := wt.Remove(ctx); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(wt.Dir); !os.IsNotExist(err) {
		t.Errorf("worktree dir still present after Remove: err=%v", err)
	}
	if n := worktreeCount(t, repo); n != 1 {
		t.Errorf("after remove: worktree count = %d, want 1 (leaked worktree)", n)
	}
}
