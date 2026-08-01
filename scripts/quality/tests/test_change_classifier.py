import unittest

from scripts.quality.change_classifier import classify_paths, parse_labels
from scripts.quality.verify_quality_gate import verify


class ChangeClassifierTest(unittest.TestCase):
    def test_docs_only_is_lightweight(self):
        result = classify_paths(["README.md", "docs/PROTO_SETUP.md"], set())
        self.assertEqual(result.name, "docs")

    def test_engineering_paths_are_not_treated_as_docs(self):
        for path in ("AGENTS.md", ".github/workflows/quality.yml", "scripts/quality/check.py", "docs/specs/authentication.md"):
            with self.subTest(path=path):
                self.assertEqual(classify_paths([path], set()).name, "engineering")

    def test_product_change_requires_develop_label(self):
        result = classify_paths(["backend/internal/auth/handler.go"], set())
        self.assertEqual(result.name, "invalid")
        self.assertIn("develop-loop", result.reason)
        self.assertEqual(classify_paths(["backend/internal/auth/handler.go"], {"develop-loop"}).name, "develop")

    def test_label_can_only_increase_checks(self):
        self.assertEqual(classify_paths(["README.md"], {"develop-loop"}).name, "develop")

    def test_unknown_paths_fail_closed_even_when_labeled(self):
        self.assertEqual(classify_paths(["new-runtime/main.rs"], {"develop-loop"}).name, "invalid")

    def test_labels_accept_json_or_csv(self):
        self.assertEqual(parse_labels('["documentation", "develop-loop"]'), {"documentation", "develop-loop"})
        self.assertEqual(parse_labels("documentation,develop-loop"), {"documentation", "develop-loop"})

    def test_final_gate_requires_only_the_selected_branch(self):
        results = {"classify": "success", "docs": "success", "engineering": "skipped"}
        self.assertEqual(verify("docs", results), [])
        self.assertTrue(verify("engineering", results))
        self.assertTrue(verify("invalid", results))


if __name__ == "__main__":
    unittest.main()
