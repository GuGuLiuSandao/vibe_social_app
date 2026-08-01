#!/usr/bin/env python3
"""Classify a change from its actual paths and optional Develop Loop label."""

from __future__ import annotations

import argparse
import fnmatch
import json
import subprocess
from dataclasses import dataclass
from pathlib import Path


DEVELOP_LABEL = "develop-loop"

DOC_PATTERNS = (
    "*.md",
    "LICENSE",
    "docs/*.md",
    "docs/backend/**",
    "docs/plans/**",
)

ENGINEERING_PATTERNS = (
    ".codex/**",
    ".engineering-loop/**",
    ".github/**",
    ".dockerignore",
    ".gitignore",
    "AGENTS.md",
    "Jenkinsfile",
    "Makefile",
    "buf.gen.yaml",
    "docker-compose*.yml",
    "docs/changes/**",
    "docs/specs/**",
    "scripts/**",
)

PRODUCT_PATTERNS = (
    "backend/**",
    "frontend/**",
    "proto/**",
)


@dataclass(frozen=True)
class Classification:
    name: str
    reason: str
    files: tuple[str, ...]


def _matches(path: str, patterns: tuple[str, ...]) -> bool:
    return any(fnmatch.fnmatchcase(path, pattern) for pattern in patterns)


def classify_paths(paths: list[str], labels: set[str]) -> Classification:
    files = tuple(sorted(set(path.strip("/") for path in paths if path.strip("/"))))
    if not files:
        return Classification("invalid", "no changed files were detected", files)

    unknown = [path for path in files if not _matches(path, DOC_PATTERNS + ENGINEERING_PATTERNS + PRODUCT_PATTERNS)]
    if unknown:
        return Classification("invalid", f"unclassified paths: {', '.join(unknown)}", files)

    product = [path for path in files if _matches(path, PRODUCT_PATTERNS)]
    if DEVELOP_LABEL in labels:
        return Classification("develop", "develop-loop label requires the complete quality suite", files)
    if product:
        return Classification(
            "invalid",
            f"product paths require the develop-loop label: {', '.join(product)}",
            files,
        )

    engineering = [path for path in files if _matches(path, ENGINEERING_PATTERNS)]
    if engineering:
        return Classification("engineering", "engineering or process paths changed", files)
    return Classification("docs", "only documentation paths changed", files)


def _git_lines(root: Path, *args: str) -> list[str]:
    result = subprocess.run(
        ["git", *args],
        cwd=root,
        check=True,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    return [line for line in result.stdout.splitlines() if line]


def changed_paths(root: Path, base: str, head: str, include_worktree: bool) -> list[str]:
    paths = set(_git_lines(root, "diff", "--name-only", f"{base}...{head}"))
    if include_worktree:
        paths.update(_git_lines(root, "diff", "--name-only", head))
        paths.update(_git_lines(root, "ls-files", "--others", "--exclude-standard"))
    return sorted(paths)


def parse_labels(raw: str) -> set[str]:
    raw = raw.strip()
    if not raw:
        return set()
    if raw.startswith("["):
        value = json.loads(raw)
        return {str(item).strip() for item in value if str(item).strip()}
    return {item.strip() for item in raw.split(",") if item.strip()}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base", required=True)
    parser.add_argument("--head", default="HEAD")
    parser.add_argument("--labels", default="")
    parser.add_argument("--include-worktree", action="store_true")
    parser.add_argument("--github-output")
    args = parser.parse_args()

    root = Path(__file__).resolve().parents[2]
    result = classify_paths(changed_paths(root, args.base, args.head, args.include_worktree), parse_labels(args.labels))
    print(f"change classification: {result.name}")
    print(f"reason: {result.reason}")
    for path in result.files:
        print(f"  {path}")

    if args.github_output:
        with Path(args.github_output).open("a", encoding="utf-8") as output:
            output.write(f"classification={result.name}\n")
            output.write(f"reason={result.reason}\n")
    return 0 if result.name != "invalid" else 2


if __name__ == "__main__":
    raise SystemExit(main())
