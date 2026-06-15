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

# sqlc is a prebuilt-binary build tool, deliberately kept OUT of go.mod (a
# `go tool` directive would merge sqlc's whole dependency tree into the runtime
# module). SQLC_VERSION is the single declared pin: CI provisions this exact
# version via sqlc-dev/setup-sqlc, and `make install-tools` (Task 4) will land
# it locally. Override SQLC to point at a non-PATH binary if needed.
SQLC_VERSION ?= 1.31.1
SQLC ?= sqlc

.DEFAULT_GOAL := help

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

# --- Meta --------------------------------------------------------------------
.PHONY: help
help: ## List available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'
