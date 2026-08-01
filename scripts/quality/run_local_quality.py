#!/usr/bin/env python3
"""Dispatch local quality gates using the repository change policy."""

from __future__ import annotations

import os
import subprocess
from pathlib import Path

from change_classifier import changed_paths, classify_paths, parse_labels


def resolve_base(root: Path) -> str:
    configured = os.environ.get("BASE_SHA", "").strip()
    candidates = [configured] if configured else ["origin/master", "master", "HEAD^"]
    for candidate in candidates:
        result = subprocess.run(
            ["git", "rev-parse", "--verify", candidate],
            cwd=root,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        if result.returncode == 0:
            return candidate
    raise SystemExit("unable to determine comparison base; set BASE_SHA=<trusted-base>")


def main() -> int:
    root = Path(__file__).resolve().parents[2]
    base = resolve_base(root)
    labels = parse_labels(os.environ.get("CHANGE_LABELS", ""))
    result = classify_paths(changed_paths(root, base, "HEAD", include_worktree=True), labels)

    print(f"local change classification: {result.name}")
    print(f"comparison base: {base}")
    print(f"reason: {result.reason}")
    if result.name == "invalid":
        print("For a requirement change, rerun with CHANGE_LABELS=develop-loop after completing its Develop Loop artifacts.")
        return 2

    target = {
        "docs": "quality-docs",
        "engineering": "quality-engineering",
        "develop": "quality-develop",
    }[result.name]
    return subprocess.run(["make", target], cwd=root).returncode


if __name__ == "__main__":
    raise SystemExit(main())
