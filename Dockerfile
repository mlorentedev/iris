# syntax=docker/dockerfile:1
# SPDX-License-Identifier: Apache-2.0
# Copyright 2026 Manu Lorente
#
# Multi-stage build (per bootstrap-contract section 4 / ADR-002): compile a
# fully static binary, then ship it on distroless/static:nonroot — no shell, no
# package manager, non-root by default. modernc SQLite is pure-Go, so the binary
# needs no libc and CGO stays disabled.

FROM golang:1.26-alpine AS build
WORKDIR /src

# Download modules in their own layer so source edits don't bust the cache.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build metadata, injected by `make docker-build`.
ARG VERSION=0.0.0-dev
ARG GIT_SHA=unknown
ARG BUILD_DATE=unknown

# CGO_ENABLED=0 -> static binary for distroless/static. -trimpath drops local
# paths; -s -w strip the symbol table and DWARF to shrink the image.
RUN CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.gitSHA=${GIT_SHA} -X main.buildDate=${BUILD_DATE}" \
      -o /motor ./cmd/motor

# Pre-create the data dir owned by the distroless nonroot uid (65532) so the
# motor can write its SQLite DB when /data is an ephemeral or mounted volume.
RUN install -d -o 65532 -g 65532 /data

FROM gcr.io/distroless/static:nonroot
COPY --from=build /motor /motor
COPY --from=build --chown=65532:65532 /data /data

# SQLite DB lives under the writable /data volume (mounted in SDD-034g deploy).
ENV IRIS_DB_PATH=/data/iris.db
EXPOSE 8080
USER nonroot:nonroot
# CMD (not ENTRYPOINT) so subcommands are passed as the full argv, matching the
# verification form `docker run iris:dev /motor --version` and the K8s deploy
# (which sets command explicitly). Bare `docker run iris:dev` starts the server.
CMD ["/motor"]
