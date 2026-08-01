import subprocess
import os
import shutil
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]


class WorkflowContractTest(unittest.TestCase):
    def test_DLQ_TC_030_031_032_033_workflow_contract(self):
        result = subprocess.run(
            ["node", "scripts/quality/verify_workflow.mjs"],
            cwd=ROOT,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("six always-running jobs", result.stdout)

    def test_DLQ_TC_034_make_quality_dag(self):
        makefile = (ROOT / "Makefile").read_text()
        expected_targets = ["quality-static", "test-backend", "test-frontend", "test-integration", "mutation-backend", "mutation-frontend"]
        quality_line = next(line for line in makefile.splitlines() if line.startswith("quality:"))
        self.assertEqual(quality_line.split()[1:], expected_targets)
        self.assertIn("quality-static: verify-traceability", makefile)
        for command in (
            "git diff --check",
            "go build ./...",
            "go vet ./...",
            "npm --prefix frontend run build",
            "docker compose config --quiet",
            "docker-compose.integration.yml config --quiet",
        ):
            self.assertIn(command, makefile)
        with tempfile.TemporaryDirectory(prefix="dlq-make-") as temp:
            bin_dir = Path(temp)
            for command in ("python3", "git", "go", "npm", "docker"):
                executable = bin_dir / command
                executable.write_text(f'#!/usr/bin/env bash\n[ "${{FAKE_FAIL_COMMAND:-}}" = "{command}" ] && exit 41\nexit 0\n')
                executable.chmod(0o755)
            env = os.environ.copy()
            env["PATH"] = str(bin_dir) + os.pathsep + env["PATH"]
            make = shutil.which("make")
            overrides = bin_dir / "overrides.mk"
            override_recipes = "\n".join(f"{target}:\n\t@true" for target in expected_targets[1:])
            overrides.write_text(override_recipes + "\nquality:\n\t@true\n")
            for failed in ("python3", "git", "go", "npm", "docker"):
                with self.subTest(failed=failed):
                    env["FAKE_FAIL_COMMAND"] = failed
                    result = subprocess.run([make, "-f", "Makefile", "-f", str(overrides), "quality"], cwd=ROOT, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                    self.assertNotEqual(result.returncode, 0)
            aggregate = bin_dir / "aggregate.mk"
            aggregate.write_text(
                "\n".join(
                    f"{target}:\n\t@test \"$@\" != \"$(FAIL_TARGET)\" || exit 41"
                    for target in expected_targets
                )
                + "\nquality:\n\t@true\n"
            )
            env["FAKE_FAIL_COMMAND"] = ""
            for failed in expected_targets:
                with self.subTest(aggregate_child=failed):
                    result = subprocess.run(
                        [make, "-f", "Makefile", "-f", str(aggregate), "quality", f"FAIL_TARGET={failed}"],
                        cwd=ROOT,
                        env=env,
                        stdout=subprocess.PIPE,
                        stderr=subprocess.PIPE,
                    )
                    self.assertNotEqual(result.returncode, 0)


if __name__ == "__main__":
    unittest.main()
