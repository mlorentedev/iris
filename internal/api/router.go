// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

// Package api wires the motor's HTTP surface: a chi router exposing operational
// endpoints (health) at the root, with the operator API to land under /api/v1
// in later tasks (per ADR-002).
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds the motor's HTTP handler.
func NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	// Operational probes (Kubernetes liveness/readiness). No /api/v1 prefix —
	// these are infrastructure endpoints, not the operator API.
	r.Get("/healthz", handleHealthz)
	r.Get("/readyz", handleReadyz)

	return r
}

// handleHealthz is the liveness probe: the process is up and serving.
func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReadyz is the readiness probe. v0 skeleton: no subsystems are wired yet.
// DB (Task 3) and NATS (SDD-034b) will register their checks here as they land;
// until then readiness is unconditional rather than a hardcoded {"db":"ok"} lie.
func handleReadyz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", "err", err)
	}
}
