#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Copyright 2026 Manu Lorente
#
# Push the per-registry image tags built by `make docker-build`. Each registry
# logs in with its own secret; a registry whose secret is missing is skipped
# with a warning rather than failing the whole push (v0: Docker Hub secrets are
# added to the repo after merge). ghcr.io uses GITHUB_TOKEN, docker.io uses
# DOCKER_HUB_USERNAME + DOCKER_HUB_TOKEN.
set -uo pipefail

VERSION="${VERSION:?set by Makefile}"
GIT_SHA="${GIT_SHA:?set by Makefile}"
REGISTRIES="${REGISTRIES:?set by Makefile}"

login_registry() { # $1 = registry host
  case "$1" in
    ghcr.io)
      [ -n "${GITHUB_TOKEN:-}" ] || return 1
      echo "$GITHUB_TOKEN" | docker login ghcr.io -u "${GITHUB_ACTOR:-mlorentedev}" --password-stdin
      ;;
    docker.io | index.docker.io)
      { [ -n "${DOCKER_HUB_USERNAME:-}" ] && [ -n "${DOCKER_HUB_TOKEN:-}" ]; } || return 1
      echo "$DOCKER_HUB_TOKEN" | docker login -u "$DOCKER_HUB_USERNAME" --password-stdin
      ;;
    *)
      # Unknown registry (e.g. a future self-hosted registry.kubelab.live):
      # assume an ambient login is already in place and try to push.
      return 0
      ;;
  esac
}

pushed=0
skipped=0
for repo in $REGISTRIES; do
  host="${repo%%/*}"
  if login_registry "$host"; then
    docker push "$repo:$GIT_SHA"
    docker push "$repo:$VERSION"
    pushed=$((pushed + 1))
  else
    echo "WARN: no credentials for $host -> skipping $repo (set its login secret to enable)" >&2
    skipped=$((skipped + 1))
  fi
done

echo "[docker-push] pushed=$pushed skipped=$skipped"
