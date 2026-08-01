#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { parse } from "../../frontend/node_modules/yaml/dist/index.js";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const workflowPath = path.join(root, ".github/workflows/develop-quality.yml");
const workflow = parse(fs.readFileSync(workflowPath, "utf8"));
const requiredJobs = ["static", "backend", "frontend", "integration", "backend-mutation", "frontend-mutation"];

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

assert(workflow.on?.pull_request !== undefined, "pull_request trigger is missing");
assert(workflow.on?.push?.branches?.includes("master"), "master push trigger is missing");
assert(workflow.on?.workflow_dispatch?.inputs?.base_sha?.required === true, "required manual base_sha is missing");
assert(!JSON.stringify(workflow.on).includes('"paths"'), "path filters are forbidden");
assert(!JSON.stringify(workflow.on).includes('"paths-ignore"'), "path-ignore filters are forbidden");
assert(workflow.permissions?.contents === "read", "workflow permissions must be contents: read");
assert(workflow.concurrency?.["cancel-in-progress"] === true, "concurrency cancellation is required");
assert(Object.keys(workflow.jobs).length === requiredJobs.length, "workflow must have exactly six semantic jobs");

for (const name of requiredJobs) {
  const job = workflow.jobs[name];
  assert(job, `missing job: ${name}`);
  assert(Number.isInteger(job["timeout-minutes"]), `${name} timeout is missing`);
  assert(job["continue-on-error"] === undefined, `${name} may not continue on error`);
  assert(job.if === undefined, `${name} may not be conditionally skipped`);
  assert(job.needs === undefined, `${name} may not inherit a skipped dependency`);
  const checkout = job.steps.find((step) => step.uses === "actions/checkout@v4");
  assert(checkout?.with?.["fetch-depth"] === 0, `${name} checkout must fetch full history`);
  assert(
    checkout?.with?.ref?.includes("pull_request.head.sha") && checkout?.with?.ref?.includes("github.sha"),
    `${name} checkout ref does not follow event semantics`,
  );
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
const baseExpression = workflow.env?.MUTATION_BASE_SHA ?? "";
for (const fragment of ["pull_request.base.sha", "github.event.before", "inputs.base_sha"]) {
  assert(baseExpression.includes(fragment), `mutation base expression is missing ${fragment}`);
}
console.log("workflow contract: six always-running jobs verified");
