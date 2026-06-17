// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

// Command fakepi is a test stand-in for the `pi` coding-agent CLI. It replays a
// fixture of canonical `pi --mode json` JSONL events to stdout and exits with a
// configurable code, so the worker's pi driver is testable in CI with no tokens
// and no network. It is NOT shipped: it lives under testdata/ (ignored by the go
// tool for `./...`) and is compiled on demand by the worker tests.
//
// Knobs (env):
//
//	FAKEPI_FIXTURE  path to a JSONL file whose non-empty lines are echoed to
//	                stdout, one event per line (required).
//	FAKEPI_EXIT     process exit code (default 0).
//	FAKEPI_HANG     if set, block forever after emitting the fixture instead of
//	                exiting — models a pi session still working when SIGTERM
//	                arrives, so the worker's reap-on-cancel path can be tested.
//
// pi's real argv (`--mode json "<prompt>"`) is accepted and ignored; the fixture
// alone decides the event stream.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
)

func main() {
	fixture := os.Getenv("FAKEPI_FIXTURE")
	if fixture == "" {
		fmt.Fprintln(os.Stderr, "fakepi: FAKEPI_FIXTURE is required")
		os.Exit(2)
	}

	f, err := os.Open(fixture)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fakepi: open fixture: %v\n", err)
		os.Exit(2)
	}
	defer func() { _ = f.Close() }()

	out := bufio.NewWriter(os.Stdout)
	in := bufio.NewScanner(f)
	in.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // tool results can be large
	for in.Scan() {
		line := in.Bytes()
		if len(line) == 0 {
			continue
		}
		_, _ = out.Write(line)
		_ = out.WriteByte('\n')
		_ = out.Flush() // flush per line so the reader sees a live stream
	}

	if os.Getenv("FAKEPI_HANG") != "" {
		select {} // block until the parent kills us (ctx cancel / SIGTERM)
	}

	code := 0
	if v := os.Getenv("FAKEPI_EXIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			code = n
		}
	}
	os.Exit(code)
}
