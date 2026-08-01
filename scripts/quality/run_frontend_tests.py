#!/usr/bin/env python3
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
REPORT = ROOT / "quality" / "frontend-test.json"
REQUIRED_FILES = {"uid.test.js", "ws.test.js"}


def validate_report(payload: object) -> tuple[bool, str]:
    if not isinstance(payload, dict):
        return False, "Vitest report root must be an object"
    numeric_fields = (
        "numTotalTestSuites",
        "numPassedTestSuites",
        "numFailedTestSuites",
        "numTotalTests",
        "numPassedTests",
        "numFailedTests",
    )
    for field in numeric_fields:
        if type(payload.get(field)) is not int or payload[field] < 0:
            return False, f"Vitest field {field} is missing or invalid"
    suites = payload.get("testResults")
    if not isinstance(suites, list) or any(not isinstance(suite, dict) for suite in suites):
        return False, "Vitest testResults must be a list of objects"
    seen = {Path(suite.get("name", "")).name for suite in suites if type(suite.get("name")) is str}
    missing = REQUIRED_FILES - seen
    if payload["numTotalTestSuites"] < 2 or len(suites) < 2 or payload["numTotalTests"] < 6 or missing:
        return False, f"frontend report is incomplete: suites={len(suites)}, tests={payload['numTotalTests']}, missing={sorted(missing)}"
    if payload["numFailedTests"] or payload["numFailedTestSuites"]:
        return False, "frontend report contains failures"
    if payload["numPassedTests"] != payload["numTotalTests"] or payload["numPassedTestSuites"] != payload["numTotalTestSuites"]:
        return False, "frontend passed counters do not match totals"
    return True, f"frontend tests: {payload['numTotalTests']} passed in {len(suites)} files"


def main() -> int:
    REPORT.parent.mkdir(parents=True, exist_ok=True)
    result = subprocess.run(
        [
            "npm",
            "test",
            "--",
            "--passWithNoTests=false",
            "--reporter=json",
            f"--outputFile={REPORT}",
        ],
        cwd=ROOT / "frontend",
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    if result.stdout:
        print(result.stdout, end="")
    if result.returncode:
        return result.returncode
    if not REPORT.is_file() or not REPORT.stat().st_size:
        print("frontend JSON report is missing or empty", file=sys.stderr)
        return 2
    try:
        payload = json.loads(REPORT.read_text())
    except json.JSONDecodeError as error:
        print(f"invalid Vitest JSON report: {error}", file=sys.stderr)
        return 3
    valid, message = validate_report(payload)
    if not valid:
        print(message, file=sys.stderr)
        return 4
    print(message)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
