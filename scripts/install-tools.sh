#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Copyright 2026 Manu Lorente
#
# Download pinned dev tools into ./bin (idempotent). Versions come from the
# Makefile (single source of truth) via the environment. Re-run with FORCE=1 to
# refresh after a version bump.
#
# Tools: sqlc (codegen) · nats (smoke-test) · tailwindcss (console CSS, SDD-034e)
# · air (motor hot-reload). Each upstream names its assets differently, so the
# OS/ARCH mapping is per-tool. HTTPS + pinned versions; no upstream checksums.
set -euo pipefail

: "${SQLC_VERSION:?set by Makefile}"
: "${NATS_VERSION:?set by Makefile}"
: "${TAILWIND_VERSION:?set by Makefile}"
: "${AIR_VERSION:?set by Makefile}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="$ROOT/bin"
mkdir -p "$BIN"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"   # linux | darwin
march="$(uname -m)"
case "$march" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "unsupported arch: $march" >&2; exit 1 ;;
esac

have() { # tool already present (and not forced)?
  [ -x "$BIN/$1" ] && [ "${FORCE:-0}" != "1" ]
}
log() { printf '>> %s\n' "$*"; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

install_sqlc() {
  have sqlc && { log "sqlc present (v$SQLC_VERSION)"; return; }
  log "sqlc v$SQLC_VERSION"
  curl -fsSL "https://github.com/sqlc-dev/sqlc/releases/download/v${SQLC_VERSION}/sqlc_${SQLC_VERSION}_${os}_${arch}.tar.gz" \
    | tar -xz -C "$tmp" sqlc
  install -m 0755 "$tmp/sqlc" "$BIN/sqlc"
}

install_nats() {
  have nats && { log "nats present (v$NATS_VERSION)"; return; }
  log "nats v$NATS_VERSION"
  local name="nats-${NATS_VERSION}-${os}-${arch}"
  curl -fsSL "https://github.com/nats-io/natscli/releases/download/v${NATS_VERSION}/${name}.zip" -o "$tmp/nats.zip"
  ( cd "$tmp" && unzip -qo nats.zip )
  install -m 0755 "$tmp/${name}/nats" "$BIN/nats"
}

install_tailwind() {
  have tailwindcss && { log "tailwindcss present (v$TAILWIND_VERSION)"; return; }
  log "tailwindcss v$TAILWIND_VERSION"
  # tailwind names assets os=linux|macos, arch=x64|arm64 (unlike the others).
  local tw_os="$os" tw_arch="$arch"
  [ "$tw_os" = "darwin" ] && tw_os=macos
  [ "$tw_arch" = "amd64" ] && tw_arch=x64
  curl -fsSL "https://github.com/tailwindlabs/tailwindcss/releases/download/v${TAILWIND_VERSION}/tailwindcss-${tw_os}-${tw_arch}" \
    -o "$BIN/tailwindcss"
  chmod 0755 "$BIN/tailwindcss"
}

install_air() {
  have air && { log "air present (v$AIR_VERSION)"; return; }
  log "air v$AIR_VERSION"
  curl -fsSL "https://github.com/air-verse/air/releases/download/v${AIR_VERSION}/air_${AIR_VERSION}_${os}_${arch}" \
    -o "$BIN/air"
  chmod 0755 "$BIN/air"
}

install_sqlc
install_nats
install_tailwind
install_air

echo "[install-tools] ready in ./bin: sqlc nats tailwindcss air"
