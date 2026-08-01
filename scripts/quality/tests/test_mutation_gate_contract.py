import hashlib
import json
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

from scripts.quality.mutation_config import frontend_config
from scripts.quality import run_frontend_mutation as frontend_runner
from scripts.quality import verify_weak_mutation as weak_proof

ROOT = Path(__file__).resolve().parents[3]
FIXTURES = ROOT / "scripts/quality/tests/fixtures/weak-suites"


class MutationGateContractTest(unittest.TestCase):
    def test_DLQ_TC_027_backend_wrapper_binds_plan_report_and_weak_proof(self):
        wrapper = (ROOT / "scripts/quality/run_backend_mutation.sh").read_text()
        self.assertIn('verify_mutation_report.py" backend "$REPORT" "$PLAN"', wrapper)
        self.assertIn('verify_weak_mutation.py" backend', wrapper)
        self.assertIn("--threshold-mcover 100", wrapper)
        self.assertIn("go version -m", wrapper)

    def test_DLQ_TC_028_frontend_wrapper_binds_plan_report_and_weak_proof(self):
        wrapper = (ROOT / "scripts/quality/run_frontend_mutation.py").read_text()
        self.assertIn('"frontend", str(REPORT), str(PLAN)', wrapper)
        self.assertIn('plan["frontend_files"]', wrapper)
        self.assertIn("verify_weak_mutation.py", wrapper)
        changed_targets = [path.removeprefix("frontend/") for path in ["frontend/src/lib/ws.js"]]
        config = frontend_config(changed_targets, Path("changed-report.json"))
        self.assertIn('["src/lib/ws.js"]', config)
        self.assertNotIn("src/lib/uid.js", config)
        with tempfile.TemporaryDirectory(prefix="dlq-frontend-wrapper-") as temp:
            root = Path(temp)
            frontend = root / "frontend"
            frontend.mkdir()
            captured = []
            plan = {
                "schema_version": 1,
                "frontend_smoke": False,
                "frontend_files": ["frontend/src/lib/ws.js"],
            }

            def fake_run(command, **kwargs):
                rendered = " ".join(map(str, command))
                if "mutation_targets.py" in rendered:
                    return SimpleNamespace(returncode=0, stdout=json.dumps(plan), stderr="")
                if command[:3] == ["npx", "stryker", "run"]:
                    captured.append(Path(command[3]).read_text())
                if "verify_mutation_report.py" in rendered:
                    return SimpleNamespace(returncode=0, stdout="verified\n", stderr="")
                return SimpleNamespace(returncode=0, stdout="", stderr="")

            with (
                patch.object(frontend_runner, "ROOT", root),
                patch.object(frontend_runner, "FRONTEND", frontend),
                patch.object(frontend_runner, "PLAN", root / "quality/mutation-plan.json"),
                patch.object(frontend_runner, "REPORT", root / "quality/frontend-mutation.json"),
                patch.object(frontend_runner.subprocess, "run", side_effect=fake_run),
            ):
                self.assertEqual(frontend_runner.main(), 0)
            self.assertEqual(len(captured), 1)
            self.assertIn('["src/lib/ws.js"]', captured[0])
            self.assertNotIn("src/lib/uid.js", captured[0])

    def test_DLQ_TC_029_weak_manifests_patch_exact_strong_tests(self):
        for name in ("backend-jwt-v1", "frontend-uid-v1"):
            payload = json.loads((FIXTURES / f"{name}.json").read_text())
            strong = ROOT / payload["strong_test"]
            weak = ROOT / payload["weak_replacement"]
            patch = ROOT / payload["patch"]
            self.assertEqual(hashlib.sha256(strong.read_bytes()).hexdigest(), payload["strong_sha256"])
            self.assertEqual(hashlib.sha256(weak.read_bytes()).hexdigest(), payload["weak_sha256"])
            patch_text = patch.read_text()
            self.assertIn(payload["strong_test"], patch_text)
            self.assertNotIn("/src/lib/uid.js\n", patch_text)
            self.assertNotIn("/internal/auth/jwt.go\n", patch_text)
            for test_name in payload["replaced_tests"]:
                self.assertIn(test_name, strong.read_text())
        proof = (ROOT / "scripts/quality/verify_weak_mutation.py").read_text()
        self.assertIn("unpatched backend strong control", proof)
        self.assertIn("unpatched frontend strong control", proof)
        self.assertEqual(proof.count("frontend_config([\"src/lib/uid.js\"]"), 3)
        self.assertIn("social-dlq-backend-strong-", proof)
        self.assertIn("social-dlq-backend-weak-", proof)
        self.assertIn("social-dlq-frontend-strong-", proof)
        self.assertIn("social-dlq-frontend-weak-", proof)
        summary = (ROOT / "scripts/quality/write_quality_summary.py").read_text()
        self.assertIn("verify_mutation_proof()", summary)
        self.assertIn("mutation proof artifact hash mismatch", summary)

        for name, kind in (("backend-jwt-v1", "backend"), ("frontend-uid-v1", "frontend")):
            payload = weak_proof.manifest(name)
            weak_proof.verify_protective_inventory(payload, kind, ROOT)
            if kind == "backend":
                broken = dict(payload)
                broken["protective_tests"] = payload["protective_tests"][:-1]
                with self.assertRaisesRegex(RuntimeError, "protective test inventory mismatch"):
                    weak_proof.verify_protective_inventory(broken, kind, ROOT)

        frontend_payload = json.loads((FIXTURES / "frontend-uid-v1.json").read_text())
        tests = [
            {"id": str(index), "name": item["name"]}
            for index, item in enumerate(frontend_payload["protective_tests"])
        ]
        report_payload = {
            "testFiles": {"src/lib/uid.test.js": {"tests": tests}},
            "files": {"src/lib/uid.js": {"mutants": [{"coveredBy": [test["id"] for test in tests]}]}},
        }
        with tempfile.TemporaryDirectory(prefix="dlq-stryker-inventory-") as temp:
            report = Path(temp) / "report.json"
            report.write_text(json.dumps(report_payload))
            weak_proof.verify_stryker_inventory(frontend_payload, report)
            broken = dict(frontend_payload)
            broken["protective_tests"] = frontend_payload["protective_tests"][:-1]
            with self.assertRaisesRegex(RuntimeError, "Stryker protective identity mismatch"):
                weak_proof.verify_stryker_inventory(broken, report)

        with tempfile.TemporaryDirectory(prefix="dlq-inventory-drift-") as temp:
            root = Path(temp)
            auth = root / "backend/internal/auth"
            auth.mkdir(parents=True)
            (auth / "jwt_test.go").write_text((ROOT / "backend/internal/auth/jwt_test.go").read_text())
            (auth / "extra_test.go").write_text(
                "package auth\nfunc TestUnlistedProtection(t *testing.T) { _, _ = GenerateToken(1, \"x\", nil) }\n"
            )
            payload = json.loads((FIXTURES / "backend-jwt-v1.json").read_text())
            with self.assertRaisesRegex(RuntimeError, "TestUnlistedProtection"):
                weak_proof.verify_protective_inventory(payload, "backend", root)

            frontend = root / "frontend/src/lib"
            frontend.mkdir(parents=True)
            (frontend / "uid.test.js").write_text((ROOT / "frontend/src/lib/uid.test.js").read_text())
            (frontend / "extra.test.js").write_text(
                'it("unlisted direct caller", () => { parseUid("1"); });\n'
            )
            payload = json.loads((FIXTURES / "frontend-uid-v1.json").read_text())
            with self.assertRaisesRegex(RuntimeError, "extra.test.js"):
                weak_proof.verify_protective_inventory(payload, "frontend", root)

        with tempfile.TemporaryDirectory(prefix="dlq-tree-integrity-") as temp:
            root = Path(temp)
            protected = root / "protected.test"
            protected.write_text("strong")
            before = weak_proof.tree_hashes(root)
            protected.write_text("weak")
            weak_sha = hashlib.sha256(b"weak").hexdigest()
            weak_proof.validate_tree_change(before, weak_proof.tree_hashes(root), "protected.test", weak_sha)
            (root / "unexpected").write_text("drift")
            with self.assertRaisesRegex(RuntimeError, "unexpected paths"):
                weak_proof.validate_tree_change(before, weak_proof.tree_hashes(root), "protected.test", weak_sha)


if __name__ == "__main__":
    unittest.main()
