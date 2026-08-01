#!/usr/bin/env python3
import json
import sys
from pathlib import Path


def validate_plan(plan: dict) -> tuple[bool, str]:
    required = {
        "schema_version": int,
        "reason": str,
        "backend_packages": list,
        "backend_allowed_files": list,
        "frontend_files": list,
        "backend_smoke": bool,
        "frontend_smoke": bool,
    }
    if not isinstance(plan, dict):
        return False, "mutation plan must be an object"
    for field, kind in required.items():
        if type(plan.get(field)) is not kind:
            return False, f"mutation plan field {field} has invalid type"
    if plan["schema_version"] != 1:
        return False, "unsupported mutation plan schema"
    for field in ("backend_packages", "backend_allowed_files", "frontend_files"):
        values = plan[field]
        if not values or any(type(value) is not str or not value for value in values) or len(values) != len(set(values)):
            return False, f"mutation plan field {field} is empty, duplicated, or invalid"
    return True, "ok"


def resolve_backend_file(name: str, allowed: set[str]) -> str | None:
    if type(name) is not str or not name or Path(name).is_absolute() or ".." in Path(name).parts:
        return None
    normalized = name.removeprefix("backend/").removeprefix("./")
    matches = [path for path in allowed if path.removeprefix("backend/") == normalized or path.endswith("/" + normalized)]
    return matches[0] if len(matches) == 1 else None


def backend(payload: dict, plan: dict | None = None) -> tuple[bool, str]:
    valid, message = validate_plan(plan)
    if not valid:
        return False, message
    if not isinstance(payload, dict) or not isinstance(payload.get("files"), list):
        return False, "Gremlins files list is missing"
    allowed = set(plan["backend_allowed_files"])
    actual_files = set()
    mutations = []
    for file in payload["files"]:
        if not isinstance(file, dict) or not isinstance(file.get("mutations"), list):
            return False, "Gremlins file entry is malformed"
        resolved = resolve_backend_file(file.get("file_name"), allowed)
        if resolved is None:
            return False, f"Gremlins file is outside planned scope: {file.get('file_name')!r}"
        actual_files.add(resolved)
        mutations.extend(file["mutations"])
    if not mutations or type(payload.get("mutants_total")) is not int or payload["mutants_total"] != len(mutations):
        return False, "Gremlins report has zero mutants or inconsistent total"
    for counter in ("mutants_killed", "mutants_lived", "mutants_not_viable", "mutants_not_covered"):
        if type(payload.get(counter)) is not int or payload[counter] < 0:
            return False, f"Gremlins counter {counter} is invalid"
    statuses = {mutation.get("status") if isinstance(mutation, dict) else None for mutation in mutations}
    unsafe = statuses - {"KILLED", "NOT_VIABLE"}
    if unsafe:
        return False, f"Gremlins contains unsafe statuses: {sorted(map(repr, unsafe))}"
    if payload["mutants_lived"] != 0 or payload["mutants_not_covered"] != 0:
        return False, "Gremlins contains lived or uncovered mutants"
    if payload["mutants_killed"] + payload["mutants_not_viable"] != len(mutations):
        return False, "Gremlins counters do not match file mutations"
    represented = {"social_app/" + str(Path(path).parent.relative_to("backend")) for path in actual_files}
    missing_packages = set(plan["backend_packages"]) - represented
    if missing_packages:
        return False, f"Gremlins omitted planned packages: {sorted(missing_packages)}"
    return True, f"Gremlins: {len(mutations)} protected mutants in {sorted(actual_files)}"


def frontend(payload: dict, plan: dict | None = None) -> tuple[bool, str]:
    valid, message = validate_plan(plan)
    if not valid:
        return False, message
    files = payload.get("files") if isinstance(payload, dict) else None
    if not isinstance(files, dict):
        return False, "Stryker files object is missing"
    expected = {path.removeprefix("frontend/") for path in plan["frontend_files"]}
    actual = set(files)
    if actual != expected:
        return False, f"Stryker target mismatch: expected={sorted(expected)}, actual={sorted(actual)}"
    mutants = []
    for file in files.values():
        if not isinstance(file, dict) or not isinstance(file.get("mutants"), list):
            return False, "Stryker file entry is malformed"
        mutants.extend(file["mutants"])
    if not mutants:
        return False, "Stryker report has zero mutants"
    statuses = {mutant.get("status") if isinstance(mutant, dict) else None for mutant in mutants}
    if statuses != {"Killed"}:
        return False, f"Stryker contains unsafe statuses: {sorted(map(repr, statuses - {'Killed'}))}"
    return True, f"Stryker: {len(mutants)} protected mutants in {sorted(actual)}"


def load(path: Path, label: str) -> dict:
    if not path.is_file() or not path.stat().st_size:
        raise ValueError(f"{label} is missing or empty")
    payload = json.loads(path.read_text())
    if not isinstance(payload, dict):
        raise ValueError(f"{label} root must be an object")
    return payload


def main() -> int:
    if len(sys.argv) != 4 or sys.argv[1] not in {"backend", "frontend"}:
        print("usage: verify_mutation_report.py backend|frontend REPORT PLAN", file=sys.stderr)
        return 2
    try:
        payload = load(Path(sys.argv[2]), "mutation report")
        plan = load(Path(sys.argv[3]), "mutation plan")
    except (OSError, ValueError, json.JSONDecodeError) as error:
        print(f"invalid mutation evidence: {error}", file=sys.stderr)
        return 3
    ok, message = backend(payload, plan) if sys.argv[1] == "backend" else frontend(payload, plan)
    print(message, file=sys.stdout if ok else sys.stderr)
    return 0 if ok else 4


if __name__ == "__main__":
    raise SystemExit(main())
