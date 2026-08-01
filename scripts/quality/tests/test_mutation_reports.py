import copy
import unittest

from scripts.quality.verify_mutation_report import backend, frontend, validate_plan


PLAN = {
    "schema_version": 1,
    "base_sha": None,
    "reason": "test",
    "backend_changed_files": [],
    "backend_mutation_targets": {"backend/internal/auth/jwt.go": None},
    "backend_packages": ["social_app/internal/auth"],
    "backend_allowed_files": ["backend/internal/auth/jwt.go"],
    "frontend_files": ["frontend/src/lib/uid.js"],
    "frontend_mutation_targets": ["src/lib/uid.js"],
    "backend_smoke": True,
    "frontend_smoke": True,
}


class MutationReportTest(unittest.TestCase):
    def backend_valid(self):
        return {"files": [{"file_name": "jwt.go", "mutations": [{"status": "KILLED"}]}], "mutants_total": 1, "mutants_killed": 1, "mutants_not_viable": 0, "mutants_lived": 0, "mutants_not_covered": 0}

    def test_DLQ_TC_025_gremlins_matrix_and_scope_fail_closed(self):
        self.assertTrue(backend(self.backend_valid(), PLAN)[0])
        for state in ("LIVED", "NOT COVERED", "TIMED OUT", "RUNNABLE", "UNKNOWN", None):
            fixture = self.backend_valid()
            fixture["files"][0]["mutations"][0]["status"] = state
            fixture["mutants_killed"] = 0
            with self.subTest(state=state):
                self.assertFalse(backend(fixture, PLAN)[0])
        for mutation in (
            lambda value: value.update({"mutants_total": 0}),
            lambda value: value.update({"mutants_total": "1"}),
            lambda value: value["files"][0].update({"file_name": "outside.go"}),
            lambda value: value.update({"files": []}),
            lambda value: value.update({"mutants_killed": 2}),
        ):
            fixture = self.backend_valid()
            mutation(fixture)
            self.assertFalse(backend(fixture, PLAN)[0])

    def test_DLQ_TC_026_stryker_matrix_and_exact_scope_fail_closed(self):
        valid = {"files": {"src/lib/uid.js": {"mutants": [{"status": "Killed"}]}}}
        self.assertTrue(frontend(valid, PLAN)[0])
        plan_with_zero_mutant_target = {**PLAN, "frontend_files": [*PLAN["frontend_files"], "frontend/src/lib/no-operators.js"]}
        self.assertTrue(frontend(valid, plan_with_zero_mutant_target)[0])
        for state in ("Survived", "NoCoverage", "Timeout", "RuntimeError", "CompileError", "Ignored", "Pending", "Unknown", None):
            with self.subTest(state=state):
                self.assertFalse(frontend({"files": {"src/lib/uid.js": {"mutants": [{"status": state}]}}}, PLAN)[0])
        self.assertFalse(frontend({"files": {}}, PLAN)[0])
        self.assertFalse(frontend({"files": {"src/outside.js": {"mutants": [{"status": "Killed"}]}}}, PLAN)[0])
        self.assertFalse(frontend({"files": {"src/lib/uid.js": {"mutants": [{"status": "Killed"}]}, "src/outside.js": {"mutants": [{"status": "Killed"}]}}}, PLAN)[0])

    def test_DLQ_TC_024_plan_schema_is_strict(self):
        self.assertTrue(validate_plan(PLAN)[0])
        for broken in (None, {}, {**PLAN, "schema_version": 2}, {**PLAN, "frontend_files": []}, {**PLAN, "backend_smoke": "true"}):
            self.assertFalse(validate_plan(broken)[0])


if __name__ == "__main__":
    unittest.main()
