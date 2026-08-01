#!/usr/bin/env python3
import json
import os
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


class DiscoveryError(ValueError):
    pass


def git(root: Path, *args: str, check: bool = True) -> subprocess.CompletedProcess:
    result = subprocess.run(["git", *args], cwd=root, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if check and result.returncode:
        raise DiscoveryError(result.stderr.strip() or f"git {' '.join(args)} failed")
    return result


def package_for(path: str) -> str:
    directory = Path(path).parent.relative_to("backend")
    return "social_app/" + directory.as_posix()


def package_files(root: Path, package: str) -> list[str]:
    directory = root / "backend" / package.removeprefix("social_app/")
    files = []
    for path in directory.glob("*.go"):
        relative = path.relative_to(root).as_posix()
        if not path.name.endswith("_test.go") and "/internal/proto/" not in relative:
            files.append(relative)
    return sorted(files)


def smoke_plan(reason: str, base: str | None = None) -> dict:
    return {
        "schema_version": 1,
        "base_sha": base,
        "reason": reason,
        "backend_changed_files": [],
        "backend_packages": ["social_app/internal/auth"],
        "backend_allowed_files": ["backend/internal/auth/jwt.go"],
        "frontend_files": ["frontend/src/lib/uid.js"],
        "backend_smoke": True,
        "frontend_smoke": True,
    }


def discover(root: Path, base: str) -> dict:
    for revision in (base, "HEAD"):
        if git(root, "cat-file", "-e", f"{revision}^{{commit}}", check=False).returncode:
            raise DiscoveryError(f"unknown commit: {revision}")
    if git(root, "merge-base", "--is-ancestor", base, "HEAD", check=False).returncode:
        raise DiscoveryError("MUTATION_BASE_SHA is not an ancestor of HEAD")

    raw = git(root, "diff", "--raw", f"{base}...HEAD").stdout
    for line in raw.splitlines():
        header = line.split("\t", 1)[0].split()
        if len(header) >= 2 and (header[0].endswith("160000") or header[1] == "160000"):
            raise DiscoveryError("submodule changes are not supported")

    result = git(root, "diff", "--name-status", "--find-renames", f"{base}...HEAD")
    backend = []
    frontend = []
    saw_record = False
    saw_live_path = False
    for line in result.stdout.splitlines():
        saw_record = True
        fields = line.split("\t")
        state = fields[0]
        if state.startswith("R"):
            if len(fields) != 3 or not state[1:].isdigit():
                raise DiscoveryError(f"invalid rename record: {line}")
            path = fields[2]
        elif state in {"A", "M", "D"}:
            if len(fields) != 2:
                raise DiscoveryError(f"invalid diff record: {line}")
            path = fields[1]
        else:
            raise DiscoveryError(f"unsupported diff state: {state}")
        candidate = Path(path)
        if candidate.is_absolute() or ".." in candidate.parts:
            raise DiscoveryError(f"path escapes repository: {path}")
        if state == "D":
            continue
        saw_live_path = True
        if path.startswith("backend/") and path.endswith(".go") and not path.endswith("_test.go") and "/internal/proto/" not in path:
            backend.append(path)
        if path.startswith("frontend/src/") and candidate.suffix in {".js", ".jsx", ".ts", ".tsx"} and ".test." not in path and "/proto/" not in path:
            frontend.append(path)

    if not saw_record:
        reason = "empty"
    elif not saw_live_path:
        reason = "deleted-only"
    elif not backend and not frontend:
        reason = "not-applicable"
    else:
        reason = "changed"
    backend = sorted(set(backend))
    frontend = sorted(set(frontend))
    packages = sorted({package_for(path) for path in backend})
    backend_smoke = not backend
    frontend_smoke = not frontend
    if backend_smoke:
        packages = ["social_app/internal/auth"]
        allowed = ["backend/internal/auth/jwt.go"]
    else:
        allowed = sorted({path for package in packages for path in package_files(root, package)})
    if frontend_smoke:
        frontend = ["frontend/src/lib/uid.js"]
    return {
        "schema_version": 1,
        "base_sha": base,
        "reason": reason,
        "backend_changed_files": backend,
        "backend_packages": packages,
        "backend_allowed_files": allowed,
        "frontend_files": frontend,
        "backend_smoke": backend_smoke,
        "frontend_smoke": frontend_smoke,
    }


def main() -> int:
    base = os.environ.get("MUTATION_BASE_SHA", "")
    try:
        if not base:
            if os.environ.get("CI", "").lower() == "true":
                raise DiscoveryError("MUTATION_BASE_SHA is required in CI")
            payload = smoke_plan("local-uncommitted")
        else:
            payload = discover(ROOT, base)
    except DiscoveryError as error:
        print(error, file=sys.stderr)
        return 2
    print(json.dumps(payload, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
