# syntax=docker/dockerfile:1
# SPDX-License-Identifier: Apache-2.0
# Copyright 2026 Manu Lorente
#
# iris fleet worker image (ADR-009). Unlike the motor's distroless image, the
# worker ships on a Node base: pi is npm-only (@earendil-works/pi-coding-agent,
# no standalone binary), so the runtime needs Node + git (for per-job worktrees).
# Multi-stage: compile the static Go supervisor, then assemble Node + pinned pi +
# the Go binary. pi is installed --ignore-scripts and pinned; upgrades are
# deliberate dependency bumps.

# --- Stage 1: build the Go worker (static, like the motor) -------------------
FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=0.0.0-dev
ARG GIT_SHA=unknown
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.gitSHA=${GIT_SHA} -X main.buildDate=${BUILD_DATE}" \
      -o /worker ./cmd/worker

# --- Stage 2: Node runtime with pinned pi + git ------------------------------
# node:22-alpine satisfies pi's engines (>=22.19.0). git is required for the
# per-job worktree lifecycle; ca-certificates for the provider HTTPS calls pi makes.
FROM node:22-alpine

# Pin pi; bumping this ARG is the upgrade path (pinned + tested per ADR-009).
ARG PI_VERSION=0.79.6

RUN apk add --no-cache git ca-certificates \
 && npm install -g --ignore-scripts @earendil-works/pi-coding-agent@${PI_VERSION} \
 && npm cache clean --force

COPY --from=build /worker /usr/local/bin/worker

# Run as the unprivileged `node` user that the base image ships (uid 1000).
# Pre-create the worktree base owned by it so the worker can write at runtime.
ENV WORKER_WORKTREE_BASE=/home/node/worktrees
RUN mkdir -p /home/node/worktrees && chown -R node:node /home/node/worktrees
USER node

# Bare `docker run` starts the supervisor; `worker --version` works for probes.
ENTRYPOINT ["worker"]
