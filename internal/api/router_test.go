// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mlorentedev/iris/internal/api"
)

// fakePinger stands in for *sql.DB so the readiness probe can be exercised in
// both the healthy and failing directions without a real database.
type fakePinger struct{ err error }

func (f fakePinger) PingContext(context.Context) error { return f.err }

func TestHealthEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		ping       error // nil = DB healthy
		wantStatus int
		wantKey    string // a key the JSON body must contain
		wantVal    string // expected value for wantKey
	}{
		{"healthz is liveness", "/healthz", nil, http.StatusOK, "status", "ok"},
		{"readyz ok when db up", "/readyz", nil, http.StatusOK, "db", "ok"},
		{"readyz 503 when db down", "/readyz", errors.New("dial fail"), http.StatusServiceUnavailable, "db", "down"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(api.NewRouter(fakePinger{err: tc.ping}))
			defer srv.Close()

			resp, err := http.Get(srv.URL + tc.path)
			if err != nil {
				t.Fatalf("GET %s: %v", tc.path, err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
			if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			var body map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got := body[tc.wantKey]; got != tc.wantVal {
				t.Errorf("body[%q] = %q, want %q (body=%v)", tc.wantKey, got, tc.wantVal, body)
			}
		})
	}
}
