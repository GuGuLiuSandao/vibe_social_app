# Test Cases Review: social-app-develop-loop-quality

## Result

- Status: **revise / FAIL**
- Score: **74/100**
- Pass threshold: **80/100**
- Blocker present: **yes — 4 explicit blockers**
- Reviewer: Independent Test Reviewer

The design has unusually strong coverage of fail-closed reports, lifecycle failures, trusted mutation scope, CI parity, and traceability. It is not executable as written, however: one integration assertion cannot be constructed from the protobuf contract, the suite does not prove Redis participation, the manager cases contradict the stated black-box/public-contract policy, and the weak-test mutation proof does not define a deterministic weakened suite. Any one of these blockers forces FAIL regardless of score.

## Assessment

| Dimension | Weight | Score | Evidence |
|---|---:|---:|---|
| Requirement and acceptance coverage | 18 | 16 | DLQ-001–009 and all first-batch specification IDs are mapped. DLQ-004 is only partially proven because no assertion observes Redis behavior. |
| Boundary and error paths | 14 | 13 | Strong negative matrices cover reports, lifecycle exits/signals, diff trust, mutation statuses, and traceability links. Product-level HTTP errors beyond WS authentication are intentionally economical. |
| Assertion exactness | 14 | 9 | Most expected values are precise, but TC-011 asserts a nonexistent submitted nickname and TC-003 leaves time truncation/tolerance ambiguous. |
| Isolation and cleanup | 12 | 10 | Unique Compose projects, dynamic backend ports, sentinel resources, signals, and parallel runs are covered. Concurrent evidence-path allocation is required by TC-022 but not explicitly contracted. |
| Automation feasibility | 14 | 9 | Golden fixtures and fake executables are feasible. TC-005/006 conflict with the public-behavior rule, and TC-029 does not specify a reproducible sandbox transformation. |
| Mutation-test validity | 12 | 8 | Real fixed-target smokes plus strict report verification are valuable. The weak-suite meta-test cannot reliably guarantee the demanded survivor categories without defining exactly which protective tests remain. |
| Weak-test resistance | 6 | 5 | Exact response fields, side effects, report minima, scope equality, and negative fixtures resist superficial green tests; Redis remains an unobserved dependency. |
| Scope economy and redundancy | 5 | 2 | Thirty-six P0 cases include several overlapping static workflow and integration-lifecycle contracts; the gate is costly and some cases can be consolidated without losing risk coverage. |
| Traceability | 5 | 2 | Mapping is comprehensive, but TC-035 allows an “explicitly documented required target,” weakening the otherwise exact requirement that traceability run in a mandatory gate. |
| **Total** | **100** | **74** | **FAIL — blockers present** |

## Required Changes

