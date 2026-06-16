// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

// Package k8s implements the runtime.AgentRuntime base contract on Kubernetes
// via kubernetes-sigs/agent-sandbox: each iris agent is one Sandbox custom
// resource (agents.x-k8s.io/v1beta1) backing a single pod with a stable
// identity. Idempotency and spec drift derive from labels on the Sandbox CR, so
// the adapter holds no state of its own and survives a motor restart — the same
// stateless, label-driven design as the Docker adapter.
//
// agent-sandbox is treated as a swappable substrate (ADR-005 / OQ-1): only this
// package imports its API, and the runtime contract leaks nothing K8s- or
// agent-sandbox-specific. Replacing it with a raw StatefulSet/Pod fallback would
// touch this package alone.
package k8s

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	utilexec "k8s.io/utils/exec"
	sandboxv1 "sigs.k8s.io/agent-sandbox/api/v1beta1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mlorentedev/iris/internal/runtime"
)

// Labels iris stamps on every managed Sandbox. They are the adapter's only
// source of truth for which agent a Sandbox belongs to and what spec it was
// started with — no external state store, mirroring the Docker adapter.
const (
	labelManaged  = "iris.managed"   // "true" on every iris-managed Sandbox
	labelAgent    = "iris.agent"     // AgentSpec.Name
	labelTeam     = "iris.team"      // AgentSpec.TeamID
	labelSpecHash = "iris.spec-hash" // AgentSpec.SpecHash(), for drift detection

	// agentContainerName is the name of the agent's container inside the Sandbox
	// pod. Exec targets it; the sidecar capability adds containers alongside it.
	agentContainerName = "agent"
)

// Runtime is the Kubernetes substrate adapter. cli does typed CRUD on the
// Sandbox CR; core + restConfig drive pod exec (a subresource the controller-
// runtime client does not expose).
type Runtime struct {
	cli        ctrlclient.Client
	core       kubernetes.Interface
	restConfig *rest.Config
	namespace  string
}

// Compile-time proof the adapter satisfies the FROZEN base contract.
var _ runtime.AgentRuntime = (*Runtime)(nil)

// New connects to the cluster using the in-cluster config when running inside a
// pod, falling back to the standard kubeconfig loading rules (KUBECONFIG /
// ~/.kube/config) otherwise. namespace is where agents are provisioned; empty
// means "default".
func New(namespace string) (*Runtime, error) {
	cfg, err := restConfig()
	if err != nil {
		return nil, err
	}
	return NewWithConfig(cfg, namespace)
}

// NewWithConfig builds the adapter from an explicit rest.Config. It is the seam
// tests use to point the adapter at a throwaway cluster.
func NewWithConfig(cfg *rest.Config, namespace string) (*Runtime, error) {
	scheme, err := buildScheme()
	if err != nil {
		return nil, fmt.Errorf("k8s: build scheme: %w", err)
	}
	cli, err := ctrlclient.New(cfg, ctrlclient.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("k8s: build client: %w", err)
	}
	core, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("k8s: build core client: %w", err)
	}
	if namespace == "" {
		namespace = "default"
	}
	return &Runtime{cli: cli, core: core, restConfig: cfg, namespace: namespace}, nil
}

// restConfig loads the in-cluster config, falling back to kubeconfig. Pattern
// adapted from agent-sandbox examples/sandboxed-tools (Apache-2.0).
func restConfig() (*rest.Config, error) {
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("k8s: load kubeconfig: %w", err)
	}
	return cfg, nil
}

// buildScheme registers core/v1 (pods, for exec) and the agent-sandbox API so
// the typed client can encode/decode Sandbox objects.
func buildScheme() (*apiruntime.Scheme, error) {
	scheme := apiruntime.NewScheme()
	if err := kscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := sandboxv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	return scheme, nil
}

