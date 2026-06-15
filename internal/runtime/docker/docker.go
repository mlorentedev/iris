// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

// Package docker implements the runtime.AgentRuntime base contract on a single
// Docker host: each iris agent is one labelled container. Idempotency and spec
// drift are derived from container labels, so the adapter holds no state of its
// own and survives a motor restart. Capability sub-interfaces (Ollama, Qdrant,
// Network) build on this base in follow-up work.
package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/mlorentedev/iris/internal/runtime"
)

// Labels iris stamps on every managed container. They are the adapter's only
// source of truth for which agent a container belongs to and what spec it was
// started with — no external state store.
const (
	labelManaged  = "iris.managed"   // "true" on every iris-managed container
	labelAgent    = "iris.agent"     // AgentSpec.Name
	labelTeam     = "iris.team"      // AgentSpec.TeamID
	labelSpecHash = "iris.spec-hash" // AgentSpec.SpecHash(), for drift detection
)

// Runtime is the Docker substrate adapter.
type Runtime struct {
	cli client.APIClient
}

// Compile-time proof the adapter satisfies the FROZEN base contract.
var _ runtime.AgentRuntime = (*Runtime)(nil)

// New connects to the Docker daemon using the standard environment
// (DOCKER_HOST, DOCKER_TLS_VERIFY, ...) and negotiates the API version with the
// server so the adapter works across daemon versions.
func New() (*Runtime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker: connect to daemon: %w", err)
	}
	return &Runtime{cli: cli}, nil
}

// Close releases the underlying Docker client.
func (r *Runtime) Close() error { return r.cli.Close() }

// Start launches the agent as a labelled container, or returns the existing one
// when the spec is unchanged (idempotent) / ErrSpecDrift when it differs.
func (r *Runtime) Start(ctx context.Context, spec runtime.AgentSpec) (runtime.AgentHandle, error) {
	hash := spec.SpecHash()

	existing, err := r.find(ctx, spec.Name, spec.TeamID)
	if err != nil {
		return runtime.AgentHandle{}, err
	}
	if existing != nil {
		if existing.Labels[labelSpecHash] != hash {
			return runtime.AgentHandle{}, runtime.ErrSpecDrift
		}
		return r.handle(existing.ID, spec.Name, spec.TeamID), nil
	}

	if err := r.ensureImage(ctx, spec.Image); err != nil {
		return runtime.AgentHandle{}, err
	}

	cfg := &container.Config{
		Image: spec.Image,
		Cmd:   spec.Command, // nil → image's default entrypoint/CMD
		Env:   envSlice(spec.Env),
		Labels: map[string]string{
			labelManaged:  "true",
			labelAgent:    spec.Name,
			labelTeam:     spec.TeamID,
			labelSpecHash: hash,
		},
	}
	// Init runs a tiny init (tini) as PID 1 so the agent process receives
	// forwarded signals (graceful SIGTERM on Stop) and child zombies are reaped.
	useInit := true
	host := &container.HostConfig{
		Init: &useInit,
		Resources: container.Resources{
			NanoCPUs: int64(spec.Resources.CPUMillis) * 1_000_000, // millicores → nano-CPUs
			Memory:   spec.Resources.MemoryBytes,
		},
	}
	created, err := r.cli.ContainerCreate(ctx, cfg, host, nil, nil, containerName(spec.Name, spec.TeamID))
	if err != nil {
		return runtime.AgentHandle{}, fmt.Errorf("docker: create %s: %w", spec.Name, err)
	}
	if err := r.cli.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		return runtime.AgentHandle{}, fmt.Errorf("docker: start %s: %w", spec.Name, err)
	}
	return r.handle(created.ID, spec.Name, spec.TeamID), nil
}

// Stop gracefully stops and removes the container. Missing containers are a
// no-op so Stop is idempotent.
func (r *Runtime) Stop(ctx context.Context, h runtime.AgentHandle) error {
	if err := r.cli.ContainerStop(ctx, h.ID, container.StopOptions{}); err != nil && !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("docker: stop %s: %w", h.Name, err)
	}
	if err := r.cli.ContainerRemove(ctx, h.ID, container.RemoveOptions{Force: true}); err != nil && !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("docker: remove %s: %w", h.Name, err)
	}
	return nil
}

