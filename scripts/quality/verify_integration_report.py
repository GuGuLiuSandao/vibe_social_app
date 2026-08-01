#!/usr/bin/env python3
import json
import sys
from pathlib import Path


def main() -> int:
    if len(sys.argv) != 2:
        print("usage: verify_integration_report.py REPORT", file=sys.stderr)
        return 2
    path = Path(sys.argv[1])
    if not path.is_file() or not path.stat().st_size:
        print("integration report is missing or empty", file=sys.stderr)
        return 3
    passed = set()
    try:
        for line in path.read_text().splitlines():
            event = json.loads(line)
            if event.get("Action") == "pass" and event.get("Test"):
                passed.add(event["Test"])
    except json.JSONDecodeError as error:
        print(f"invalid integration JSONL: {error}", file=sys.stderr)
        return 4
    expected = {
        "TestDLQ_TC_011_016_AUTH_HTTP_001_RegisterLogin",
        "TestDLQ_TC_017_WS_HTTP_001_AuthenticatedPingPong",
    }
    missing = expected - passed
    if missing or len(passed) < 2:
        print(f"required integration tests did not pass: {sorted(missing)}", file=sys.stderr)
        return 5
    print(f"integration report: {len(passed)} test events passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
