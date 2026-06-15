// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

// Package api wires the motor's HTTP surface: a chi router exposing operational
// endpoints (health) at the root, with the operator API to land under /api/v1
// in later tasks (per ADR-002).
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// pinger is the slice of *sql.DB that the readiness probe needs. Depending on an
// interface (not the concrete handle) keeps package api decoupled from package
// db and lets tests inject a healthy or failing backend without a real socket.
type pinger interface {
	PingContext(ctx context.Context) error
}

// NewRouter builds the motor's HTTP handler. db backs the readiness probe.
func NewRouter(db pinger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	// Operational probes (Kubernetes liveness/readiness). No /api/v1 prefix —
	// these are infrastructure endpoints, not the operator API.
	r.Get("/healthz", handleHealthz)
	r.Get("/readyz", handleReadyz(db))

	return r
}

// handleHealthz is the liveness probe: the process is up and serving. It must
// stay dependency-free — a liveness failure tells Kubernetes to restart the
// pod, which would never fix a downstream (DB/NATS) outage.
func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReadyz is the readiness probe: serves 200 {"db":"ok"} only while the
// database answers a ping, else 503 {"db":"down"} so Kubernetes pulls the pod
// from rotation until its dependencies recover. NATS will add its own key here
// in SDD-034b; the readyz body grows one dependency at a time as they land.
func handleReadyz(db pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			slog.Warn("readiness check failed", "dep", "db", "err", err)
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"db": "down"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"db": "ok"})
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", "err", err)
	}
}
