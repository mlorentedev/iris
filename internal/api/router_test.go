// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Manu Lorente

package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mlorentedev/iris/internal/api"
)

func TestHealthEndpoints(t *testing.T) {
	srv := httptest.NewServer(api.NewRouter())
	defer srv.Close()

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantKey    string // a key the JSON body must contain
	}{
		{"healthz is liveness", "/healthz", http.StatusOK, "status"},
		{"readyz is readiness", "/readyz", http.StatusOK, "status"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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

			var body map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if _, ok := body[tc.wantKey]; !ok {
				t.Errorf("body missing key %q: got %v", tc.wantKey, body)
			}
		})
	}
}
