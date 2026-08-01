#!/usr/bin/env python3
import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT))

from scripts.quality.mutation_config import frontend_config

FIXTURES = ROOT / "scripts/quality/tests/fixtures/weak-suites"


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def tree_hashes(root: Path) -> dict[str, str]:
    return {
        path.relative_to(root).as_posix(): digest(path)
        for path in root.rglob("*")
        if path.is_file() and ".stryker-tmp" not in path.parts
    }


def mapping_digest(mapping: dict[str, str]) -> str:
    return hashlib.sha256(json.dumps(mapping, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def validate_tree_change(before: dict[str, str], after: dict[str, str], expected_path: str, expected_sha: str) -> None:
    changed = {path for path in before.keys() | after.keys() if before.get(path) != after.get(path)}
    if changed != {expected_path}:
        raise RuntimeError(f"weak patch changed unexpected paths: {sorted(changed)}")
    if after.get(expected_path) != expected_sha:
        raise RuntimeError(f"weak patch output hash mismatch: {expected_path}")


def protective_tests(kind: str, text: str, target_calls: tuple[str, ...]) -> set[str]:
    pattern = r"^func\s+(Test\w+)\s*\(" if kind == "backend" else r'^describe\(["\']([^"\']+)["\']'
    matches = list(re.finditer(pattern, text, re.MULTILINE))
    found = set()
    for index, match in enumerate(matches):
        end = matches[index + 1].start() if index + 1 < len(matches) else len(text)
        section = text[match.start():end]
        if any(re.search(rf"\b{re.escape(call)}\s*\(", section) for call in target_calls):
            found.add(match.group(1))
    return found


def suite_inventory(root: Path, kind: str) -> set[tuple[str, str]]:
    calls = ("GenerateToken", "ParseToken") if kind == "backend" else ("parseUid", "isWhitelistUid")
    if kind == "backend":
        files = sorted((root / "backend/internal/auth").glob("*_test.go"))
    else:
        files = sorted(path for path in (root / "frontend/src").rglob("*") if path.is_file() and ".test." in path.name)
    found = set()
    for path in files:
        for name in protective_tests(kind, path.read_text(), calls):
            found.add((path.relative_to(root).as_posix(), name))
    return found


def frontend_target_calling_files(root: Path) -> set[str]:
    calls = ("parseUid", "isWhitelistUid")
    files = sorted(path for path in (root / "frontend/src").rglob("*") if path.is_file() and ".test." in path.name)
    return {
        path.relative_to(root).as_posix()
        for path in files
        if any(re.search(rf"\b{call}\s*\(", path.read_text()) for call in calls)
    }


def stryker_protective_inventory(report: Path) -> set[tuple[str, str]]:
    payload = json.loads(report.read_text())
    identities = {}
    for file, data in payload.get("testFiles", {}).items():
        for test in data.get("tests", []):
            identities[str(test["id"])] = (f"frontend/{file}", test["name"])
    covered = {
        str(test_id)
        for data in payload.get("files", {}).values()
        for mutant in data.get("mutants", [])
        for test_id in mutant.get("coveredBy", [])
    }
    if missing := covered - identities.keys():
        raise RuntimeError(f"Stryker report references unknown test IDs: {sorted(missing)}")
    return {identities[test_id] for test_id in covered}


def verify_protective_inventory(payload: dict, kind: str, root: Path) -> set[tuple[str, str]]:
    expected = {(item["file"], item["name"]) for item in payload["protective_tests"]}
    if kind == "frontend":
        found_files = frontend_target_calling_files(root)
        expected_files = {file for file, _ in expected}
        if found_files != expected_files:
            raise RuntimeError(f"protective test inventory mismatch: found_files={sorted(found_files)}, expected_files={sorted(expected_files)}")
        return expected
    found = suite_inventory(root, kind)
    if found != expected:
        raise RuntimeError(f"protective test inventory mismatch: found={sorted(found)}, expected={sorted(expected)}")
    return found


def verify_stryker_inventory(payload: dict, report: Path) -> set[tuple[str, str]]:
    found = stryker_protective_inventory(report)
    expected = {(item["file"], item["name"]) for item in payload["protective_tests"]}
    if found != expected:
        raise RuntimeError(f"Stryker protective identity mismatch: found={sorted(found)}, expected={sorted(expected)}")
    return found


def manifest(name: str) -> dict:
    payload = json.loads((FIXTURES / f"{name}.json").read_text())
    strong = ROOT / payload["strong_test"]
    weak = ROOT / payload["weak_replacement"]
    patch = ROOT / payload["patch"]
    if digest(strong) != payload["strong_sha256"]:
        raise RuntimeError(f"strong test drifted: {strong}")
    if digest(weak) != payload["weak_sha256"]:
        raise RuntimeError(f"weak replacement drifted: {weak}")
    if not patch.is_file() or not patch.stat().st_size:
        raise RuntimeError(f"weak patch is missing: {patch}")
    verify_protective_inventory(payload, "backend" if name.startswith("backend-") else "frontend", ROOT)
    if payload["weak_test"] not in weak.read_text():
        raise RuntimeError("weak test name is missing from replacement")
    return payload


def backend_plan(path: Path, payload: dict) -> None:
    path.write_text(json.dumps({"schema_version": 1, "base_sha": None, "reason": "weak-proof", "backend_changed_files": [], "backend_mutation_targets": {payload["target"]: None}, "backend_packages": ["social_app/internal/auth"], "backend_allowed_files": [payload["target"]], "frontend_files": ["frontend/src/lib/uid.js"], "frontend_mutation_targets": ["src/lib/uid.js"], "backend_smoke": True, "frontend_smoke": True}))


def gremlins_command(binary: Path, report: Path) -> list[str]:
    return [str(binary), "unleash", "./internal/auth", "--exclude-files", r"handler\.go", "--output", str(report), "--threshold-efficacy", "100", "--threshold-mcover", "100", "--workers", "1", "--timeout-coefficient", "20", "--silent"]


def frontend_plan(path: Path, payload: dict) -> None:
    path.write_text(json.dumps({"schema_version": 1, "base_sha": None, "reason": "weak-proof", "backend_changed_files": [], "backend_mutation_targets": {"backend/internal/auth/jwt.go": None}, "backend_packages": ["social_app/internal/auth"], "backend_allowed_files": ["backend/internal/auth/jwt.go"], "frontend_files": [payload["target"]], "frontend_mutation_targets": [payload["target"].removeprefix("frontend/")], "backend_smoke": True, "frontend_smoke": True}))


def copy_frontend(destination: Path) -> None:
    shutil.copytree(
        ROOT / "frontend",
        destination,
        ignore=shutil.ignore_patterns("node_modules", "dist", ".stryker-tmp"),
    )
    os.symlink(ROOT / "frontend/node_modules", destination / "node_modules", target_is_directory=True)


def write_frontend_proof_config(sandbox: Path) -> str:
    name = "vitest.proof.config.mjs"
    (sandbox / name).write_text(
        'export default { test: { environment: "jsdom", setupFiles: "./src/test/setup.js", include: ["src/lib/uid.test.js"] } };\n'
    )
    return name


def retain_report(kind: str, phase: str, source: Path) -> dict[str, str]:
    relative = Path("mutation-proof-details") / f"{kind}-{phase}-report.json"
    destination = ROOT / "quality" / relative
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(source, destination)
    return {"path": relative.as_posix(), "sha256": digest(destination)}


def command_digest(command: list[str], report: Path) -> str:
    canonical = ["<REPORT>" if item == str(report) else item for item in command]
    return hashlib.sha256(json.dumps(canonical, separators=(",", ":")).encode()).hexdigest()


def backend_weak() -> dict:
    payload = manifest("backend-jwt-v1")
    inventory = [{"file": file, "name": name} for file, name in sorted(verify_protective_inventory(payload, "backend", ROOT))]
    binary = ROOT / "quality/bin/gremlins"
    if not binary.is_file():
        raise RuntimeError("strong backend mutation must install Gremlins before weak proof")
    with tempfile.TemporaryDirectory(prefix="social-dlq-backend-strong-") as temp:
        sandbox = Path(temp) / "backend"
        shutil.copytree(ROOT / "backend", sandbox)
        strong_tree = tree_hashes(sandbox)
        baseline = subprocess.run(["go", "test", "./internal/auth"], cwd=sandbox)
        if baseline.returncode:
            raise RuntimeError("unpatched backend ordinary baseline failed in disposable copy")
        strong_report = Path(temp) / "backend-strong-mutation.json"
        plan = Path(temp) / "backend-plan.json"
        backend_plan(plan, payload)
        strong_command = gremlins_command(binary, strong_report)
        strong = subprocess.run(strong_command, cwd=sandbox)
        strong_verifier = subprocess.run(
            [sys.executable, str(ROOT / "scripts/quality/verify_mutation_report.py"), "backend", str(strong_report), str(plan)],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        if strong.returncode or strong_verifier.returncode:
            raise RuntimeError("unpatched backend strong control did not pass in disposable copy")
        strong_statuses = sorted({mutation["status"] for file in json.loads(strong_report.read_text())["files"] for mutation in file["mutations"]})
        strong_retained = retain_report("backend", "strong", strong_report)
        strong_copy_id = Path(temp).name
    with tempfile.TemporaryDirectory(prefix="social-dlq-backend-weak-") as temp:
        sandbox = Path(temp) / "backend"
        shutil.copytree(ROOT / "backend", sandbox)
        before = tree_hashes(sandbox)
        subprocess.run(["patch", "-p1", "-i", str(ROOT / payload["patch"])], cwd=temp, check=True)
        after = tree_hashes(sandbox)
        changed_paths = sorted(path for path in before.keys() | after.keys() if before.get(path) != after.get(path))
        validate_tree_change(before, after, payload["strong_test"].removeprefix("backend/"), payload["weak_sha256"])
        if digest(sandbox / payload["target"].removeprefix("backend/")) != digest(ROOT / payload["target"]):
            raise RuntimeError("backend weak patch changed production target")
        baseline = subprocess.run(["go", "test", "./internal/auth"], cwd=sandbox)
        if baseline.returncode:
            raise RuntimeError("weak backend ordinary baseline failed")
        report = Path(temp) / "backend-weak-mutation.json"
        plan = Path(temp) / "backend-plan.json"
        backend_plan(plan, payload)
        weak_command = gremlins_command(binary, report)
        tool = subprocess.run(weak_command, cwd=sandbox)
        evidence = json.loads(report.read_text())
        statuses = {mutation["status"] for file in evidence["files"] for mutation in file["mutations"]}
        verifier = subprocess.run(
            [sys.executable, str(ROOT / "scripts/quality/verify_mutation_report.py"), "backend", str(report), str(plan)],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        if tool.returncode not in {0, 1} or verifier.returncode == 0 or not statuses.intersection(payload["allowed_weak_statuses"]):
            raise RuntimeError(f"weak backend suite did not fail for assertion weakness: {sorted(statuses)}")
        weak_retained = retain_report("backend", "weak", report)
        weak_copy_id = Path(temp).name
    print("weak backend proof: ordinary tests passed and mutation gate rejected weak assertions")
    return {
        "status": "weak-baseline-green-mutation-rejected",
        "protective_inventory": inventory,
        "production_target_sha256": digest(ROOT / payload["target"]),
        "strong": {"copy_id": strong_copy_id, "ordinary_baseline": 0, "mutation_exit": strong.returncode, "verifier_exit": strong_verifier.returncode, "command_sha256": command_digest(strong_command, strong_report), "tree_sha256": mapping_digest(strong_tree), "statuses": strong_statuses, "report": strong_retained},
        "weak": {"copy_id": weak_copy_id, "ordinary_baseline": 0, "mutation_exit": tool.returncode, "verifier_exit": verifier.returncode, "command_sha256": command_digest(weak_command, report), "tree_before_sha256": mapping_digest(before), "tree_after_sha256": mapping_digest(after), "changed_paths": changed_paths, "statuses": sorted(statuses), "report": weak_retained},
    }


def frontend_weak() -> dict:
    payload = manifest("frontend-uid-v1")
    inventory = [{"file": file, "name": name} for file, name in sorted(verify_protective_inventory(payload, "frontend", ROOT))]
    with tempfile.TemporaryDirectory(prefix="social-dlq-frontend-strong-") as temp:
        sandbox = Path(temp) / "frontend"
        copy_frontend(sandbox)
        strong_tree = tree_hashes(sandbox)
        plan = Path(temp) / "frontend-plan.json"
        frontend_plan(plan, payload)
        baseline = subprocess.run(["npm", "test", "--", "--run", "src/lib/uid.test.js"], cwd=sandbox)
        if baseline.returncode:
            raise RuntimeError("unpatched frontend ordinary baseline failed in disposable copy")
        strong_report = Path(temp) / "frontend-strong-mutation.json"
        config = sandbox / "stryker.proof.mjs"
        strong_config = frontend_config(["src/lib/uid.js"], strong_report, write_frontend_proof_config(sandbox))
        config.write_text(strong_config)
        strong = subprocess.run(["npx", "stryker", "run", str(config)], cwd=sandbox)
        strong_verifier = subprocess.run(
            [sys.executable, str(ROOT / "scripts/quality/verify_mutation_report.py"), "frontend", str(strong_report), str(plan)],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        if strong.returncode or strong_verifier.returncode:
            raise RuntimeError("unpatched frontend strong control did not pass in disposable copy")
        runtime_inventory = verify_stryker_inventory(payload, strong_report)
        inventory = [{"file": file, "name": name} for file, name in sorted(runtime_inventory)]
        strong_statuses = sorted({mutant["status"] for file in json.loads(strong_report.read_text())["files"].values() for mutant in file["mutants"]})
        strong_retained = retain_report("frontend", "strong", strong_report)
        strong_copy_id = Path(temp).name
    with tempfile.TemporaryDirectory(prefix="social-dlq-frontend-weak-") as temp:
        sandbox = Path(temp) / "frontend"
        copy_frontend(sandbox)
        before = tree_hashes(sandbox)
        subprocess.run(["patch", "-p1", "-i", str(ROOT / payload["patch"])], cwd=temp, check=True)
        after = tree_hashes(sandbox)
        changed_paths = sorted(path for path in before.keys() | after.keys() if before.get(path) != after.get(path))
        validate_tree_change(before, after, payload["strong_test"].removeprefix("frontend/"), payload["weak_sha256"])
        if digest(sandbox / payload["target"].removeprefix("frontend/")) != digest(ROOT / payload["target"]):
            raise RuntimeError("frontend weak patch changed production target")
        baseline = subprocess.run(["npm", "test", "--", "--run", "src/lib/uid.test.js"], cwd=sandbox)
        if baseline.returncode:
            raise RuntimeError("weak frontend ordinary baseline failed")
        report = Path(temp) / "frontend-weak-mutation.json"
        plan = Path(temp) / "frontend-plan.json"
        frontend_plan(plan, payload)
        config = sandbox / "stryker.proof.mjs"
        weak_config = frontend_config(["src/lib/uid.js"], report, write_frontend_proof_config(sandbox))
        config.write_text(weak_config)
        tool = subprocess.run(["npx", "stryker", "run", str(config)], cwd=sandbox)
        evidence = json.loads(report.read_text())
        statuses = {mutant["status"] for file in evidence["files"].values() for mutant in file["mutants"]}
        verifier = subprocess.run(
            [sys.executable, str(ROOT / "scripts/quality/verify_mutation_report.py"), "frontend", str(report), str(plan)],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        if tool.returncode not in {0, 1} or verifier.returncode == 0 or not statuses.intersection(payload["allowed_weak_statuses"]):
            raise RuntimeError(f"weak frontend suite did not fail for assertion weakness: {sorted(statuses)}")
        weak_retained = retain_report("frontend", "weak", report)
        weak_copy_id = Path(temp).name
    print("weak frontend proof: ordinary tests passed and mutation gate rejected weak assertions")
    canonical_config = frontend_config(["src/lib/uid.js"], Path("<REPORT>"))
    config_sha = hashlib.sha256(canonical_config.encode()).hexdigest()
    return {
        "status": "weak-baseline-green-mutation-rejected",
        "protective_inventory": inventory,
        "production_target_sha256": digest(ROOT / payload["target"]),
        "strong": {"copy_id": strong_copy_id, "ordinary_baseline": 0, "mutation_exit": strong.returncode, "verifier_exit": strong_verifier.returncode, "config_sha256": config_sha, "tree_sha256": mapping_digest(strong_tree), "statuses": strong_statuses, "report": strong_retained},
        "weak": {"copy_id": weak_copy_id, "ordinary_baseline": 0, "mutation_exit": tool.returncode, "verifier_exit": verifier.returncode, "config_sha256": config_sha, "tree_before_sha256": mapping_digest(before), "tree_after_sha256": mapping_digest(after), "changed_paths": changed_paths, "statuses": sorted(statuses), "report": weak_retained},
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("target", choices=("backend", "frontend"))
    args = parser.parse_args()
    before = subprocess.check_output(["git", "status", "--porcelain=v1"], cwd=ROOT)
    try:
        details = backend_weak() if args.target == "backend" else frontend_weak()
    except (OSError, KeyError, ValueError, RuntimeError, json.JSONDecodeError) as error:
        print(f"weak mutation proof failed: {error}", file=sys.stderr)
        return 2
    after = subprocess.check_output(["git", "status", "--porcelain=v1"], cwd=ROOT)
    if before != after:
        print("weak mutation proof changed caller worktree status", file=sys.stderr)
        return 3
    proof_path = ROOT / "quality/mutation-proof.json"
    proof = {"schema_version": 1, "backend": "pending", "frontend": "pending"}
    if proof_path.is_file():
        try:
            existing = json.loads(proof_path.read_text())
            if isinstance(existing, dict) and existing.get("schema_version") == 1:
                proof.update(existing)
        except json.JSONDecodeError:
            pass
    details_path = ROOT / "quality/mutation-proof-details" / f"{args.target}.json"
    details_path.parent.mkdir(parents=True, exist_ok=True)
    details_path.write_text(json.dumps(details, indent=2, sort_keys=True) + "\n")
    proof[args.target] = {
        "status": details["status"],
        "details": {
            "path": details_path.relative_to(ROOT / "quality").as_posix(),
            "sha256": digest(details_path),
        },
    }
    proof_path.parent.mkdir(parents=True, exist_ok=True)
    proof_path.write_text(json.dumps(proof, indent=2, sort_keys=True) + "\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
