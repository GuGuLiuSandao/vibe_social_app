#!/usr/bin/env python3
import json
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))

from scripts.quality.mutation_config import frontend_config

FRONTEND = ROOT / "frontend"
PLAN = ROOT / "quality" / "mutation-plan.json"
REPORT = ROOT / "quality" / "frontend-mutation.json"


def main() -> int:
    baseline = subprocess.run([sys.executable, str(ROOT / "scripts/quality/run_frontend_tests.py")], cwd=ROOT)
    if baseline.returncode:
        return baseline.returncode
    target = subprocess.run(
        [sys.executable, str(ROOT / "scripts/quality/mutation_targets.py")],
        cwd=ROOT,
        text=True,
        stdout=subprocess.PIPE,
    )
    if target.returncode:
        return target.returncode
    PLAN.parent.mkdir(parents=True, exist_ok=True)
    PLAN.write_text(target.stdout)
    plan = json.loads(target.stdout)
    targets = plan["frontend_mutation_targets"]
    if not targets:
        print("frontend mutation plan selected no targets", file=sys.stderr)
        return 2
    config = frontend_config(targets, REPORT)
    with tempfile.NamedTemporaryFile("w", suffix=".mjs", prefix="stryker-dlq-", dir=FRONTEND, delete=False) as handle:
        handle.write(config)
        config_path = Path(handle.name)
    try:
        result = subprocess.run(["npx", "stryker", "run", str(config_path)], cwd=FRONTEND)
        if result.returncode:
            return result.returncode
    finally:
        config_path.unlink(missing_ok=True)
    verified = subprocess.run(
        [sys.executable, str(ROOT / "scripts/quality/verify_mutation_report.py"), "frontend", str(REPORT), str(PLAN)],
        cwd=ROOT,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if verified.stdout:
        print(verified.stdout, end="")
    if verified.stderr:
        print(verified.stderr, end="", file=sys.stderr)
    if verified.returncode:
        return verified.returncode
    (ROOT / "quality/frontend-mutation-summary.txt").write_text(verified.stdout)
    return subprocess.run(
        [sys.executable, str(ROOT / "scripts/quality/verify_weak_mutation.py"), "frontend"],
        cwd=ROOT,
    ).returncode


if __name__ == "__main__":
    raise SystemExit(main())
