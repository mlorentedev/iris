// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// healthzOutput is the liveness response body: {"status":"ok"}.
type healthzOutput struct {
	Body struct {
		Status string `json:"status" example:"ok" doc:"Liveness status"`
	}
}

// readyzOutput is the readiness response. Status is set dynamically by the
// handler: 200 while the database answers, 503 otherwise — so Kubernetes pulls
// the pod from rotation when a dependency is unhealthy.
type readyzOutput struct {
	Status int
	Body   struct {
		DB string `json:"db" example:"ok" doc:"Database readiness (ok|down)"`
	}
}

// registerHealth wires the Kubernetes liveness/readiness probes as huma
// operations so they appear in the OpenAPI spec. They live at the root (no
// /api/v1 prefix) because they are infrastructure endpoints, not the operator
// API.
func registerHealth(api huma.API, db Pinger) {
	huma.Register(api, huma.Operation{
		OperationID: "get-healthz",
		Method:      http.MethodGet,
		Path:        "/healthz",
		Summary:     "Liveness probe",
		Description: "Reports that the process is up and serving. Dependency-free: a liveness failure restarts the pod, which would never fix a downstream outage.",
		Tags:        []string{"ops"},
	}, func(_ context.Context, _ *struct{}) (*healthzOutput, error) {
		out := &healthzOutput{}
		out.Body.Status = "ok"
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-readyz",
		Method:      http.MethodGet,
		Path:        "/readyz",
		Summary:     "Readiness probe",
		Description: "Reports 200 {\"db\":\"ok\"} while the database is reachable, else 503 {\"db\":\"down\"}. The \"nats\" key joins this body when the NATS client lands in SDD-034b.",
		Tags:        []string{"ops"},
	}, func(ctx context.Context, _ *struct{}) (*readyzOutput, error) {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		out := &readyzOutput{}
		if err := db.PingContext(pingCtx); err != nil {
			slog.Warn("readiness check failed", "dep", "db", "err", err)
			out.Status = http.StatusServiceUnavailable
			out.Body.DB = "down"
			return out, nil
		}
		out.Status = http.StatusOK
		out.Body.DB = "ok"
		return out, nil
	})
}
