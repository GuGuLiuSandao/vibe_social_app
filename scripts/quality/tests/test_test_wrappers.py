import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]


class TestWrapperTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="dlq-wrapper-")
        self.bin = Path(self.temp.name)
        self.path = str(self.bin) + os.pathsep + os.environ["PATH"]

    def tearDown(self):
        self.temp.cleanup()

    def executable(self, name, content):
        path = self.bin / name
        path.write_text(content)
        path.chmod(0o755)

    def test_DLQ_TC_013_backend_wrapper_propagates_runner_and_rejects_zero(self):
        self.executable("go", "#!/usr/bin/env bash\n[ -n \"${FAKE_EXIT:-}\" ] && exit \"$FAKE_EXIT\"\nprintf '%s' \"${FAKE_REPORT:-}\"\n")
        env = os.environ.copy()
        env["PATH"] = self.path
        env["FAKE_EXIT"] = "29"
        result = subprocess.run(["python3", "scripts/quality/run_backend_tests.py"], cwd=ROOT, env=env)
        self.assertEqual(result.returncode, 29)
        env.pop("FAKE_EXIT")
        env["FAKE_REPORT"] = ""
        result = subprocess.run(["python3", "scripts/quality/run_backend_tests.py"], cwd=ROOT, env=env)
        self.assertNotEqual(result.returncode, 0)

    def test_DLQ_TC_015_frontend_wrapper_propagates_runner_and_rejects_bad_report(self):
        self.executable("npm", "#!/usr/bin/env bash\n[ -n \"${FAKE_EXIT:-}\" ] && exit \"$FAKE_EXIT\"\nfor arg in \"$@\"; do case \"$arg\" in --outputFile=*) cp \"$FAKE_REPORT_FILE\" \"${arg#--outputFile=}\";; esac; done\n")
        env = os.environ.copy()
        env["PATH"] = self.path
        env["FAKE_EXIT"] = "29"
        result = subprocess.run(["python3", "scripts/quality/run_frontend_tests.py"], cwd=ROOT, env=env)
        self.assertEqual(result.returncode, 29)
        env.pop("FAKE_EXIT")
        invalid = self.bin / "invalid.json"
        invalid.write_text("{}")
        env["FAKE_REPORT_FILE"] = str(invalid)
        result = subprocess.run(["python3", "scripts/quality/run_frontend_tests.py"], cwd=ROOT, env=env)
        self.assertNotEqual(result.returncode, 0)


if __name__ == "__main__":
    unittest.main()
