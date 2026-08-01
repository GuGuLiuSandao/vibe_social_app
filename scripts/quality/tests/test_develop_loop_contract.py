import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]


class DevelopLoopContractTest(unittest.TestCase):
    def setUp(self):
        self.readme = (ROOT / ".engineering-loop/develop/README.md").read_text()

    def test_DLQ_TC_001_standard_order_and_no_quick_bypass(self):
        expected = [
            "Requirement Contract",
            "Technical Design",
            "Design Review",
            "Test Design",
            "Test Review",
            "Implementation",
            "Code Review",
            "Local Quality Gates",
        ]
        flow = re.search(r"```text\n(.*?)\n```", self.readme, re.DOTALL).group(1)
        positions = [flow.index(stage) for stage in expected]
        self.assertEqual(positions, sorted(positions))
        self.assertIn("不省略独立 Test Design、Test Review 和 Code Review", self.readme)
        self.assertIn("PASS", self.readme)

    def test_DLQ_TC_002_independent_roles_are_write_bounded(self):
        designer = (ROOT / ".codex/agents/develop-test-designer.toml").read_text()
        reviewer = (ROOT / ".codex/agents/develop-test-reviewer.toml").read_text()
        design_reviewer = (ROOT / ".codex/agents/develop-design-reviewer.toml").read_text()
        implementer = (ROOT / ".codex/agents/develop-implementer.toml").read_text()
        code_reviewer = (ROOT / ".codex/agents/develop-code-reviewer.toml").read_text()
        self.assertIn("只维护当前 change 的 testcases.md", designer)
        self.assertIn("只维护当前 change 的 testcases-review.md", reviewer)
        self.assertIn("不修改测试设计或代码", reviewer)
        self.assertIn("只维护当前 change 的 design-review.md", design_reviewer)
        self.assertIn("已通过的 design/design-review", implementer)
        self.assertIn("testcases/testcases-review", implementer)
        self.assertIn("不修改", code_reviewer)
        self.assertIn("blocker", reviewer)
        self.assertIn("三轮", reviewer)


if __name__ == "__main__":
    unittest.main()
