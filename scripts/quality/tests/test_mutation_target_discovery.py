import subprocess
import tempfile
import unittest
from pathlib import Path

from scripts.quality.mutation_targets import DiscoveryError, discover


class MutationTargetDiscoveryTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="dlq-git-")
        self.root = Path(self.temp.name)
        self.git("init", "-q")
        self.git("config", "user.email", "test@example.test")
        self.git("config", "user.name", "Test")
        (self.root / "README.md").write_text("base\n")
        self.git("add", ".")
        self.git("commit", "-qm", "base")
        self.base = self.git("rev-parse", "HEAD").stdout.strip()
        self.main_branch = self.git("rev-parse", "--abbrev-ref", "HEAD").stdout.strip()

    def tearDown(self):
        self.temp.cleanup()

    def git(self, *args):
        return subprocess.run(["git", *args], cwd=self.root, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True)

    def commit_file(self, path, content):
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content)
        self.git("add", ".")
        self.git("commit", "-qm", path)

    def test_DLQ_TC_023_changed_files_map_to_trusted_targets(self):
        self.commit_file("backend/internal/auth/jwt.go", "package auth\nfunc f(){ _ = 1 + 2 }\n")
        plan = discover(self.root, self.base)
        self.assertEqual(plan["backend_changed_files"], ["backend/internal/auth/jwt.go"])
        self.assertEqual(plan["backend_packages"], ["social_app/internal/auth"])
        self.assertIn("backend/internal/auth/jwt.go", plan["backend_allowed_files"])
        self.assertTrue(plan["frontend_smoke"])

    def test_DLQ_TC_023_empty_docs_deleted_and_rename_classes(self):
        self.assertEqual(discover(self.root, self.base)["reason"], "empty")
        self.commit_file("docs/note.md", "note\n")
        self.assertEqual(discover(self.root, self.base)["reason"], "not-applicable")
        self.git("rm", "docs/note.md")
        self.git("commit", "-qm", "delete")
        # The aggregate base includes add+delete and therefore resolves empty; use the add commit as deletion base.
        deletion_base = self.git("rev-parse", "HEAD^").stdout.strip()
        self.assertEqual(discover(self.root, deletion_base)["reason"], "deleted-only")

    def test_DLQ_TC_023_unknown_nonancestor_and_submodule_fail(self):
        with self.assertRaises(DiscoveryError):
            discover(self.root, "deadbeef")
        self.git("checkout", "-qb", "other", self.base)
        self.commit_file("other.txt", "other\n")
        other = self.git("rev-parse", "HEAD").stdout.strip()
        self.git("checkout", "-q", self.main_branch)
        self.commit_file("main.txt", "main\n")
        with self.assertRaises(DiscoveryError):
            discover(self.root, other)
        blob = self.git("rev-parse", "HEAD").stdout.strip()
        self.git("update-index", "--add", "--cacheinfo", f"160000,{blob},vendor/sub")
        self.git("commit", "-qm", "gitlink")
        with self.assertRaises(DiscoveryError):
            discover(self.root, self.base)


if __name__ == "__main__":
    unittest.main()
