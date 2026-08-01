import json
import os
import signal
import subprocess
import tempfile
import time
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
RUNNER = ROOT / "scripts/quality/run_integration.sh"
VERIFIER = ROOT / "scripts/quality/verify_integration_report.py"

DOCKER_FAKE = r'''#!/usr/bin/env bash
echo "$*" >>"$FAKE_CALL_LOG"
if [ "$1" = "inspect" ]; then echo healthy; exit 0; fi
for arg in "$@"; do
  case "$arg" in
    up) [ "${FAKE_FAIL_STAGE:-}" = up ] && exit 17; exit 0 ;;
    logs) exit 0 ;;
    down) [ "${FAKE_FAIL_STAGE:-}" = cleanup ] && exit 19; exit 0 ;;
    port) [ "${FAKE_FAIL_STAGE:-}" = port ] && exit 0; echo 127.0.0.1:43123; exit 0 ;;
    run)
      [ "${FAKE_FAIL_STAGE:-}" = test ] && exit 23
      [ "${FAKE_FAIL_STAGE:-}" = block ] && { while :; do sleep 1; done; }
      if [ "${FAKE_FAIL_STAGE:-}" = verifier ]; then echo broken; exit 0; fi
      printf '%s\n' '{"Action":"pass","Package":"social_app/integration","Test":"TestDLQ_TC_011_016_AUTH_HTTP_001_RegisterLogin"}'
      printf '%s\n' '{"Action":"pass","Package":"social_app/integration","Test":"TestDLQ_TC_017_WS_HTTP_001_AuthenticatedPingPong"}'
      exit 0 ;;
    ps)
      for value in "$@"; do [ "$value" = "-q" ] && { echo fake-container; exit 0; }; done
      exit 0 ;;
  esac
done
exit 0
'''

CURL_FAKE = r'''#!/usr/bin/env bash
if [ "${FAKE_FAIL_STAGE:-}" = readiness ]; then printf 000; else printf 415; fi
'''


class IntegrationLifecycleTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="dlq-lifecycle-")
        self.root = Path(self.temp.name)
        self.docker = self.root / "docker"
        self.curl = self.root / "curl"
        self.log = self.root / "calls.log"
        self.docker.write_text(DOCKER_FAKE)
        self.curl.write_text(CURL_FAKE)
        self.docker.chmod(0o755)
        self.curl.chmod(0o755)

    def tearDown(self):
        self.temp.cleanup()

    def run_wrapper(self, stage=""):
        env = os.environ.copy()
        env.update({
            "QUALITY_DOCKER_BIN": str(self.docker),
            "QUALITY_CURL_BIN": str(self.curl),
            "QUALITY_UPDATE_LATEST": "0",
            "FAKE_CALL_LOG": str(self.log),
            "FAKE_FAIL_STAGE": stage,
        })
        return subprocess.run(["bash", str(RUNNER)], cwd=ROOT, env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    def test_DLQ_TC_019_failures_diagnose_then_cleanup_once(self):
        for stage in ("up", "port", "test", "verifier", "cleanup"):
            with self.subTest(stage=stage):
                self.log.write_text("")
                result = self.run_wrapper(stage)
                self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
                calls = self.log.read_text().splitlines()
                downs = [index for index, call in enumerate(calls) if " down " in f" {call} "]
                self.assertEqual(len(downs), 1, calls)
                logs = [index for index, call in enumerate(calls) if " logs " in f" {call} "]
                self.assertTrue(logs and logs[0] < downs[0], calls)

    def test_DLQ_TC_020_signal_traps_are_fail_closed(self):
        env = os.environ.copy()
        env.update({"QUALITY_DOCKER_BIN": str(self.docker), "QUALITY_CURL_BIN": str(self.curl), "QUALITY_UPDATE_LATEST": "0", "FAKE_CALL_LOG": str(self.log), "FAKE_FAIL_STAGE": "block"})
        process = subprocess.Popen(["bash", str(RUNNER)], cwd=ROOT, env=env, start_new_session=True)
        deadline = time.time() + 10
        while time.time() < deadline:
            if self.log.is_file() and " run " in f" {self.log.read_text()} ":
                break
            time.sleep(0.05)
        else:
            process.kill()
            self.fail("wrapper did not reach blocking test stage")
        os.killpg(process.pid, signal.SIGTERM)
        self.assertEqual(process.wait(timeout=10), 143)
        calls = self.log.read_text().splitlines()
        downs = [index for index, call in enumerate(calls) if " down " in f" {call} "]
        logs = [index for index, call in enumerate(calls) if " logs " in f" {call} "]
        self.assertEqual(len(downs), 1, calls)
        self.assertTrue(logs and logs[0] < downs[0], calls)

    def test_DLQ_TC_021_cleanup_is_scoped_to_generated_project(self):
        result = self.run_wrapper()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        calls = self.log.read_text().splitlines()
        projects = []
        for call in calls:
            parts = call.split()
            if "-p" in parts:
                projects.append(parts[parts.index("-p") + 1])
        self.assertTrue(projects)
        self.assertEqual(len(set(projects)), 1)
        self.assertRegex(projects[0], r"^social-app-it-\d+-\d+$")
        self.assertNotIn("sentinel", " ".join(calls))

    def test_DLQ_TC_022_parallel_runs_allocate_distinct_projects(self):
        env = os.environ.copy()
        env.update({"QUALITY_DOCKER_BIN": str(self.docker), "QUALITY_CURL_BIN": str(self.curl), "QUALITY_UPDATE_LATEST": "0", "FAKE_CALL_LOG": str(self.log)})
        first = subprocess.Popen(["bash", str(RUNNER)], cwd=ROOT, env=env)
        second = subprocess.Popen(["bash", str(RUNNER)], cwd=ROOT, env=env)
        self.assertEqual(first.wait(timeout=20), 0)
        self.assertEqual(second.wait(timeout=20), 0)
        projects = set()
        for call in self.log.read_text().splitlines():
            parts = call.split()
            if "-p" in parts:
                projects.add(parts[parts.index("-p") + 1])
        self.assertEqual(len(projects), 2, projects)


class IntegrationReportTest(unittest.TestCase):
    def test_DLQ_TC_018_report_requires_two_independent_flows(self):
        with tempfile.TemporaryDirectory() as temp:
            report = Path(temp) / "report.jsonl"
            auth = {"Action": "pass", "Test": "TestDLQ_TC_011_016_AUTH_HTTP_001_RegisterLogin"}
            ws = {"Action": "pass", "Test": "TestDLQ_TC_017_WS_HTTP_001_AuthenticatedPingPong"}
            report.write_text(json.dumps(auth) + "\n" + json.dumps(ws) + "\n")
            self.assertEqual(subprocess.run(["python3", str(VERIFIER), str(report)]).returncode, 0)
            report.write_text(json.dumps(auth) + "\n")
            self.assertNotEqual(subprocess.run(["python3", str(VERIFIER), str(report)]).returncode, 0)
            report.write_text("broken\n")
            self.assertNotEqual(subprocess.run(["python3", str(VERIFIER), str(report)]).returncode, 0)


if __name__ == "__main__":
    unittest.main()
