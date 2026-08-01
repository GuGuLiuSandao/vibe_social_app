import json
import unittest

from scripts.quality.run_backend_tests import validate_report as validate_backend
from scripts.quality.run_frontend_tests import validate_report as validate_frontend


def go_event(action, package=None, test=None):
    event = {"Action": action}
    if package is not None:
        event["Package"] = package
    if test is not None:
        event["Test"] = test
    return json.dumps(event)


class BackendReportContractTest(unittest.TestCase):
    def valid(self):
        lines = []
        for index in range(3):
            lines.append(go_event("run", "social_app/internal/auth", f"TestAuth{index}"))
            lines.append(go_event("pass", "social_app/internal/auth", f"TestAuth{index}"))
            lines.append(go_event("run", "social_app/internal/websocket", f"TestWS{index}"))
            lines.append(go_event("pass", "social_app/internal/websocket", f"TestWS{index}"))
        return "\n".join(lines)

    def test_DLQ_TC_012_valid_backend_report(self):
        self.assertTrue(validate_backend(self.valid())[0])

    def test_DLQ_TC_013_backend_report_negative_matrix(self):
        fixtures = {
            "empty": "",
            "malformed": "{broken",
            "non-object": "[]",
            "pass-without-run": "\n".join(go_event("pass", "social_app/internal/auth", f"Test{i}") for i in range(6)),
            "five tests": "\n".join(go_event("run", "social_app/internal/auth", f"Test{i}") for i in range(5)),
            "missing websocket": "\n".join(go_event("run", "social_app/internal/auth", f"Test{i}") for i in range(6)),
            "bad identity type": json.dumps({"Action": "run", "Package": 7, "Test": "Test"}),
        }
        for name, fixture in fixtures.items():
            with self.subTest(name=name):
                self.assertFalse(validate_backend(fixture)[0])


class FrontendReportContractTest(unittest.TestCase):
    def valid(self):
        return {
            "numTotalTestSuites": 2,
            "numPassedTestSuites": 2,
            "numFailedTestSuites": 0,
            "numTotalTests": 6,
            "numPassedTests": 6,
            "numFailedTests": 0,
            "testResults": [{"name": "/repo/uid.test.js"}, {"name": "/repo/ws.test.js"}],
        }

    def test_DLQ_TC_014_valid_frontend_report(self):
        self.assertTrue(validate_frontend(self.valid())[0])

    def test_DLQ_TC_015_frontend_report_negative_matrix(self):
        fixtures = [None, [], {}, {**self.valid(), "numTotalTests": "6"}, {**self.valid(), "numTotalTests": 5, "numPassedTests": 5}, {**self.valid(), "numFailedTests": 1, "numPassedTests": 5}, {**self.valid(), "testResults": [{"name": "/repo/uid.test.js"}]}, {**self.valid(), "testResults": [{"name": "/repo/other.test.js"}, {"name": "/repo/ws.test.js"}]}]
        for fixture in fixtures:
            with self.subTest(fixture=fixture):
                self.assertFalse(validate_frontend(fixture)[0])


if __name__ == "__main__":
    unittest.main()
