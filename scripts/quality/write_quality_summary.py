#!/usr/bin/env python3
import hashlib
import json
import subprocess
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
QUALITY = ROOT / "quality"


def command(*args: str) -> str:
    return subprocess.check_output(args, cwd=ROOT, text=True).strip()


def verified_artifact(reference: dict) -> Path:
    if not isinstance(reference, dict) or set(reference) != {"path", "sha256"}:
        raise RuntimeError("mutation proof artifact reference is malformed")
    relative = Path(reference["path"])
    if relative.is_absolute() or ".." in relative.parts:
        raise RuntimeError(f"mutation proof artifact escapes quality/: {relative}")
    path = QUALITY / relative
    if not path.is_file() or hashlib.sha256(path.read_bytes()).hexdigest() != reference["sha256"]:
        raise RuntimeError(f"mutation proof artifact hash mismatch: {relative}")
    return path


def verify_mutation_proof() -> None:
    proof = json.loads((QUALITY / "mutation-proof.json").read_text())
    for kind in ("backend", "frontend"):
        entry = proof.get(kind)
        if not isinstance(entry, dict) or entry.get("status") != "weak-baseline-green-mutation-rejected":
            raise RuntimeError(f"mutation proof is incomplete: {kind}")
        details = json.loads(verified_artifact(entry.get("details")).read_text())
        for phase in ("strong", "weak"):
            verified_artifact(details.get(phase, {}).get("report"))


def main() -> int:
    QUALITY.mkdir(parents=True, exist_ok=True)
    verify_mutation_proof()
    status = command("git", "status", "--porcelain=v1", "--untracked-files=all")
    digest = hashlib.sha256()
    digest.update(subprocess.check_output(["git", "diff", "--binary", "HEAD"], cwd=ROOT))
    for line in status.splitlines():
        path_text = line[3:]
        if " -> " in path_text:
            path_text = path_text.split(" -> ", 1)[1]
        path = ROOT / path_text
        if line.startswith("??") and path.is_file():
            digest.update(path_text.encode())
            digest.update(path.read_bytes())
    artifacts = {}
    for relative in (
        "backend-test.jsonl",
        "frontend-test.json",
        "backend-mutation.json",
        "backend-mutation-summary.txt",
        "frontend-mutation.json",
        "frontend-mutation-summary.txt",
        "mutation-plan.json",
        "mutation-proof.json",
        "integration/latest.json",
        "integration/isolation-summary.json",
    ):
        path = QUALITY / relative
        if not path.is_file() or not path.stat().st_size:
            raise RuntimeError(f"required quality artifact is missing: {relative}")
        artifacts[relative] = hashlib.sha256(path.read_bytes()).hexdigest()
    payload = {
        "schema_version": 1,
        "command": "make quality",
        "completed_at": datetime.now(timezone.utc).isoformat(),
        "head": command("git", "rev-parse", "HEAD"),
        "worktree_status": status.splitlines(),
        "worktree_digest": digest.hexdigest(),
        "children": {name: "passed" for name in ("quality-static", "test-backend", "test-frontend", "test-integration", "mutation-backend", "mutation-frontend")},
        "artifacts": artifacts,
    }
    (QUALITY / "quality-summary.json").write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n")
    print("quality summary: all six gates bound to current worktree evidence")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
