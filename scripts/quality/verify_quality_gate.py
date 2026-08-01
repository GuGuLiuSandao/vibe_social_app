#!/usr/bin/env python3
"""Validate that the jobs required by a classified change all succeeded."""

from __future__ import annotations

import argparse


REQUIRED_RESULTS = {
    "docs": ("classify", "docs"),
    "engineering": ("classify", "engineering"),
    "develop": (
        "classify",
        "static",
        "backend",
        "frontend",
        "integration",
        "backend-mutation",
        "frontend-mutation",
    ),
}


def verify(classification: str, results: dict[str, str]) -> list[str]:
    required = REQUIRED_RESULTS.get(classification)
    if required is None:
        return [f"unsupported or invalid classification: {classification}"]
    return [f"{job} did not succeed (result: {results.get(job, 'missing')})" for job in required if results.get(job) != "success"]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--classification", required=True)
    parser.add_argument("--result", action="append", default=[])
    args = parser.parse_args()
    results = dict(item.split("=", 1) for item in args.result)
    failures = verify(args.classification, results)
    if failures:
        for failure in failures:
            print(f"quality gate failure: {failure}")
        return 1
    print(f"quality gate passed for {args.classification}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
