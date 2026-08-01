#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
suffix="$$-$(od -An -N4 -tu4 /dev/urandom | tr -d ' ')"
sentinel_network="social-app-it-sentinel-network-$suffix"
sentinel_volume="social-app-it-sentinel-volume-$suffix"
first_pid=""
second_pid=""

cleanup() {
  [ -n "$first_pid" ] && kill "$first_pid" 2>/dev/null || true
  [ -n "$second_pid" ] && kill "$second_pid" 2>/dev/null || true
  docker network rm "$sentinel_network" >/dev/null 2>&1 || true
  docker volume rm "$sentinel_volume" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

docker network create "$sentinel_network" >/dev/null
docker volume create "$sentinel_volume" >/dev/null

QUALITY_UPDATE_LATEST=0 bash "$ROOT/scripts/quality/run_integration.sh" >"$ROOT/quality/integration/parallel-first.log" 2>&1 &
first_pid=$!
QUALITY_UPDATE_LATEST=0 bash "$ROOT/scripts/quality/run_integration.sh" >"$ROOT/quality/integration/parallel-second.log" 2>&1 &
second_pid=$!

first_rc=0
second_rc=0
wait "$first_pid" || first_rc=$?
first_pid=""
wait "$second_pid" || second_rc=$?
second_pid=""
[ "$first_rc" -eq 0 ] && [ "$second_rc" -eq 0 ] || { echo "parallel integration failed: $first_rc/$second_rc" >&2; exit 2; }

docker network inspect "$sentinel_network" >/dev/null
docker volume inspect "$sentinel_volume" >/dev/null

if docker ps -a --format '{{.Names}}' | grep -q '^social-app-it-'; then
  echo "parallel integration left containers behind" >&2
  exit 3
fi
remaining_networks=$(docker network ls --format '{{.Name}}' | grep '^social-app-it-' | grep -v "^${sentinel_network}$" || true)
remaining_volumes=$(docker volume ls --format '{{.Name}}' | grep '^social-app-it-' | grep -v "^${sentinel_volume}$" || true)
if [ -n "$remaining_networks" ] || [ -n "$remaining_volumes" ]; then
  echo "parallel integration left networks or volumes behind" >&2
  exit 4
fi

printf '{"parallel_runs":2,"sentinel_preserved":true}\n' >"$ROOT/quality/integration/isolation-summary.json"
