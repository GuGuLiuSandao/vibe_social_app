#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { parse } from "../../frontend/node_modules/yaml/dist/index.js";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const workflowPath = path.join(root, ".github/workflows/develop-quality.yml");
const workflow = parse(fs.readFileSync(workflowPath, "utf8"));
const developJobs = ["static", "backend", "frontend", "integration", "backend-mutation", "frontend-mutation"];
const allJobs = ["classify", "docs", "engineering", ...developJobs, "quality-gate"];

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

assert(workflow.on?.pull_request !== undefined, "pull_request trigger is missing");
for (const event of ["opened", "synchronize", "reopened", "labeled", "unlabeled"]) {
  assert(workflow.on.pull_request.types?.includes(event), `pull_request trigger is missing ${event}`);
}
assert(workflow.on?.push?.branches?.includes("master"), "master push trigger is missing");
assert(workflow.on?.workflow_dispatch?.inputs?.base_sha?.required === true, "required manual base_sha is missing");
assert(!JSON.stringify(workflow.on).includes('"paths"'), "path filters are forbidden");
assert(!JSON.stringify(workflow.on).includes('"paths-ignore"'), "path-ignore filters are forbidden");
assert(workflow.permissions?.contents === "read", "workflow permissions must be contents: read");
assert(workflow.concurrency?.["cancel-in-progress"] === true, "concurrency cancellation is required");
assert(JSON.stringify(Object.keys(workflow.jobs).sort()) === JSON.stringify(allJobs.sort()), "workflow job set is incomplete");

const classifier = workflow.jobs.classify;
assert(classifier.outputs?.classification?.includes("classifier.outputs.classification"), "classification output is missing");
assert(
  classifier.steps.some((step) => step.run?.includes("change_classifier.py") && step.run?.includes("--github-output")),
  "classifier must inspect the diff and publish its result",
);

for (const [name, classification] of [["docs", "docs"], ["engineering", "engineering"]]) {
  const job = workflow.jobs[name];
  assert(job.needs === "classify", `${name} must depend on classification`);
  assert(job.if?.includes(`classification == '${classification}'`), `${name} classification condition is missing`);
}

for (const name of developJobs) {
  const job = workflow.jobs[name];
  assert(job, `missing job: ${name}`);
  assert(Number.isInteger(job["timeout-minutes"]), `${name} timeout is missing`);
  assert(job["continue-on-error"] === undefined, `${name} may not continue on error`);
  assert(job.if?.includes("classification == 'develop'"), `${name} must be limited to develop changes`);
  assert(job.needs === "classify", `${name} must depend on classification`);
  const checkout = job.steps.find((step) => step.uses === "actions/checkout@v4");
  assert(checkout?.with?.["fetch-depth"] === 0, `${name} checkout must fetch full history`);
  assert(checkout?.with?.ref === undefined, `${name} must test the default PR merge ref`);
  const upload = job.steps.find((step) => step.uses === "actions/upload-artifact@v4");
  assert(upload?.if === "always()", `${name} evidence upload must use always()`);
  assert(upload?.with?.path === "quality/**", `${name} evidence path is incomplete`);
}

const expectedTargets = {
  static: "make quality-static",
  backend: "make test-backend",
  frontend: "make test-frontend",
  integration: "make test-integration",
  "backend-mutation": "make mutation-backend",
  "frontend-mutation": "make mutation-frontend",
};
for (const [name, command] of Object.entries(expectedTargets)) {
  assert(workflow.jobs[name].steps.some((step) => step.run === command), `${name} must invoke ${command}`);
}
const finalGate = workflow.jobs["quality-gate"];
assert(finalGate.if?.includes("always()"), "quality-gate must always evaluate prior results");
assert(allJobs.filter((name) => name !== "quality-gate").every((name) => finalGate.needs.includes(name)), "quality-gate needs every branch job");
assert(finalGate.steps.some((step) => step.run?.includes("verify_quality_gate.py")), "quality-gate verifier is missing");
const baseExpression = workflow.env?.MUTATION_BASE_SHA ?? "";
for (const fragment of ["pull_request.base.sha", "github.event.before", "inputs.base_sha"]) {
  assert(baseExpression.includes(fragment), `mutation base expression is missing ${fragment}`);
}
console.log("workflow contract: classified DAG and stable quality-gate verified");
