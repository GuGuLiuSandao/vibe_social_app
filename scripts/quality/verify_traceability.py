#!/usr/bin/env python3
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
MANIFEST = ROOT / "docs/specs/traceability.json"
CASES = ROOT / "docs/changes/social-app-develop-loop-quality/testcases.md"
REQUIREMENT = ROOT / "docs/changes/social-app-develop-loop-quality/requirement.md"
EVIDENCE_BY_COMMAND = {
    "make quality-static": {"quality/static-contract.json"},
    "make test-backend": {"quality/backend-test.jsonl"},
    "make test-frontend": {"quality/frontend-test.json"},
    "make test-integration": {"quality/integration/latest.json"},
    "make mutation-backend": {"quality/backend-mutation.json"},
    "make mutation-frontend": {"quality/frontend-mutation.json"},
    "make mutation-backend && make mutation-frontend": {"quality/mutation-proof.json"},
}


def inside(root: Path, relative: object) -> Path:
    if type(relative) is not str or not relative:
        raise ValueError("path must be a non-empty string")
    candidate = Path(relative)
    if candidate.is_absolute() or ".." in candidate.parts:
        raise ValueError(f"path escapes repository: {relative}")
    resolved = (root / candidate).resolve()
    if root.resolve() not in resolved.parents and resolved != root.resolve():
        raise ValueError(f"path escapes repository: {relative}")
    return resolved


def exact_test_identity(path: Path, name: str) -> bool:
    text = path.read_text()
    escaped = re.escape(name)
    suffix = path.suffix
    if suffix == ".go":
        return bool(re.search(rf"^func\s+{escaped}\s*\(", text, re.MULTILINE))
    if suffix == ".py":
        return bool(re.search(rf"^\s*def\s+{escaped}\s*\(", text, re.MULTILINE))
    if suffix in {".js", ".jsx", ".mjs", ".ts", ".tsx"}:
        return bool(re.search(rf"[\"']{escaped}[\"']", text))
    return bool(re.search(rf"(?<![A-Za-z0-9_]){escaped}(?![A-Za-z0-9_])", text))


def exact_case_identity(path: Path, name: str, case: str) -> bool:
    number = case.rsplit("-", 1)[-1]
    normalized_name = re.sub(r"[-_]", " ", name)
    if re.search(rf"DLQ\s+TC(?:\s+\d{{3}})*\s+{re.escape(number)}\b", normalized_name):
        return True
    return bool(re.search(rf"(?<![A-Z0-9-]){re.escape(case)}(?![A-Z0-9-])", path.read_text()))


def validate(root: Path, manifest_path: Path, cases_path: Path, requirement_path: Path) -> list[str]:
    errors = []
    required_specs = {"AUTH-001", "WS-001", "CLIENT-001", "CLIENT-002", "AUTH-HTTP-001", "WS-HTTP-001"}
    try:
        payload = json.loads(manifest_path.read_text())
    except (OSError, json.JSONDecodeError) as error:
        return [f"manifest read/schema error: {error}"]
    if not isinstance(payload, dict) or payload.get("schema_version") != 1 or not isinstance(payload.get("entries"), list):
        return ["manifest root must be schema_version 1 with an entries list"]
    try:
        case_text = cases_path.read_text()
        requirement_text = requirement_path.read_text()
    except OSError as error:
        return [f"contract read error: {error}"]
    required_cases = set(re.findall(r"^### (DLQ-TC-0(?:0[1-9]|[12][0-9]|3[0-6])):", case_text, re.MULTILINE))
    seen_cases = set()
    seen_rows = set()
    covered_specs = set()
    required_fields = {"acceptance_id", "case_id", "test_file", "test_name", "command", "evidence"}
    for index, entry in enumerate(payload["entries"]):
        if not isinstance(entry, dict) or not required_fields <= set(entry):
            errors.append(f"entry {index} is incomplete")
            continue
        row = tuple(entry.get(field) for field in sorted(required_fields))
        if row in seen_rows:
            errors.append(f"duplicate entry at index {index}")
        seen_rows.add(row)
        case = entry["case_id"]
        acceptance = entry["acceptance_id"]
        if case in seen_cases:
            errors.append(f"duplicate Case ID: {case}")
        seen_cases.add(case)
        if case not in required_cases:
            errors.append(f"unknown or non-P0 Case ID: {case}")
        if type(acceptance) is not str or not re.fullmatch(r"DLQ-00[1-9]", acceptance) or acceptance not in requirement_text:
            errors.append(f"unknown acceptance ID: {acceptance}")
        try:
            test_file = inside(root, entry["test_file"])
            evidence = inside(root, entry["evidence"])
        except ValueError as error:
            errors.append(str(error))
            continue
        if not test_file.is_file():
            errors.append(f"missing test file: {entry['test_file']}")
        elif type(entry["test_name"]) is not str or not exact_test_identity(test_file, entry["test_name"]):
            errors.append(f"mismatched full test name: {entry['test_name']} in {entry['test_file']}")
        elif not exact_case_identity(test_file, entry["test_name"], case):
            errors.append(f"test file lacks exact Case ID: {case} in {entry['test_file']}")
        command = entry["command"]
        if type(command) is not str or not command.strip():
            errors.append(f"missing command for {case}")
        if type(entry["evidence"]) is not str or not entry["evidence"].startswith("quality/"):
            errors.append(f"evidence must be under quality/: {entry['evidence']}")
        elif entry["evidence"] not in EVIDENCE_BY_COMMAND.get(command, set()):
            errors.append(f"unresolvable evidence binding: {command} -> {entry['evidence']}")
        specification = entry.get("specification_id")
        if specification is not None:
            try:
                spec_file = inside(root, entry.get("specification_file"))
            except ValueError as error:
                errors.append(str(error))
            else:
                if type(specification) is not str or specification not in required_specs:
                    errors.append(f"unknown specification: {specification}")
                elif not spec_file.is_file():
                    errors.append(f"missing specification document: {entry.get('specification_file')}")
                elif not re.search(rf"(?<![A-Z0-9-]){re.escape(specification)}(?![A-Z0-9-])", spec_file.read_text()):
                    errors.append(f"specification document lacks exact ID: {specification}")
                else:
                    covered_specs.add(specification)
    if missing := required_cases - seen_cases:
        errors.append(f"uncovered P0 Cases: {sorted(missing)}")
    if extra := seen_cases - required_cases:
        errors.append(f"unexpected Cases: {sorted(extra)}")
    if missing := required_specs - covered_specs:
        errors.append(f"uncovered specifications: {sorted(missing)}")
    return errors


def main() -> int:
    errors = validate(ROOT, MANIFEST, CASES, REQUIREMENT)
    if errors:
        print("\n".join(errors), file=sys.stderr)
        return 3
    entries = json.loads(MANIFEST.read_text())["entries"]
    print(f"traceability: {len(entries)} P0 Case links verified")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
