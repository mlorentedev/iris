// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package worker

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
)

// maxEventLine bounds a single JSONL event line. pi tool results (whole-file
// reads, command output) can be large, so the scanner buffer is sized well above
// bufio's 64 KiB default to avoid bufio.ErrTooLong mid-stream.
const maxEventLine = 1 << 20 // 1 MiB

// Driver runs pi as a subprocess in `--mode json` and streams its event lines to
// a handler. It is deliberately I/O only: it knows how to spawn pi and surface
// its events and exit code, but nothing about NATS, worktrees, or the iris
// protocol — those compose around it in later tasks.
type Driver struct {
	Bin  string   // pi binary; default "pi" (resolved on PATH)
	Args []string // extra args inserted before the prompt, e.g. ["--model", "x"]
	Dir  string   // working directory: the per-job git worktree
	Env  []string // process environment (inference passthrough); nil inherits
}

// Result summarizes a finished pi run. A non-zero ExitCode is NOT returned as a
// Go error — it is data the caller classifies (the failure path in AC5 turns it
// into an activity_event). Run returns an error only for spawn/pipe failures.
type Result struct {
	ExitCode    int  // process exit status (-1 if it never started cleanly)
	SawAgentEnd bool // whether the terminal agent_end event was observed
}

// Run launches `pi --mode json <prompt>` in d.Dir, decoding each stdout line
// into an Event and passing it to handle, until the stream ends. It blocks until
// the process exits and returns its Result. Cancelling ctx terminates the
// subprocess (exec.CommandContext), which is what graceful shutdown relies on.
func (d *Driver) Run(ctx context.Context, prompt string, handle func(Event)) (Result, error) {
	bin := d.Bin
	if bin == "" {
		bin = "pi"
	}
	args := append([]string{"--mode", "json"}, d.Args...)
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = d.Dir
	cmd.Env = d.Env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Result{ExitCode: -1}, fmt.Errorf("worker: pi stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return Result{ExitCode: -1}, fmt.Errorf("worker: start pi %q: %w", bin, err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), maxEventLine)

	var res Result
	// Drain pi's event stream to EOF. We never break early on agent_end: pi
	// closes stdout when it exits, so reading to EOF is what lets cmd.Wait()
	// return instead of deadlocking on an unread pipe. agent_end is recorded as
	// a flag, not a stop condition.
	for scanner.Scan() {
		ev, err := decodeEvent(scanner.Bytes())
		if err != nil {
			// One unparseable telemetry line is a pi-side protocol violation,
			// not a reason to abort an otherwise-healthy coding session (the
			// real work lives in the worktree, not in this parser). Skip it —
			// but never silently: WARN with the line truncated so it stays
			// diagnosable without flooding logs with large tool outputs. A
			// broken *stream* (vs one bad line) is caught by scanner.Err() below.
			slog.Warn("worker: skipping unparseable pi event",
				"err", err, "line", truncate(scanner.Bytes(), 256))
			continue
		}
		handle(ev)
		if ev.Type == EventAgentEnd {
			res.SawAgentEnd = true
		}
	}
	scanErr := scanner.Err()

	waitErr := cmd.Wait()
	res.ExitCode = exitCodeFrom(waitErr)

	// A stream that broke mid-read (pipe died, or a line exceeded maxEventLine)
	// is a real failure even when Wait() later reports an exit code — surface it
	// (with the partial Result still populated) so the dispatch layer turns it
	// into a failure activity_event (AC5) rather than a silently-truncated success.
	if scanErr != nil {
		return res, fmt.Errorf("worker: read pi event stream: %w", scanErr)
	}
	return res, nil
}

// truncate renders at most n bytes of b as a string, for bounded log output.
func truncate(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n]) + "..."
	}
	return string(b)
}

// exitCodeFrom maps the error from cmd.Wait() to a numeric exit code: 0 on
// success, the process status for a normal non-zero exit, and -1 for anything
// else (signalled, killed by ctx cancel, failed to run).
func exitCodeFrom(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := errors.AsType[*exec.ExitError](err); ok {
		return ee.ExitCode()
	}
	return -1
}