| ID | Problem | Location | Type | Required change |
|---|---|---|---|---|
| **B1** | The registration request has only `username`, `email`, and `password`; it has no nickname field. Therefore “nickname equal submitted values” cannot be executed. Current handler behavior assigns nickname from username. | DLQ-TC-011 | Assertion/executability blocker | Replace the assertion with `register.user.nickname == submitted username`, then assert login returns the same nickname. If a caller-supplied nickname is intended, that is a product/contract change and must first be added to Requirement, Design, and protobuf—not introduced by a test. |
| **B2** | The suite starts Redis but never observes that the tested flow actually uses it. Login/register do not use Redis; WS online registration logs Redis failure and still serves ping/pong. The readiness `415` path returns before DB or Redis access, so it cannot prove either dependency is usable. | DLQ-TC-016, DLQ-TC-017; coverage of DLQ-004 | Coverage/isolation blocker | Require Compose health for PostgreSQL and Redis separately. After authenticated WS registration, bounded-poll the isolated Redis instance from inside the Compose network (or through a test-only inspection command) and assert the expected online key/value for the UID; after close, assert the offline transition or key removal. Keep the HTTP registration assertion as the database/AutoMigrate proof. A mere container-running or `415` assertion is insufficient. |
| **B3** | The execution policy says tests must use public behavior/documented report contracts and must not assert private implementation structure, but TC-005/006 explicitly call private manager methods and inspect the internal map/pointer identity. No public deterministic manager API exposes all demanded observations. | Section 1; DLQ-TC-005, DLQ-TC-006 | Internal contradiction blocker | Either (preferred) narrow the policy to permit same-package tests of the documented manager contract and name the allowed seam (`registerClient`, `unregisterClient`, `clientSnapshot`), or redesign the cases around an exported/testable manager abstraction. Do not require production behavior changes merely to satisfy an unstated public API. Also construct the minimal `Server` directly so the unit tests do not inherit global DB/Redis state through `NewServer`. |
| **B4** | The deliberately weakened suites are described semantically (“success/non-empty-only”, “smoke/no-throw”) but not as exact transformations. Other JWT/UID tests may still kill every mutant, while tool-generated mutants can vary even at a pinned version. Thus the required survivor category is not reproducible and the case can fail despite a correct mutation gate. | DLQ-TC-029 | Mutation-validity blocker | Define checked-in patch fixtures or a manifest of exact test names/files to replace or disable in each disposable copy; assert the sandbox diff equals that manifest and that all other relevant JWT/UID protective tests are accounted for. Run a strong-suite control and weak-suite variant against the same target/tool version, require both ordinary baselines green, require strong mutation pass and weak mutation failure, and accept the tool's documented weak outcome set. Do not use filler tests that exercise the mutation target. |
| **RC1** | NumericDate values are second-granularity while captured Go times include subsecond precision; “within the captured interval with test clock tolerance” does not define an executable bound. | DLQ-TC-003 | Required exactness fix | State exact comparisons after truncation, e.g. `IssuedAt/NotBefore` are within `[before.Truncate(second), after.Truncate(second)]`, expiration differs from issued-at by exactly 168 hours at second precision, and no arbitrary sleep is used. |
| **RC2** | TC-035 permits traceability verification in `quality-static` “or an explicitly documented required target,” which can leave DLQ-008 outside `make quality`/CI. | DLQ-TC-035, DLQ-TC-034 | Required traceability fix | Make `verify-traceability` an exact child of `quality-static` (or an exact named child of `quality`) and assert the workflow reaches it through the same Make DAG. Remove the open-ended alternative. |
| **RC3** | TC-022 requires parallel runs to have distinct report/log paths, while the design names shared `quality/` report locations and does not define allocation or artifact aggregation. | DLQ-TC-022 | Required isolation fix | Specify per-run directories such as `quality/integration/<project>/`, pass the path explicitly to runner and verifier, and define how a normal single run exposes/uploads its evidence. Assert no shared file is truncated or overwritten. |

## Risks

- Gremlins operates at Go package scope, so a diff in one auth file can mutate unrelated package production files. The verifier checks package boundaries but the cases do not cap runtime or mutant count; large future packages may make every-event mutation impractical.
- Requiring every non-target/docs-only event to run two real mutation smokes is stronger than the Requirement's auditable-skip allowance and materially increases CI cost. It is valid as a chosen design policy, but should be recorded as such rather than implied by DLQ-005 alone.
- Static YAML expression “resolution” in TC-031 can easily become a home-grown, inaccurate GitHub Actions evaluator. Tests should validate explicit workflow structure and wrapper inputs using event fixtures, not claim full GitHub expression semantics unless a real evaluator is used.
- TC-016 treats successful registration as schema readiness, which is sound for the exercised user tables but not proof of all AutoMigrate targets. Keep the claim scoped to the auth flow.
- Exact action majors are supply-chain constraints, not full pinning. If “pins tools” means immutable Actions, commit-SHA pinning would be needed; otherwise rename the assertion to “approved major versions.”

## Redundant or Low-Value Cases

| Cases | Finding | Economy recommendation |
|---|---|---|
| DLQ-TC-030, 031, 032, 033 | Four P0 files/cases repeatedly parse the same workflow and overlap on event eligibility, SHA rules, job presence, skip prevention, and targets. | Implement one table-driven workflow contract suite with event/diff fixtures while preserving the four Case IDs as subtests. |
| DLQ-TC-016, 019, 020 | Lifecycle ordering, readiness, cleanup, signals, and diagnostics overlap heavily. | Keep one lifecycle state-machine harness with table-driven failure and signal subtests; reserve real Docker only for dependency/isolation observations. |
| DLQ-TC-012 and 014 | Positive golden report fixtures add little independently from the corresponding negative suites, which already need a valid control. | Keep the IDs for traceability but implement each as the valid-control subtest of TC-013/015 rather than separate harnesses. |
| DLQ-TC-021 | Creating a sentinel container as well as network and volume increases Docker cost; project-scoped `down` risk is primarily network/volume scope. | A sentinel volume and network are sufficient unless cleanup code contains container-name matching. |
| DLQ-TC-035/036 | Exact-ID/file/name checks overlap with process static checks but remain justified because DLQ-008 requires machine traceability. | Retain, but run from one fixture-driven verifier suite and one mandatory Make node. |