// Status inspects the container and maps its state to an AgentStatus.
func (r *Runtime) Status(ctx context.Context, h runtime.AgentHandle) (runtime.AgentStatus, error) {
	insp, err := r.cli.ContainerInspect(ctx, h.ID)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return "", runtime.ErrAgentNotFound
		}
		return "", fmt.Errorf("docker: inspect %s: %w", h.Name, err)
	}
	if insp.State == nil {
		return runtime.StatusFailed, nil
	}
	return mapState(insp.State), nil
}

// Exec runs cmd inside the container and captures its exit code and output.
func (r *Runtime) Exec(ctx context.Context, h runtime.AgentHandle, cmd []string) (runtime.ExecResult, error) {
	ec, err := r.cli.ContainerExecCreate(ctx, h.ID, container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return runtime.ExecResult{}, runtime.ErrAgentNotFound
		}
		return runtime.ExecResult{}, fmt.Errorf("docker: exec create %s: %w", h.Name, err)
	}
	att, err := r.cli.ContainerExecAttach(ctx, ec.ID, container.ExecAttachOptions{})
	if err != nil {
		return runtime.ExecResult{}, fmt.Errorf("docker: exec attach %s: %w", h.Name, err)
	}
	defer att.Close()

	// stdout/stderr are multiplexed on one stream when no TTY is attached;
	// stdcopy demultiplexes them back into separate buffers.
	var out, errOut bytes.Buffer
	if _, err := stdcopy.StdCopy(&out, &errOut, att.Reader); err != nil {
		return runtime.ExecResult{}, fmt.Errorf("docker: exec read %s: %w", h.Name, err)
	}
	insp, err := r.cli.ContainerExecInspect(ctx, ec.ID)
	if err != nil {
		return runtime.ExecResult{}, fmt.Errorf("docker: exec inspect %s: %w", h.Name, err)
	}
	return runtime.ExecResult{ExitCode: insp.ExitCode, Stdout: out.String(), Stderr: errOut.String()}, nil
}

// find returns the single iris container for (name, team), or nil if none.
func (r *Runtime) find(ctx context.Context, name, team string) (*container.Summary, error) {
	f := filters.NewArgs(
		filters.Arg("label", labelAgent+"="+name),
		filters.Arg("label", labelTeam+"="+team),
	)
	list, err := r.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
	if err != nil {
		return nil, fmt.Errorf("docker: list %s: %w", name, err)
	}
	if len(list) == 0 {
		return nil, nil
	}
	return &list[0], nil
}

// ensureImage pulls the image only when it is not already present locally.
// ImagePull always contacts the registry, so checking presence first keeps
// Start fast and offline-tolerant once an image is cached.
func (r *Runtime) ensureImage(ctx context.Context, ref string) error {
	imgs, err := r.cli.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", ref)),
	})
	if err == nil && len(imgs) > 0 {
		return nil
	}
	return r.pull(ctx, ref)
}

// pull fetches the image from its registry.
func (r *Runtime) pull(ctx context.Context, ref string) error {
	rc, err := r.cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("docker: pull %s: %w", ref, err)
	}
	defer func() { _ = rc.Close() }()
	if _, err := io.Copy(io.Discard, rc); err != nil {
		return fmt.Errorf("docker: pull %s: %w", ref, err)
	}
	return nil
}

func (r *Runtime) handle(id, name, team string) runtime.AgentHandle {
	return runtime.AgentHandle{ID: id, Name: name, TeamID: team, Substrate: runtime.SubstrateDocker}
}

// containerName is the deterministic container name for an agent identity.
func containerName(name, team string) string {
	return fmt.Sprintf("iris-%s-%s", team, name)
}

// envSlice converts an env map to a sorted "k=v" slice for deterministic output.
func envSlice(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(env))
	for _, k := range keys {
		out = append(out, k+"="+env[k])
	}
	return out
}

// mapState translates a Docker container state into an AgentStatus.
func mapState(s *container.State) runtime.AgentStatus {
	switch s.Status {
	case container.StateCreated, container.StateRestarting:
		return runtime.StatusStarting
	case container.StateRunning, container.StatePaused:
		return runtime.StatusRunning
	case container.StateRemoving:
		return runtime.StatusStopping
	case container.StateExited:
		if s.ExitCode == 0 {
			return runtime.StatusStopped
		}
		return runtime.StatusFailed
	default: // dead, or any unknown future state
		return runtime.StatusFailed
	}
}
