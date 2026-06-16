// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sandboxv1 "sigs.k8s.io/agent-sandbox/api/v1beta1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mlorentedev/iris/internal/runtime"
	"github.com/mlorentedev/iris/internal/runtime/runtimetest"
)

func TestSandboxName(t *testing.T) {
	if got := sandboxName("alpha", "team-1"); got != "iris-team-1-alpha" {
		t.Errorf("sandboxName = %q, want iris-team-1-alpha", got)
	}
}

func TestEnvVars(t *testing.T) {
	got := envVars(map[string]string{"B": "2", "A": "1"})
	if len(got) != 2 || got[0] != (corev1.EnvVar{Name: "A", Value: "1"}) || got[1] != (corev1.EnvVar{Name: "B", Value: "2"}) {
		t.Errorf("envVars = %v, want [A=1 B=2] (sorted)", got)
	}
	if envVars(nil) != nil {
		t.Error("envVars(nil) must be nil")
	}
}

func TestResourceList(t *testing.T) {
	rl := resourceList(runtime.ResourceRequest{CPUMillis: 250, MemoryBytes: 64 << 20, GPUs: 1})
	if q := rl[corev1.ResourceCPU]; q.MilliValue() != 250 {
		t.Errorf("cpu = %s, want 250m", q.String())
	}
	if q := rl[corev1.ResourceMemory]; q.Value() != 64<<20 {
		t.Errorf("memory = %s, want %d bytes", q.String(), 64<<20)
	}
	if q := rl["nvidia.com/gpu"]; q.Value() != 1 {
		t.Errorf("gpu = %s, want 1", q.String())
	}
	if len(resourceList(runtime.ResourceRequest{})) != 0 {
		t.Error("empty ResourceRequest must yield an empty ResourceList (substrate default)")
	}
}

func TestMapStatus(t *testing.T) {
	cond := func(ct sandboxv1.ConditionType, status metav1.ConditionStatus, reason string) metav1.Condition {
		return metav1.Condition{Type: ct.String(), Status: status, Reason: reason}
	}
	cases := []struct {
		name string
		in   sandboxv1.SandboxStatus
		want runtime.AgentStatus
	}{
		{"no conditions", sandboxv1.SandboxStatus{}, runtime.StatusStarting},
		{"ready true", sandboxv1.SandboxStatus{Conditions: []metav1.Condition{
			cond(sandboxv1.SandboxConditionReady, metav1.ConditionTrue, sandboxv1.SandboxReasonDependenciesReady),
		}}, runtime.StatusRunning},
		{"ready false", sandboxv1.SandboxStatus{Conditions: []metav1.Condition{
			cond(sandboxv1.SandboxConditionReady, metav1.ConditionFalse, sandboxv1.SandboxReasonDependenciesNotReady),
		}}, runtime.StatusStarting},
		{"finished succeeded", sandboxv1.SandboxStatus{Conditions: []metav1.Condition{
			cond(sandboxv1.SandboxConditionFinished, metav1.ConditionTrue, sandboxv1.SandboxReasonPodSucceeded),
		}}, runtime.StatusStopped},
		{"finished failed", sandboxv1.SandboxStatus{Conditions: []metav1.Condition{
			cond(sandboxv1.SandboxConditionFinished, metav1.ConditionTrue, sandboxv1.SandboxReasonPodFailed),
		}}, runtime.StatusFailed},
		{"suspended", sandboxv1.SandboxStatus{Conditions: []metav1.Condition{
			cond(sandboxv1.SandboxConditionSuspended, metav1.ConditionTrue, sandboxv1.SandboxReasonSuspended),
		}}, runtime.StatusStopping},
		{"finished beats ready", sandboxv1.SandboxStatus{Conditions: []metav1.Condition{
			cond(sandboxv1.SandboxConditionReady, metav1.ConditionTrue, sandboxv1.SandboxReasonDependenciesReady),
			cond(sandboxv1.SandboxConditionFinished, metav1.ConditionTrue, sandboxv1.SandboxReasonPodFailed),
		}}, runtime.StatusFailed},
	}
	for _, c := range cases {
		if got := mapStatus(c.in); got != c.want {
			t.Errorf("mapStatus(%s) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestAgentContainer(t *testing.T) {
	c := agentContainer(runtime.AgentSpec{
		Image:   "busybox:1.37",
		Command: []string{"sleep", "3600"},
		Env:     map[string]string{"FOO": "bar"},
	})
	if c.Name != agentContainerName {
		t.Errorf("container name = %q, want %q", c.Name, agentContainerName)
	}
	if c.Image != "busybox:1.37" {
		t.Errorf("image = %q, want busybox:1.37", c.Image)
	}
	if len(c.Command) != 2 || c.Command[0] != "sleep" {
		t.Errorf("command = %v, want [sleep 3600]", c.Command)
	}
}

func TestUpsertContainer(t *testing.T) {
	pod := &corev1.PodSpec{Containers: []corev1.Container{{Name: "agent", Image: "a"}}}
	upsertContainer(pod, corev1.Container{Name: "logger", Image: "b"})
	if len(pod.Containers) != 2 {
		t.Fatalf("after append: %d containers, want 2", len(pod.Containers))
	}
	upsertContainer(pod, corev1.Container{Name: "logger", Image: "c"})
	if len(pod.Containers) != 2 {
		t.Fatalf("after upsert same name: %d containers, want 2 (replace, not duplicate)", len(pod.Containers))
	}
	if pod.Containers[1].Image != "c" {
		t.Errorf("logger image = %q, want c (replaced in place)", pod.Containers[1].Image)
	}
}

// testNamespace is where the conformance suite provisions agents on a real
// cluster. "default" always exists, so the skip-gated suite needs no setup.
const testNamespace = "default"

// requireCluster returns a live K8s adapter or skips the test. It pings the
// cluster by listing Sandboxes; a missing kubeconfig, unreachable apiserver or
// uninstalled agent-sandbox CRD all skip (not fail), so the package still
// unit-tests in restricted CI — mirroring the Docker adapter's requireDaemon.
func requireCluster(t *testing.T) *Runtime {
	t.Helper()
	rt, err := New(testNamespace)
	if err != nil {
		t.Skipf("k8s config unavailable: %v", err)
	}
	var list sandboxv1.SandboxList
	if err := rt.cli.List(context.Background(), &list, ctrlclient.InNamespace(testNamespace)); err != nil {
		t.Skipf("k8s cluster / agent-sandbox CRD unavailable: %v", err)
	}
	return rt
}

// TestK8sConformance runs the shared AgentRuntime conformance suite against a
// real cluster with agent-sandbox installed. It skips when none is reachable.
func TestK8sConformance(t *testing.T) {
	rt := requireCluster(t)
	runtimetest.RunConformance(t, func() runtime.AgentRuntime { return rt }, runtimetest.Options{
		NewSpec: func(name, teamID string) runtime.AgentSpec {
			return runtime.AgentSpec{
				Name:      name,
				TeamID:    teamID,
				Image:     "busybox:1.37",
				Command:   []string{"sleep", "3600"},
				Harness:   runtime.HarnessPi,
				Resources: runtime.ResourceRequest{CPUMillis: 250, MemoryBytes: 64 << 20},
			}
		},
		ExecProbe: []string{"true"},
	})
}