// Start provisions the agent as a labelled Sandbox CR, or returns the existing
// one when the spec is unchanged (idempotent) / ErrSpecDrift when it differs.
func (r *Runtime) Start(ctx context.Context, spec runtime.AgentSpec) (runtime.AgentHandle, error) {
	hash := spec.SpecHash()
	name := sandboxName(spec.Name, spec.TeamID)

	var existing sandboxv1.Sandbox
	err := r.cli.Get(ctx, types.NamespacedName{Namespace: r.namespace, Name: name}, &existing)
	switch {
	case err == nil:
		if existing.Labels[labelSpecHash] != hash {
			return runtime.AgentHandle{}, runtime.ErrSpecDrift
		}
		return r.handle(name, spec.Name, spec.TeamID), nil
	case !apierrors.IsNotFound(err):
		return runtime.AgentHandle{}, fmt.Errorf("k8s: get sandbox %s: %w", name, err)
	}

	sb := &sandboxv1.Sandbox{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: r.namespace,
			Labels: map[string]string{
				labelManaged:  "true",
				labelAgent:    spec.Name,
				labelTeam:     spec.TeamID,
				labelSpecHash: hash,
			},
		},
		Spec: sandboxv1.SandboxSpec{
			PodTemplate: sandboxv1.PodTemplate{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{agentContainer(spec)},
				},
			},
		},
	}
	if err := r.cli.Create(ctx, sb); err != nil {
		return runtime.AgentHandle{}, fmt.Errorf("k8s: create sandbox %s: %w", name, err)
	}
	return r.handle(name, spec.Name, spec.TeamID), nil
}

// Stop deletes the Sandbox CR; the controller reaps the backing pod. Missing
// Sandboxes are a no-op so Stop is idempotent.
func (r *Runtime) Stop(ctx context.Context, h runtime.AgentHandle) error {
	sb := &sandboxv1.Sandbox{ObjectMeta: metav1.ObjectMeta{Name: h.ID, Namespace: r.namespace}}
	if err := r.cli.Delete(ctx, sb); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("k8s: delete sandbox %s: %w", h.Name, err)
	}
	return nil
}

// Status fetches the Sandbox and maps its conditions to an AgentStatus.
func (r *Runtime) Status(ctx context.Context, h runtime.AgentHandle) (runtime.AgentStatus, error) {
	var sb sandboxv1.Sandbox
	if err := r.cli.Get(ctx, types.NamespacedName{Namespace: r.namespace, Name: h.ID}, &sb); err != nil {
		if apierrors.IsNotFound(err) {
			return "", runtime.ErrAgentNotFound
		}
		return "", fmt.Errorf("k8s: get sandbox %s: %w", h.Name, err)
	}
	return mapStatus(sb.Status), nil
}

// Exec runs cmd inside the agent container of the Sandbox's pod and captures its
// exit code and output. A non-zero exit is reported in ExecResult, not as a Go
// error (per the contract); err is non-nil only when the command could not run.
func (r *Runtime) Exec(ctx context.Context, h runtime.AgentHandle, cmd []string) (runtime.ExecResult, error) {
	var sb sandboxv1.Sandbox
	if err := r.cli.Get(ctx, types.NamespacedName{Namespace: r.namespace, Name: h.ID}, &sb); err != nil {
		if apierrors.IsNotFound(err) {
			return runtime.ExecResult{}, runtime.ErrAgentNotFound
		}
		return runtime.ExecResult{}, fmt.Errorf("k8s: get sandbox %s: %w", h.Name, err)
	}

	// The pod usually shares the Sandbox's name; a Sandbox adopted from a warm
	// pool records its pod name in an annotation.
	podName := h.ID
	if ann := sb.Annotations[sandboxv1.SandboxPodNameAnnotation]; ann != "" {
		podName = ann
	}

	req := r.core.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(r.namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: agentContainerName,
			Command:   cmd,
			Stdout:    true,
			Stderr:    true,
		}, kscheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(r.restConfig, "POST", req.URL())
	if err != nil {
		return runtime.ExecResult{}, fmt.Errorf("k8s: exec executor %s: %w", h.Name, err)
	}

	var stdout, stderr bytes.Buffer
	streamErr := executor.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: &stdout, Stderr: &stderr})
	exitCode := 0
	if streamErr != nil {
		var exitErr utilexec.ExitError
		if errors.As(streamErr, &exitErr) {
			exitCode = exitErr.ExitStatus()
		} else {
			return runtime.ExecResult{}, fmt.Errorf("k8s: exec %s: %w", h.Name, streamErr)
		}
	}
	return runtime.ExecResult{ExitCode: exitCode, Stdout: stdout.String(), Stderr: stderr.String()}, nil
}

