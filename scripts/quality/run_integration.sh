#!/usr/bin/env bash
set -uo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
COMPOSE_FILE="$ROOT/docker-compose.integration.yml"
random_suffix=$(od -An -N4 -tu4 /dev/urandom | tr -d ' ')
PROJECT="social-app-it-$$-$random_suffix"
RUN_DIR="$ROOT/quality/integration/$PROJECT"
REPORT="$RUN_DIR/integration-test.jsonl"
DOCKER_BIN=${QUALITY_DOCKER_BIN:-docker}
CURL_BIN=${QUALITY_CURL_BIN:-curl}
PYTHON_BIN=${QUALITY_PYTHON_BIN:-python3}
mkdir -p "$RUN_DIR"

primary_status=0
cleanup_status=0
cleaned=0

cleanup() {
  local incoming=$?
  if [ "$primary_status" -eq 0 ] && [ "$incoming" -ne 0 ]; then
    primary_status=$incoming
  fi
  if [ "$cleaned" -eq 0 ]; then
    cleaned=1
    "$DOCKER_BIN" compose -p "$PROJECT" -f "$COMPOSE_FILE" ps >"$RUN_DIR/compose-ps.log" 2>&1 || true
    "$DOCKER_BIN" compose -p "$PROJECT" -f "$COMPOSE_FILE" logs --no-color >"$RUN_DIR/compose.log" 2>&1 || true
    "$DOCKER_BIN" compose -p "$PROJECT" -f "$COMPOSE_FILE" down --volumes --remove-orphans >"$RUN_DIR/cleanup.log" 2>&1 || cleanup_status=$?
    "$DOCKER_BIN" image rm "${PROJECT}-backend" >>"$RUN_DIR/cleanup.log" 2>&1 || cleanup_status=$?
  fi
  if [ "$primary_status" -ne 0 ]; then
    exit "$primary_status"
  fi
  exit "$cleanup_status"
}
trap cleanup EXIT
trap 'primary_status=130; cleanup' INT
trap 'primary_status=143; cleanup' TERM

"$DOCKER_BIN" compose -p "$PROJECT" -f "$COMPOSE_FILE" up -d --build db redis backend || primary_status=$?
if [ "$primary_status" -eq 0 ]; then
  for service in db redis; do
    dependency_deadline=$((SECONDS + 90))
    health=""
    while [ "$SECONDS" -lt "$dependency_deadline" ]; do
      container=$("$DOCKER_BIN" compose -p "$PROJECT" -f "$COMPOSE_FILE" ps -q "$service" 2>/dev/null || true)
      if [ -n "$container" ]; then
        health=$("$DOCKER_BIN" inspect --format '{{.State.Health.Status}}' "$container" 2>/dev/null || true)
      fi
      [ "$health" = "healthy" ] && break
      sleep 1
    done
    if [ "$health" != "healthy" ]; then
      echo "$service health is $health" >&2
      primary_status=2
    fi
  done
fi

if [ "$primary_status" -eq 0 ]; then
  port=$("$DOCKER_BIN" compose -p "$PROJECT" -f "$COMPOSE_FILE" port backend 8080 2>/dev/null | tail -1 | awk -F: '{print $NF}')
  if ! [[ "$port" =~ ^[0-9]+$ ]] || [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
    echo "invalid backend port: $port" >&2
    primary_status=3
  fi
fi

if [ "$primary_status" -eq 0 ]; then
  deadline=$((SECONDS + 90))
  ready=0
  while [ "$SECONDS" -lt "$deadline" ]; do
    code=$("$CURL_BIN" --silent --output /dev/null --write-out '%{http_code}' --max-time 3 -X POST "http://127.0.0.1:$port/api/v1/auth/login" || true)
    if [ "$code" = "415" ]; then
      ready=1
      break
    fi
    sleep 1
  done
  if [ "$ready" -ne 1 ]; then
    echo "backend readiness timed out" >&2
    primary_status=3
  fi
fi

if [ "$primary_status" -eq 0 ]; then
  "$DOCKER_BIN" compose -p "$PROJECT" -f "$COMPOSE_FILE" run --rm tester >"$REPORT" 2>"$RUN_DIR/integration-test.stderr.log" || primary_status=$?
fi

if [ "$primary_status" -eq 0 ]; then
  "$PYTHON_BIN" "$ROOT/scripts/quality/verify_integration_report.py" "$REPORT" || primary_status=$?
fi

if [ "$primary_status" -eq 0 ] && [ "${QUALITY_UPDATE_LATEST:-1}" = "1" ]; then
  tmp="$ROOT/quality/integration/latest.json.tmp.$$"
  printf '{"project":"%s","report":"quality/integration/%s/integration-test.jsonl"}\n' "$PROJECT" "$PROJECT" >"$tmp"
  mv "$tmp" "$ROOT/quality/integration/latest.json"
fi

exit "$primary_status"