## Improvements

| Problem | Recommendation |
|---|---|
| UID case title says “canonical,” but the assertions preserve digit text such as leading zeros. | Either add a leading-zero row and specify preservation, or define canonicalization and expect the canonical decimal string. The current word is misleading. |
| WS URL assertions do not cover a base URL that already contains a query string. | If such configuration is supported, add it; otherwise state that `VITE_WS_BASE` must be a bare WS endpoint. |
| Missing/invalid WS authentication control combines two outcomes. | Make missing-token and invalid-token named table rows and assert HTTP 401 before upgrade for each. |
| Tool-report fixtures may drift from real schemas. | Generate a version-labeled real report once, minimize it manually, and include a contract test that required fields/types match the pinned tool output. |

## Implementation Focus

- Resolve B1–B4 before implementation; they are not issues that should be improvised during Coding.
- Keep fixture verifiers small and table-driven; use real Docker and mutation tools only for the few observations that fakes cannot establish.
- Make Redis participation, per-run evidence paths, and the exact weak-suite sandbox transformation observable and machine checked.
- Preserve all Case IDs in subtest names even where overlapping cases share one harness.

## Final Verdict

**FAIL / REVISE.** The score is below 80 and four explicit blockers are present. After the concrete fixes above, the design should retain its strong fail-closed coverage while becoming executable, deterministic, isolated, and materially cheaper to maintain.

---

## Re-review 1 — 2026-08-01

### Result

- Status: **PASS**
- Score: **92/100**
- Pass threshold: **80/100**
- Open blockers: **none**
- Reviewer: Independent Test Reviewer (same reviewer)

The revised test design closes all four blockers and all three required corrections from the original review. The cases are now executable without inventing a product/API change, explicitly observe both PostgreSQL and Redis participation, define the permitted manager-test seam, make the weak-suite mutation proof reproducible, and close the timing, traceability-reachability, and parallel-evidence ambiguities. No new blocker genuinely preventing executable implementation was found.

### Prior Finding Disposition

| ID | Status | Exact evidence in revised `testcases.md` |
|---|---|---|
| **B1** | **CLOSED** | **DLQ-TC-011** now states `register.user.nickname == submitted username`, explicitly notes that the request has no nickname field, and requires login to return the same nickname. This is executable against the existing registration contract and introduces no caller-supplied nickname assumption. |
| **B2** | **CLOSED** | **DLQ-TC-016** requires independent PostgreSQL and Redis Compose health checks, scopes successful registration to PostgreSQL/auth-schema usability, and explicitly says health/HTTP `415` do not prove Redis participation. **DLQ-TC-017** then requires in-network `SISMEMBER online:users <UID>` to become `1` after authenticated WS upgrade and `0` after close, with successful inspection commands and bounded 10-second polls. |
| **B3** | **CLOSED** | **Section 1, Scope and execution policy** permits one named exception: same-package `manager_test.go` may use only `registerClient`, `unregisterClient`, and `clientSnapshot`, directly constructing a minimal `Server` without `NewServer`, globals, DB, or Redis. **DLQ-TC-005/006** follow that seam and limit observations to replacement identity/count/current entries and detached snapshots. |
| **B4** | **CLOSED** | **DLQ-TC-029** defines immutable versioned manifests and checked-in unified patches, strong-file and patched-file SHA-256 checks, exact changed-path/diff validation, accounting for every relevant protective test, identical pinned target/tool/config controls, green strong and weak ordinary baselines, a passing strong mutation control, and accepted backend/frontend weak outcome sets. It also forbids filler tests from calling the mutation target and verifies the caller worktree remains unchanged. |
| **RC1** | **CLOSED** | **DLQ-TC-003** gives exact inclusive bounds at `time.Second` precision for `IssuedAt` and `NotBefore`, requires `ExpiresAt - IssuedAt == 168*time.Hour`, and disallows extra tolerance and arbitrary sleeps. |
| **RC2** | **CLOSED** | **DLQ-TC-034** makes `verify-traceability` an exact `quality-static` child, requires its failure to fail both the target and aggregate, and requires every workflow event to reach it through `make quality-static`. **DLQ-TC-035** repeats the exact reachability chain: `verify-traceability` → `quality-static` → `quality`/workflow static job. |
| **RC3** | **CLOSED** | **DLQ-TC-019** retains diagnostics under the explicit per-run `quality/integration/<COMPOSE_PROJECT_NAME>/` directory. **DLQ-TC-022** allocates and passes a distinct `QUALITY_RUN_DIR` to every producer/consumer, forbids shared-file fallback, checks byte markers against truncation/overwrite, defines atomic `latest.json` publication for a normal single run, and requires CI to upload `quality/integration/**` without flattening. Section 5 repeats this evidence allocation contract. |