func (r *Runtime) handle(name, agent, team string) runtime.AgentHandle {
	return runtime.AgentHandle{ID: name, Name: agent, TeamID: team, Substrate: runtime.SubstrateKubernetes}
}

// sandboxName is the deterministic Sandbox (and pod) name for an agent identity.
// It mirrors the Docker adapter's containerName; inputs are expected to be
// DNS-1123 safe (the agent-name contract guarantees it).
func sandboxName(agent, team string) string {
	return fmt.Sprintf("iris-%s-%s", team, agent)
}

// agentContainer builds the agent's container from the substrate-neutral spec.
func agentContainer(spec runtime.AgentSpec) corev1.Container {
	c := corev1.Container{
		Name:    agentContainerName,
		Image:   spec.Image,
		Command: spec.Command, // nil → image's baked-in entrypoint
		Env:     envVars(spec.Env),
	}
	if rl := resourceList(spec.Resources); len(rl) > 0 {
		c.Resources = corev1.ResourceRequirements{Requests: rl, Limits: rl}
	}
	return c
}

// envVars converts the env map to a sorted EnvVar slice for deterministic output.
func envVars(env map[string]string) []corev1.EnvVar {
	if len(env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]corev1.EnvVar, 0, len(env))
	for _, k := range keys {
		out = append(out, corev1.EnvVar{Name: k, Value: env[k]})
	}
	return out
}

// resourceList maps the substrate-neutral ResourceRequest onto a Kubernetes
// ResourceList. Zero fields are omitted (substrate default). CPUMillis →
// milli-CPU, MemoryBytes → bytes, GPUs → nvidia.com/gpu device count.
func resourceList(rr runtime.ResourceRequest) corev1.ResourceList {
	rl := corev1.ResourceList{}
	if rr.CPUMillis > 0 {
		rl[corev1.ResourceCPU] = *resource.NewMilliQuantity(int64(rr.CPUMillis), resource.DecimalSI)
	}
	if rr.MemoryBytes > 0 {
		rl[corev1.ResourceMemory] = *resource.NewQuantity(rr.MemoryBytes, resource.BinarySI)
	}
	if rr.GPUs > 0 {
		rl["nvidia.com/gpu"] = *resource.NewQuantity(int64(rr.GPUs), resource.DecimalSI)
	}
	return rl
}

// mapStatus translates a Sandbox's status conditions into an AgentStatus. The
// agent-sandbox condition taxonomy (Finished / Suspended / Ready) is ordered by
// precedence: a terminal Pod outcome wins over readiness, and absence of any
// positive signal means the agent is still coming up.
func mapStatus(st sandboxv1.SandboxStatus) runtime.AgentStatus {
	// Finished is terminal: distinguish a clean exit from a failure.
	if c := meta.FindStatusCondition(st.Conditions, sandboxv1.SandboxConditionFinished.String()); c != nil && c.Status == metav1.ConditionTrue {
		if c.Reason == sandboxv1.SandboxReasonPodFailed {
			return runtime.StatusFailed
		}
		return runtime.StatusStopped
	}
	// Administratively suspended: the pod is being or has been torn down.
	if c := meta.FindStatusCondition(st.Conditions, sandboxv1.SandboxConditionSuspended.String()); c != nil && c.Status == metav1.ConditionTrue {
		return runtime.StatusStopping
	}
	// Ready=True is the only signal that the agent is actually running.
	if c := meta.FindStatusCondition(st.Conditions, sandboxv1.SandboxConditionReady.String()); c != nil && c.Status == metav1.ConditionTrue {
		return runtime.StatusRunning
	}
	// No terminal, suspended or ready signal yet: still provisioning.
	return runtime.StatusStarting
}
