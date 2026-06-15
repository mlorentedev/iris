// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

// Package api wires the motor's HTTP surface: a chi router fronted by huma,
// which generates the OpenAPI 3.1 spec from typed operations (per ADR-002 +
// bootstrap-contract section 3). huma also serves /openapi.yaml, /docs and
// /schemas. The operator API lands under /api/v1 in later tasks.
package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// apiVersion is the OpenAPI document version. It describes the API contract, is
// independent of the binary build version, and is held constant so `make
// api-docs` is deterministic (no drift between local and CI).
const apiVersion = "0.1.0-dev"

// Pinger is the slice of *sql.DB the readiness probe needs. Depending on an
// interface (not the concrete handle) keeps package api decoupled from package
// db and lets tests inject a healthy or failing backend without a real socket.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// New builds the motor's HTTP handler and the huma API that documents it. db
// backs the readiness probe.
func New(db Pinger) (http.Handler, huma.API) {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	config := huma.DefaultConfig("iris motor", apiVersion)
	// DefaultConfig injects a $schema link field into every response body. The
	// operational probes have an exact, frozen body contract ({"status":"ok"} /
	// {"db":"ok"}), so drop the transformer.
	config.Transformers = nil
	api := humachi.New(r, config)

	registerHealth(api, db)
	return r, api
}

// OpenAPIYAML returns the motor's OpenAPI 3.1 spec as YAML, for `motor openapi`
// / `make api-docs`. Handlers are never invoked during generation, so a noop
// pinger suffices.
func OpenAPIYAML() ([]byte, error) {
	_, api := New(noopPinger{})
	return api.OpenAPI().YAML()
}

type noopPinger struct{}

func (noopPinger) PingContext(context.Context) error { return nil }
