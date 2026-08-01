#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
BIN="$ROOT/quality/bin/gremlins"
PLAN="$ROOT/quality/mutation-plan.json"
REPORT="$ROOT/quality/backend-mutation.json"
mkdir -p "$ROOT/quality/bin"

python3 "$ROOT/scripts/quality/run_backend_tests.py"
python3 "$ROOT/scripts/quality/mutation_targets.py" >"$PLAN"

if [ ! -x "$BIN" ]; then
  GOBIN="$ROOT/quality/bin" go install github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0
fi
if ! go version -m "$BIN" | grep -q $'mod\tgithub.com/go-gremlins/gremlins\tv0.6.0'; then
  echo "Gremlins binary is not module version v0.6.0" >&2
  exit 2
fi

base=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1])).get("base_sha") or "")' "$PLAN")
smoke=$(python3 -c 'import json,sys; print(str(json.load(open(sys.argv[1]))["backend_smoke"]).lower())' "$PLAN")
cd "$ROOT/backend"
if [ "$smoke" = "true" ]; then
  "$BIN" unleash ./internal/auth --exclude-files 'handler\.go' --output "$REPORT" --threshold-efficacy 100 --threshold-mcover 100 --workers 1 --timeout-coefficient 20 --silent
else
  "$BIN" unleash ./... --diff "$base" --output "$REPORT" --threshold-efficacy 100 --threshold-mcover 100 --workers 1 --timeout-coefficient 20 --silent
fi
python3 "$ROOT/scripts/quality/verify_mutation_report.py" backend "$REPORT" "$PLAN" | tee "$ROOT/quality/backend-mutation-summary.txt"
python3 "$ROOT/scripts/quality/verify_weak_mutation.py" backend
