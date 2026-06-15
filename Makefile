# SPDX-License-Identifier: Apache-2.0
# Copyright 2026 Manu Lorente
#
# iris motor — developer & CI tasks. Targets land incrementally across the
# SDD-034 bootstrap stream; this file starts with the DB layer (Task 3). Task 4
# extends it with dev / install-tools / smoke-test / docker-build / api-docs.

# --- Configuration -----------------------------------------------------------
# Overridable per invocation: `make migrate-up IRIS_DB_PATH=/tmp/scratch.db`.
IRIS_DB_PATH ?= iris.db
export IRIS_DB_PATH

# `make install-tools` lands pinned tool binaries in ./bin; put it first on PATH
# so every target resolves the pinned sqlc/nats/tailwindcss/air over any global.
export PATH := $(CURDIR)/bin:$(PATH)

# Pinned dev-tool versions (single source of truth, consumed by
# scripts/install-tools.sh). Build tools, deliberately kept OUT of go.mod so
# their dependency trees never leak into the runtime module. Keep SQLC_VERSION
# in sync with the sqlc-dev/setup-sqlc step in .github/workflows/ci.yml.
SQLC_VERSION     ?= 1.31.1
NATS_VERSION     ?= 0.4.0
TAILWIND_VERSION ?= 3.4.19
AIR_VERSION      ?= 1.65.3
SQLC             ?= sqlc

# Container build metadata + tagging. VERSION is the semver pre-release tag;
# GIT_SHA is the immutable identity for GitOps. REGISTRIES is space-separated;
# `docker-build` tags every registry, `docker-push` pushes to those it can log
# into (graceful skip otherwise).
VERSION     ?= 0.1.0-dev
GIT_SHA     := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE  := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
IMAGE_LOCAL ?= iris:dev
REGISTRIES  ?= ghcr.io/mlorentedev/iris docker.io/mlorentedev/iris

.DEFAULT_GOAL := help

# --- Dev environment ---------------------------------------------------------
.PHONY: install-tools
install-tools: ## Download pinned dev tools (sqlc, nats, tailwindcss, air) into ./bin
	SQLC_VERSION=$(SQLC_VERSION) NATS_VERSION=$(NATS_VERSION) \
	TAILWIND_VERSION=$(TAILWIND_VERSION) AIR_VERSION=$(AIR_VERSION) \
	./scripts/install-tools.sh

.PHONY: dev
dev: ## Bring up the substrate (NATS + worker) and run the motor via air hot-reload
	@command -v air >/dev/null 2>&1 || { \
		echo "ERROR: air not found on PATH"; \
		echo "  WHY: the motor hot-reload loop needs air"; \
		echo "  FIX: run 'make install-tools'"; exit 1; }
	docker compose -f compose.dev.yml up -d --wait
	air -c .air.toml

.PHONY: dev-down
dev-down: ## Stop and remove the dev substrate containers
	docker compose -f compose.dev.yml down

.PHONY: smoke-test
smoke-test: ## Run the 5 substrate checks (needs 'make dev' running)
	@command -v nats >/dev/null 2>&1 || { \
		echo "ERROR: nats CLI not found on PATH"; \
		echo "  WHY: smoke-test checks 3 and 5 use the nats CLI"; \
		echo "  FIX: run 'make install-tools'"; exit 1; }
	./scripts/smoke-test.sh

# --- Database ----------------------------------------------------------------
.PHONY: migrate-up
migrate-up: ## Apply all pending migrations to $(IRIS_DB_PATH)
	go run ./cmd/motor migrate up

.PHONY: migrate-down
migrate-down: ## Roll the schema all the way back (destructive, dev-only)
	go run ./cmd/motor migrate down

.PHONY: generate
generate: ## Regenerate sqlc typed queries from SQL
	$(SQLC) generate

.PHONY: sqlc-diff
sqlc-diff: generate ## Fail if committed sqlc output drifts from a fresh generate
	git diff --exit-code internal/db/sqlc

# --- API docs ----------------------------------------------------------------
.PHONY: api-docs
api-docs: ## Regenerate docs/api.yaml (OpenAPI 3.1) from the huma routes
	@mkdir -p docs
	go run ./cmd/motor openapi > docs/api.yaml

.PHONY: api-docs-diff
api-docs-diff: api-docs ## Fail if committed docs/api.yaml drifts from a fresh generate
	git diff --exit-code docs/api.yaml

# --- Container ---------------------------------------------------------------
.PHONY: docker-build
docker-build: ## Build the image, tagging $(IMAGE_LOCAL) + <registry>:{sha,version}
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_SHA=$(GIT_SHA) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(IMAGE_LOCAL) \
		$(foreach r,$(REGISTRIES),-t $(r):$(GIT_SHA) -t $(r):$(VERSION)) \
		.

.PHONY: docker-push
docker-push: ## Push per-registry tags (logs in per registry; skips those without secrets)
	VERSION=$(VERSION) GIT_SHA=$(GIT_SHA) REGISTRIES="$(REGISTRIES)" \
	./scripts/docker-push.sh

# --- Meta --------------------------------------------------------------------
.PHONY: help
help: ## List available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'