### Re-score

| Dimension | Weight | Score | Re-review evidence |
|---|---:|---:|---|
| Requirement and acceptance coverage | 18 | 18 | DLQ-001–009 and all first-batch specification IDs remain mapped; PostgreSQL and Redis now each have an observable application-level proof. |
| Boundary and error paths | 14 | 14 | Negative matrices cover authentication, report schemas, lifecycle stages/signals, untrusted diffs, mutation statuses, CI event/SHA behavior, and broken traceability links. |
| Assertion exactness | 14 | 13 | Nickname, NumericDate precision, Redis transitions, workflow reachability, evidence paths, and weak mutation outcomes are now explicit. A few implementation-level schema details will still need to match retained real tool samples, as the cases already require. |
| Isolation and cleanup | 12 | 12 | Unique Compose projects, unpublished dependencies, dynamic backend ports, diagnostic-first teardown, sentinel protection, disposable mutation copies, and per-run evidence directories are all machine checked. |
| Automation feasibility | 14 | 13 | Each P0 case names an automation location and practical seam/fixture strategy. Real Docker and mutation execution are confined to observations that fakes cannot establish, though the mutation meta-test will be relatively expensive. |
| Mutation-test validity | 12 | 11 | Strong controls, exact targets, fail-closed verifiers, immutable weak-suite transformations, hash/diff checks, and documented accepted survivor categories make the proof reproducible. Tool-level equivalent-mutant risk remains but is explicitly governed. |
| Weak-test resistance | 6 | 6 | Exact fields, identities, side effects, report minima, target equality, Redis membership, negative fixtures, and deliberate weak-suite controls resist superficial green tests. |
| Scope economy and redundancy | 5 | 2 | The design retains 36 P0 Case IDs and costly workflow/lifecycle overlap, but Section 5 provides reasonable table-driven/fake-based consolidation and confines real mutation smokes to fixed targets. This is a maintainability cost, not an executability blocker. |
| Traceability | 5 | 3 | Mandatory Make/CI reachability and comprehensive positive/negative manifest checks are now exact. The final evidence paths necessarily become fully resolvable only during implementation. |
| **Total** | **100** | **92** | **PASS — no open blocker** |

### New Blocker Check

**None.** The revised cases add some stronger implementation detail than the earlier Technical Design—for example fixed mutation smokes for no-target diffs, per-run integration evidence allocation, and mandatory traceability under `quality-static`. These are explicit, executable refinements within the confirmed quality-gate scope; they do not require a product/API contract change or make the implementation internally impossible.

### Implementation Focus

- Implement the exact manager seam and test it with a directly constructed minimal `Server`; do not widen private-structure access or introduce DB/Redis setup into those unit tests.
- Make the integration wrapper own one explicit run directory from allocation through report verification, diagnostics, cleanup, `latest.json`, and artifact upload. Prove Redis membership through the project-scoped network, not container health.
- Treat the weak-suite manifests, patches, hashes, protective-test inventory, and accepted status sets as one versioned contract. Fail before mutation on any drift.
- Keep workflow and Make DAG contract tests table-driven, but preserve every stable Case ID in test/subtest names and traceability entries.
- Capture retained, version-labeled real report samples before finalizing minimized fixtures so verifier schemas follow the pinned tools rather than assumptions.

### Final Verdict

**PASS (92/100).** B1–B4 and RC1–RC3 are all closed with exact, executable evidence, and no new blocker is open. Implementation may proceed, with primary attention on the Redis observation path, per-run evidence plumbing, deterministic weak-suite mutation harness, and mandatory traceability reachability.
