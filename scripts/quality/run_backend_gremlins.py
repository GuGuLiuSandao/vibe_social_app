#!/usr/bin/env python3
import json
import re
import subprocess
import sys
import tempfile
from pathlib import Path


def package_path(package: str) -> str:
    prefix = "social_app/"
    if not package.startswith(prefix):
        raise ValueError(f"unexpected backend package: {package}")
    return package.removeprefix(prefix)


def mutation_line(mutation: dict) -> int | None:
    value = mutation.get("line")
    return value if type(value) is int and value > 0 else None


def in_ranges(line: int | None, ranges: list[list[int]] | None) -> bool:
    if ranges is None:
        return True
    return line is not None and any(start <= line <= end for start, end in ranges)


def normalize_status(status: object) -> object:
    return status.replace(" ", "_") if isinstance(status, str) else status


def run_package(binary: Path, backend: Path, package: str, changed: list[str], allowed: list[str]) -> list[dict]:
    relative_package = package_path(package)
    unchanged = [
        path for path in allowed
        if str(Path(path).parent).removeprefix("backend/") == relative_package and path not in changed
    ]
    with tempfile.NamedTemporaryFile(suffix=".json", prefix="gremlins-", delete=False) as handle:
        raw_report = Path(handle.name)
    raw_report.unlink(missing_ok=True)
    command = [
        str(binary), "unleash", f"./{relative_package}",
        "--output", str(raw_report),
        "--threshold-efficacy", "0", "--threshold-mcover", "0",
        "--workers", "1", "--timeout-coefficient", "20", "--silent",
    ]
    for path in unchanged:
        command.extend(["--exclude-files", re.escape(Path(path).name) + "$"])
    try:
        result = subprocess.run(command, cwd=backend)
        if result.returncode:
            raise RuntimeError(f"Gremlins failed for {package} with exit code {result.returncode}")
        if not raw_report.is_file() or not raw_report.stat().st_size:
            raise RuntimeError(f"Gremlins did not produce a report for {package}")
        payload = json.loads(raw_report.read_text())
    finally:
        raw_report.unlink(missing_ok=True)
    files = payload.get("files")
    if not isinstance(files, list):
        raise ValueError(f"Gremlins report for {package} has no files list")
    return files


def main() -> int:
    if len(sys.argv) != 4:
        print("usage: run_backend_gremlins.py BINARY PLAN REPORT", file=sys.stderr)
        return 2
    binary, plan_path, report_path = map(Path, sys.argv[1:])
    backend = Path.cwd()
    try:
        plan = json.loads(plan_path.read_text())
        targets = plan["backend_mutation_targets"]
        changed = list(targets)
        allowed = plan["backend_allowed_files"]
        output_files = []
        for package in plan["backend_packages"]:
            raw_files = run_package(binary, backend, package, changed, allowed)
            package_changed = [
                path for path in changed
                if str(Path(path).parent).removeprefix("backend/") == package_path(package)
            ]
            by_name = {Path(path).name: path for path in package_changed}
            if len(by_name) != len(package_changed):
                raise ValueError(f"ambiguous changed filenames in {package}")
            for entry in raw_files:
                path = by_name.get(entry.get("file_name"))
                if path is None:
                    continue
                mutations = []
                for mutation in entry.get("mutations", []):
                    if in_ranges(mutation_line(mutation), targets[path]):
                        normalized = dict(mutation)
                        normalized["status"] = normalize_status(normalized.get("status"))
                        mutations.append(normalized)
                if mutations:
                    output_files.append({"file_name": entry["file_name"], "mutations": mutations})
        mutations = [mutation for entry in output_files for mutation in entry["mutations"]]
        counts = {
            status: sum(mutation.get("status") == status for mutation in mutations)
            for status in ("KILLED", "LIVED", "NOT_VIABLE", "NOT_COVERED")
        }
        report = {
            "go_module": "social_app",
            "files": output_files,
            "mutants_total": len(mutations),
            "mutants_killed": counts["KILLED"],
            "mutants_lived": counts["LIVED"],
            "mutants_not_viable": counts["NOT_VIABLE"],
            "mutants_not_covered": counts["NOT_COVERED"],
        }
        report_path.write_text(json.dumps(report, sort_keys=True))
    except (OSError, ValueError, KeyError, json.JSONDecodeError, RuntimeError) as error:
        report_path.unlink(missing_ok=True)
        print(error, file=sys.stderr)
        return 3
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
