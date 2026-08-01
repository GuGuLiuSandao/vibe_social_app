#!/usr/bin/env python3
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
REPORT = ROOT / "quality" / "backend-test.jsonl"
REQUIRED_PACKAGES = {"social_app/internal/auth", "social_app/internal/websocket"}


def validate_report(text: str) -> tuple[bool, str]:
    packages = set()
    tests = set()
    try:
        for raw in text.splitlines():
            event = json.loads(raw)
            if not isinstance(event, dict):
                return False, "Go JSONL event must be an object"
            if event.get("Action") == "run" and event.get("Test"):
                package = event.get("Package")
                if type(package) is not str or type(event["Test"]) is not str:
                    return False, "Go run event identity has invalid type"
                packages.add(package)
                tests.add((package, event["Test"]))
    except json.JSONDecodeError as error:
        return False, f"invalid Go JSON report: {error}"
    missing = REQUIRED_PACKAGES - packages
    if missing or len(tests) < 6:
        return False, f"backend report is incomplete: tests={len(tests)}, missing_packages={sorted(missing)}"
    return True, f"backend tests: {len(tests)} started across required packages"


def main() -> int:
    REPORT.parent.mkdir(parents=True, exist_ok=True)
    result = subprocess.run(
        ["go", "test", "-json", "./..."],
        cwd=ROOT / "backend",
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    REPORT.write_text(result.stdout)
    if result.stderr:
        (REPORT.parent / "backend-test.stderr.log").write_text(result.stderr)
    if result.returncode:
        sys.stderr.write(result.stdout)
        sys.stderr.write(result.stderr)
        return result.returncode

    valid, message = validate_report(result.stdout)
    if not valid:
        print(message, file=sys.stderr)
        return 3
    print(message)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
