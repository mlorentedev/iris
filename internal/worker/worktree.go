// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package worker

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
)

// jobIDRE constrains a job id to characters that are safe in both a filesystem
// path and a git ref: it forbids '/', '.', and whitespace, so a hostile
// message_id can neither traverse out of the worktree base (e.g. "../../etc")
// nor smuggle a path/ref separator. A real MessageID is a UUID, which matches.
var jobIDRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,127}$`)

// Worktree is an isolated git worktree for a single job. pi runs with cwd = Dir,
// so its file edits land here and never touch the worker's own checkout — the
// per-job isolation boundary of ADR-002. The worktree is created per job and
// removed on completion (proposal item C: no leaked worktrees).
type Worktree struct {
	Dir    string // absolute path pi runs in
	repo   string // the source repository the worktree was added from
	branch string // the throwaway branch the worktree is checked out on
}

// NewWorktree adds a fresh worktree of repo under base, on a new branch, both
// derived from id (the job's MessageID) so concurrent jobs never collide. id is
// validated (see jobIDRE) before it is used to build a path or ref. base must
// already exist.
func NewWorktree(ctx context.Context, repo, base, id string) (*Worktree, error) {
	if !jobIDRE.MatchString(id) {
		return nil, fmt.Errorf("worker: unsafe job id %q for worktree path/ref", id)
	}
	dir := filepath.Join(base, "job-"+id)
	branch := "iris/job-" + id

	// `worktree add -b <branch> <dir> HEAD` checks out a new branch at the
	// current HEAD into an isolated directory.
	if out, err := runGit(ctx, repo, "worktree", "add", "-b", branch, dir, "HEAD"); err != nil {
		return nil, fmt.Errorf("worker: git worktree add %s: %w: %s", dir, err, out)
	}
	return &Worktree{Dir: dir, repo: repo, branch: branch}, nil
}

// Remove tears the worktree down and deletes its branch. It force-removes so a
// dirty tree (pi will have edited files) is still cleaned. nil-safe and meant to
// be deferred, so it runs on every exit path of a job. A leaked branch is
// cosmetic; a leaked worktree holds a lock and disk, so its removal is the part
// that must not be swallowed.
func (w *Worktree) Remove(ctx context.Context) error {
	if w == nil {
		return nil
	}
	if out, err := runGit(ctx, w.repo, "worktree", "remove", "--force", w.Dir); err != nil {
		return fmt.Errorf("worker: git worktree remove %s: %w: %s", w.Dir, err, out)
	}
	// Best-effort branch cleanup; its failure must not mask a successful removal.
	_, _ = runGit(ctx, w.repo, "branch", "-D", w.branch)
	return nil
}

// runGit is the single chokepoint for invoking git, mirroring protocol.SubjectFor's
// "construct in one place" style. The command is always the fixed "git" binary
// and the args are constant subcommand verbs plus validated job ids and
// worker-owned paths, passed directly as argv (never through a shell) — so the
// gosec G204 heuristic is a false positive here.
func runGit(ctx context.Context, repo string, args ...string) ([]byte, error) {
	full := append([]string{"-C", repo}, args...)
	return exec.CommandContext(ctx, "git", full...).CombinedOutput() //nolint:gosec // G204: fixed binary + validated argv, no shell
}
