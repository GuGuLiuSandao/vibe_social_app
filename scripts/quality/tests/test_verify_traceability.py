import copy
import json
import tempfile
import unittest
from pathlib import Path

from scripts.quality.verify_traceability import CASES, MANIFEST, REQUIREMENT, ROOT, validate


class TraceabilityVerifierTest(unittest.TestCase):
    def test_DLQ_TC_035_valid_manifest_resolves_all_P0_cases(self):
        self.assertEqual(validate(ROOT, MANIFEST, CASES, REQUIREMENT), [])

    def test_DLQ_TC_036_negative_manifest_matrix_fails_closed(self):
        original = json.loads(MANIFEST.read_text())
        mutations = []
        mutations.append(([], "manifest root"))
        missing_case = copy.deepcopy(original)
        missing_case["entries"].pop()
        mutations.append((missing_case, "uncovered P0 Cases"))
        unknown_case = copy.deepcopy(original)
        unknown_case["entries"][0]["case_id"] = "DLQ-TC-999"
        mutations.append((unknown_case, "unknown or non-P0 Case ID"))
        duplicate = copy.deepcopy(original)
        duplicate["entries"].append(copy.deepcopy(duplicate["entries"][0]))
        mutations.append((duplicate, "duplicate"))
        bad_path = copy.deepcopy(original)
        bad_path["entries"][0]["test_file"] = "../escape.py"
        mutations.append((bad_path, "path escapes repository"))
        partial_name = copy.deepcopy(original)
        partial_name["entries"][0]["test_name"] += "_partial"
        mutations.append((partial_name, "mismatched full test name"))
        unknown_acceptance = copy.deepcopy(original)
        unknown_acceptance["entries"][0]["acceptance_id"] = "DLQ-999"
        mutations.append((unknown_acceptance, "unknown acceptance ID"))
        unknown_spec = copy.deepcopy(original)
        target = next(entry for entry in unknown_spec["entries"] if entry.get("specification_id"))
        target["specification_id"] = "UNKNOWN-001"
        mutations.append((unknown_spec, "unknown specification"))
        missing_spec = copy.deepcopy(original)
        target = next(entry for entry in missing_spec["entries"] if entry.get("specification_id"))
        target["specification_file"] = "docs/specs/missing.md"
        mutations.append((missing_spec, "missing specification document"))
        missing_test = copy.deepcopy(original)
        missing_test["entries"][0]["test_file"] = "scripts/quality/tests/missing.py"
        mutations.append((missing_test, "missing test file"))
        no_case = copy.deepcopy(original)
        no_case["entries"][0]["test_file"] = "scripts/quality/tests/fixtures/traceability/no_case_test.py"
        no_case["entries"][0]["test_name"] = "valid_fixture_test"
        mutations.append((no_case, "test file lacks exact Case ID"))
        uncovered_spec = copy.deepcopy(original)
        for entry in uncovered_spec["entries"]:
            if entry.get("specification_id") == "AUTH-001":
                entry.pop("specification_id")
                entry.pop("specification_file")
        mutations.append((uncovered_spec, "uncovered specifications"))
        unbound_evidence = copy.deepcopy(original)
        unbound_evidence["entries"][0]["evidence"] = "quality/missing.json"
        mutations.append((unbound_evidence, "unresolvable evidence binding"))
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / "manifest.json"
            for payload, category in mutations:
                with self.subTest(category=category):
                    path.write_text(json.dumps(payload))
                    self.assertIn(category, "\n".join(validate(ROOT, path, CASES, REQUIREMENT)))
            path.write_text("{broken")
            self.assertIn("manifest read/schema error", "\n".join(validate(ROOT, path, CASES, REQUIREMENT)))


if __name__ == "__main__":
    unittest.main()
