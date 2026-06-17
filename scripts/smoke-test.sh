#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Copyright 2026 Manu Lorente
#
# iris smoke-test: the 5 substrate checks from bootstrap-contract section 2.
# Exit 0 iff the substrate works end-to-end. On failure, exit at the first
# broken check with a three-element (ERROR / WHY / FIX) message to stderr.
#
# Prerequisite: `make dev` running (motor native + NATS/worker containers up).
set -uo pipefail

MOTOR_URL="${MOTOR_URL:-http://localhost:8080}"
NATS_URL="${NATS_URL:-localhost:4223}"
COMPOSE=(docker compose -f compose.dev.yml)

fail() { # $1=check number  $2=what broke  $3=fix hint
  echo "[smoke] failed at check $1: $2" >&2
  echo "  WHY: the check command returned non-zero or unexpected output" >&2
  echo "  FIX: $3" >&2
  exit 1
}

# 1. Motor HTTP up (liveness).
curl -fsS "$MOTOR_URL/healthz" 2>/dev/null | grep -q '"status":"ok"' \
  || fail 1 "motor /healthz not serving ok" "is 'make dev' running? check the air logs"

# 2. Motor ready: DB migrated. The "nats":"ok" key joins this body in SDD-034b;
#    until then readiness is DB-only, so we assert exactly that.
curl -fsS "$MOTOR_URL/readyz" 2>/dev/null | grep -q '"db":"ok"' \
  || fail 2 "motor /readyz reports db not ready" "is IRIS_DB_PATH writable? did migrations apply on boot?"

# 3. NATS reachable from host.
nats --server "$NATS_URL" server check connection >/dev/null 2>&1 \
  || fail 3 "NATS not reachable on $NATS_URL" "is the nats container healthy? '${COMPOSE[*]} ps'"

# 4. Worker pi container running (the real SDD-034d worker, not a stub).
cid="$("${COMPOSE[@]}" ps -q worker-pi 2>/dev/null)"
status="$([ -n "$cid" ] && docker inspect --format '{{.State.Status}}' "$cid" 2>/dev/null)"
[ "$status" = "running" ] \
  || fail 4 "worker-pi not running (status=${status:-absent})" "'${COMPOSE[*]} up -d worker-pi'"

# 5. End-to-end NATS pub/sub roundtrip. The subscriber must be listening before
#    the publish (core NATS is fire-and-forget), so we background the sub first.
out="$(mktemp)"
timeout 5 nats --server "$NATS_URL" sub iris.smoke --count 1 >"$out" 2>/dev/null &
subpid=$!
sleep 0.7
nats --server "$NATS_URL" pub iris.smoke "ping" >/dev/null 2>&1
wait "$subpid" 2>/dev/null
if ! grep -q "ping" "$out"; then
  rm -f "$out"
  fail 5 "NATS pub/sub roundtrip failed" "is JetStream enabled (-js) on the nats container?"
fi
rm -f "$out"

echo "[smoke] 5/5 checks passed"
