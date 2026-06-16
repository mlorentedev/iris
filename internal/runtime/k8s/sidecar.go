// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	sandboxv1 "sigs.k8s.io/agent-sandbox/api/v1beta1"

	"github.com/mlorentedev/iris/internal/runtime"
)

// Compile-time proof the K8s adapter advertises the sidecar capability. This is
// the one capability that is K8s-only by nature (a shared Pod); Docker and
// local-process run one container per agent.
var _ runtime.SidecarManager = (*Runtime)(nil)

// DeploySidecar co-locates an auxiliary container in the agent's Sandbox pod by
// appending it to the Sandbox's PodTemplate. Idempotent on the sidecar's name: a
// sidecar already present is replaced in place rather than duplicated, so calling
// it twice with the same spec is a no-op the controller can reconcile.
func (r *Runtime) DeploySidecar(ctx context.Context, h runtime.AgentHandle, sc runtime.SidecarSpec) error {
	key := types.NamespacedName{Namespace: r.namespace, Name: h.ID}
	var sb sandboxv1.Sandbox
	if err := r.cli.Get(ctx, key, &sb); err != nil {
		if apierrors.IsNotFound(err) {
			return runtime.ErrAgentNotFound
		}
		return fmt.Errorf("k8s: get sandbox %s: %w", h.Name, err)
	}

	sidecar := corev1.Container{
		Name:  sc.Name,
		Image: sc.Image,
		Env:   envVars(sc.Env),
	}
	if rl := resourceList(sc.Resources); len(rl) > 0 {
		sidecar.Resources = corev1.ResourceRequirements{Requests: rl, Limits: rl}
	}

	upsertContainer(&sb.Spec.PodTemplate.Spec, sidecar)
	if err := r.cli.Update(ctx, &sb); err != nil {
		return fmt.Errorf("k8s: attach sidecar %s to %s: %w", sc.Name, h.Name, err)
	}
	return nil
}

// upsertContainer appends c, or replaces an existing container with the same
// name in place — keeping DeploySidecar idempotent.
func upsertContainer(pod *corev1.PodSpec, c corev1.Container) {
	for i := range pod.Containers {
		if pod.Containers[i].Name == c.Name {
			pod.Containers[i] = c
			return
		}
	}
	pod.Containers = append(pod.Containers, c)
}
