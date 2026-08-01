# Test Cases: social-app-develop-loop-quality

## 1. Scope and execution policy

- Covered: Develop Loop ordering and role independence; backend JWT and WebSocket manager behavior; frontend UID and WebSocket builders; isolated black-box PostgreSQL/Redis/HTTP/WebSocket flows; non-empty report enforcement; integration lifecycle, cleanup, and parallel isolation; Gremlins and Stryker target/report enforcement; proof that mutation detects weak tests; GitHub Actions event/SHA/full-history/job behavior; and machine-checked traceability.
- P0 means a required gate for this change. Every P0 case must be automated and pass before delivery. P0 negative-fixture cases pass only when the system under test rejects the fixture with a non-zero status and the test asserts the diagnostic category.
- P1 means follow-up hardening. P1 is deliberately excluded from the initial delivery gate unless implementation changes the corresponding behavior; it must not delay the P0 gate.
- Excluded: exhaustive historical product behavior, UI end-to-end flows, deployment/release, shared or production services, Maintenance Loop, mutation of generated protobuf, performance/load testing, and merge-queue behavior (the repository requirement covers PR, master push, and manual dispatch only).
- Test names must contain the stable Case ID and applicable specification/acceptance ID. Tests use public behavior or documented report contracts except for `backend/internal/websocket/manager_test.go`: same-package tests may exercise only the documented manager seam `registerClient`, `unregisterClient`, and `clientSnapshot`, and may directly construct the minimal `Server` value needed by that seam. Those tests must not call `NewServer`, touch package globals, or inherit DB/Redis state, and may inspect returned client identity/count/snapshots only as the manager contract requires; no other private structure is testable by policy.

## 2. Specification and acceptance mapping

| Specification / acceptance ID | Expected behavior | Cases |
|---|---|---|
| DLQ-001 | Test Design and independent Test Review precede implementation; reviewer roles remain independent | DLQ-TC-001, DLQ-TC-002 |
| DLQ-002 / AUTH-001 | JWT claims round-trip and invalid signatures/tokens are rejected; backend tests cannot empty-run | DLQ-TC-003, DLQ-TC-004, DLQ-TC-012, DLQ-TC-013 |
| DLQ-002 / WS-001 | Connection replacement, stale unregister protection, and snapshots are correct | DLQ-TC-005, DLQ-TC-006 |
| DLQ-003 / CLIENT-001 | UID parsing and whitelist boundaries are stable; frontend tests cannot empty-run | DLQ-TC-007, DLQ-TC-008, DLQ-TC-014, DLQ-TC-015 |
| DLQ-003 / CLIENT-002 | WS URLs safely encode tokens and protobuf request builders preserve IDs/types/payloads | DLQ-TC-009, DLQ-TC-010 |
| DLQ-003 | Changed group-management controls and Chat page actions retain their user-visible behavior | DLQ-TC-037, DLQ-TC-038 |
| DLQ-004 / AUTH-HTTP-001 | Real protobuf register/login proves the isolated PostgreSQL auth schema is usable | DLQ-TC-011, DLQ-TC-016 |
| DLQ-004 / WS-HTTP-001 | Authenticated real WebSocket participates in isolated Redis and returns correlated protobuf pong | DLQ-TC-017 |
| DLQ-005 | Mutation baseline, target, tool, and report evidence are trustworthy; representative mutants are killed | DLQ-TC-023 through DLQ-TC-028 |
| DLQ-006 | Weak tests, surviving/uncovered/error mutants, zero mutants, and malformed evidence fail closed | DLQ-TC-025 through DLQ-TC-029 |
| DLQ-007 | All six CI jobs run for every declared event with trustworthy SHA inputs | DLQ-TC-030 through DLQ-TC-033 |
| DLQ-008 | Specification, Case, test name/file, and evidence are machine traceable | DLQ-TC-035, DLQ-TC-036 |
| DLQ-009 | Local and CI gates share Make targets and propagate every child failure | DLQ-TC-034 |

## 3. P0 gate cases

### DLQ-TC-001: Standard Develop Loop ordering is mandatory

- Priority / level: P0; process contract/static.
- Requirement mapping: DLQ-001.
- Setup: Load the installed Develop Loop README, agent role definitions, and artifact templates as text; use a fixture change classified `standard`.
- Steps: Parse the documented standard-flow stages and role start conditions; locate any quick-flow exception.
- Exact assertions: The ordered subsequence is exactly `Requirement confirmation -> Technical Design -> Design Review PASS -> Test Design -> Test Review PASS -> Implementation -> Code Review -> Local Quality Gates -> PR/CI`; Implementation cannot start on a missing or failed Design Review or Test Review; `quick` may compress design/test documentation but cannot remove independent Code Review or applicable executable gates.
- Automation location: `scripts/quality/tests/test_develop_loop_contract.py`.
- Mutation relevance: None; protects process text from deletion/reordering mutations through static contract checks.

### DLQ-TC-002: Reviewer and author roles are independent and write-bounded

- Priority / level: P0; process contract/static.
- Requirement mapping: DLQ-001.
- Setup: Load Develop Loop role rules and Test Designer, Test Reviewer, Design Reviewer, Implementer, and Code Reviewer agent definitions.
- Steps: Resolve each author/reviewer pairing and permitted artifact paths.
- Exact assertions: Test Reviewer is not the Test Designer and may modify only `testcases-review.md`; Design Reviewer is not the design author and may modify only `design-review.md`; Code Reviewer did not author and may not modify production or test code; Test Designer creates only `testcases.md` in this stage; an explicit blocker prevents progression regardless of score; review iteration limit is three.
- Automation location: `scripts/quality/tests/test_develop_loop_contract.py`.
- Mutation relevance: None.

### DLQ-TC-003: JWT round-trip preserves identity claims

- Priority / level: P0; Go unit.
- Requirement mapping: DLQ-002, AUTH-001.
- Setup: Fixed test secret, user ID `10000001`, username `alice`; capture `before` immediately before and `after` immediately after token creation, without sleeps.
- Steps: Generate a token and parse it with the same secret.
- Exact assertions: Generation and parsing return no error; token is non-empty; `UserID == 10000001`; `Username == "alice"`; signing method is accepted as valid; at JWT NumericDate second precision, `before.Truncate(time.Second) <= IssuedAt.Time <= after.Truncate(time.Second)` and the same inclusive bound holds for `NotBefore.Time`; `ExpiresAt.Time.Sub(IssuedAt.Time) == 168*time.Hour` exactly; `ExpiresAt.Time.After(after.Truncate(time.Second))`; no tolerance beyond truncation to seconds and no arbitrary sleep is used.
- Automation location: `backend/internal/auth/jwt_test.go`, test name contains `DLQ_TC_003_AUTH_001`.
- Mutation relevance: Kills claim assignment, duration, signing, and validity-check mutants in `jwt.go`.

### DLQ-TC-004: JWT rejects wrong-secret, malformed, and expired tokens

- Priority / level: P0; Go unit, table-driven.
- Requirement mapping: DLQ-002, AUTH-001.
- Setup: One valid token; a different secret; literal `not-a-jwt`; and an HS256 token with already-expired registered claims.
- Steps: Parse each input.
- Exact assertions: Every row returns a non-nil error and nil claims; no invalid input yields a partially trusted identity. A valid control row succeeds to prove the parser is not rejecting all inputs.
- Automation location: `backend/internal/auth/jwt_test.go`, test name contains `DLQ_TC_004_AUTH_001`.
- Mutation relevance: Kills secret-use, error-path, expiry, and validity-bypass mutants.

### DLQ-TC-005: WebSocket manager replaces a connection without increasing cardinality

- Priority / level: P0; Go package unit.
- Requirement mapping: DLQ-002, WS-001.
- Setup: In package `websocket`, directly construct a minimal `Server` with only the client map/lock state required by the documented seam and no DB/Redis/global state; clients A and B share UID `42`; client C uses UID `43`.
- Steps: Call only `registerClient` to register A, C, then B, and `clientSnapshot` to observe current entries.
- Exact assertions: First registration returns no replaced client and count `1`; C produces count `2`; B returns A as the replaced client and count remains `2`; current entry for UID 42 is pointer-identical to B; UID 43 remains C.
- Automation location: `backend/internal/websocket/manager_test.go`, test name contains `DLQ_TC_005_WS_001`.
- Mutation relevance: Kills map assignment, replacement, and count mutants.

### DLQ-TC-006: Stale unregister cannot remove replacement; snapshot is detached

- Priority / level: P0; Go package unit.
- Requirement mapping: DLQ-002, WS-001.
- Setup: In package `websocket`, directly construct the same minimal `Server` (never `NewServer`), then use the documented seam to register A followed by replacement B for UID `42`, plus C for UID `43`.
- Steps: Call `unregisterClient` for stale A; obtain `clientSnapshot` and mutate only the returned slice; call `unregisterClient` for current B.
- Exact assertions: Stale unregister returns `false` and count `2`; B remains current; snapshot contains exactly B and C regardless of order; modifying the slice does not change the manager map; unregister B returns `true`, count `1`, and C remains.
- Automation location: `backend/internal/websocket/manager_test.go`, test name contains `DLQ_TC_006_WS_001`.
- Mutation relevance: Kills identity comparison, deletion, lock/snapshot, and returned-count mutants.

### DLQ-TC-007: UID parsing accepts positive digit input and preserves its text

- Priority / level: P0; Vitest module unit.
- Requirement mapping: DLQ-003, CLIENT-001.
- Setup: Table of `10000000`, `" 10000000 "`, `"00010000000"`, `1n`, a value greater than JS safe integer as digit text, `0`, `-1`, `"1.5"`, `"1e3"`, `"abc"`, empty string, `null`, and `undefined`.
- Steps: Call `parseUid` for every row.
- Exact assertions: Accepted inputs return a digit string without numeric precision loss (`1n -> "1"`); trimming removes only surrounding whitespace, and digit-string identity is preserved rather than numerically canonicalized, so `"00010000000" -> "00010000000"`; zero, negative, decimal, exponent, alphabetic, empty, null, and undefined inputs return exactly `null`; no row throws.
- Automation location: `frontend/src/lib/uid.test.js`, test name contains `DLQ-TC-007 CLIENT-001`.
- Mutation relevance: Kills regex, trim, positivity, conversion, and return-value mutants.

### DLQ-TC-008: UID whitelist includes both endpoints and excludes adjacent/invalid values

- Priority / level: P0; Vitest module unit.
- Requirement mapping: DLQ-003, CLIENT-001.
- Setup: Values `9999999`, `10000000`, `20000000`, `20000001`, `10000000.5`, `"10000000"`, nonnumeric text, and a Symbol.
- Steps: Call `isWhitelistUid`.
- Exact assertions: Exactly `10000000`, `20000000`, and digit string `"10000000"` return `true`; adjacent, fractional, and nonnumeric values return `false`; Symbol returns `false` rather than throwing.
- Automation location: `frontend/src/lib/uid.test.js`, test name contains `DLQ-TC-008 CLIENT-001`.
- Mutation relevance: Kills inclusive-boundary, integer-check, conversion, and exception-handler mutants; primary Stryker smoke protection.

### DLQ-TC-009: WebSocket URL always carries UID and safely encodes a present token

- Priority / level: P0; Vitest module unit.
- Requirement mapping: DLQ-003, CLIENT-002.
- Setup: Controlled `VITE_WS_BASE` satisfying the configuration contract: an absolute bare `ws://` or `wss://` origin/base path with no query string or fragment; UID `10000001`; token `a+b/c?d=e&f` and empty token. Bases containing `?` or `#` are unsupported configuration and are not silently merged.
- Steps: Build URLs and parse their query parameters.
- Exact assertions: For every supported bare base, origin/base path and appended `/ws` path are unchanged; `uid` occurs once and decodes to `10000001`; present token occurs once and decodes exactly to `a+b/c?d=e&f`; reserved token characters do not create extra query parameters; empty token omits the `token` parameter while retaining UID. A configuration-contract assertion documents/rejects a base containing an existing query or fragment rather than defining ambiguous merge behavior.
- Automation location: `frontend/src/lib/ws.test.js`, test name contains `DLQ-TC-009 CLIENT-002`.
- Mutation relevance: Kills token-condition, encoding, query separator, and UID omission mutants.

### DLQ-TC-010: Protobuf ping builder preserves correlation and wire shape

- Priority / level: P0; Vitest module/contract unit.
- Requirement mapping: DLQ-003, CLIENT-002, WS-HTTP-001.
- Setup: Fixed request ID `9007199254740993n`; capture times around the call.
- Steps: Call `buildAccountPing`, serialize to protobuf bytes, deserialize as `WsMessage`.
- Exact assertions: Deserialized `requestId` equals the fixed bigint exactly; type is `WS_TYPE_PING`; payload case is `account`, nested case is `ping`; timestamp is within the captured interval and positive; serialization yields a non-empty byte array.
- Automation location: `frontend/src/lib/ws.test.js`, test name contains `DLQ-TC-010 CLIENT-002`.
- Mutation relevance: Kills request-ID, message-type, timestamp, payload-case, and empty-payload mutants.

### DLQ-TC-011: Real protobuf register then login produces the same authenticated identity

- Priority / level: P0; black-box integration.
- Requirement mapping: DLQ-004, AUTH-HTTP-001.
- Setup: Fresh unique Compose project, dedicated PostgreSQL database/user/password, Redis, JWT secret, and dynamic backend endpoint; generate unique username/email and password of at least six characters.
- Steps: POST protobuf `RegisterRequest`; decode `RegisterResponse`; POST protobuf `LoginRequest` with the same credentials; decode `LoginResponse`; parse both JWTs using the test secret.
- Exact assertions: Both HTTP statuses are `200`; both content types contain `application/x-protobuf`; both protobuf error codes are `ERROR_CODE_OK` and messages are `ok`; both tokens are non-empty; registered user UID is positive, username/email equal the submitted username/email, and `register.user.nickname == submitted username` (the request has no nickname field); login returns the same UID, username, email, and nickname as registration; both token claims contain that UID and username; no pre-seeded/shared account is used. The successful write/read through the fresh database is the auth user-table/AutoMigrate readiness proof, not proof of unrelated migration targets.
- Automation location: `backend/tests/integration/auth_flow_test.go`, test name contains `DLQ_TC_011_AUTH_HTTP_001`.
- Mutation relevance: Kills observable auth handler/JWT mutations and supplies the real-dependency baseline.

### DLQ-TC-012: Backend report accepts a valid non-empty run

- Priority / level: P0; verifier contract unit with golden fixture.
- Requirement mapping: DLQ-002, DLQ-007, DLQ-009.
- Setup: Checked-in minimized golden JSONL derived from an actual pinned Go `1.25.3` `go test -json` report and labeled `go-1.25.3`, containing valid objects, at least one `Action=run` test in each exact package `social_app/internal/auth` and `social_app/internal/websocket`, at least six total run events, and no underlying failure.
- Steps: Invoke the Go report verifier on the fixture.
- Exact assertions: Exit status is `0`; diagnostic states total tests `>= 6` and both required packages are present.
- Automation location: `scripts/quality/tests/test_verify_go_test_report.py`; fixture `scripts/quality/tests/fixtures/go/valid.jsonl`.
- Mutation relevance: Kills verifier comparison and package-presence mutants.

### DLQ-TC-013: Backend report and wrapper fail closed

- Priority / level: P0; verifier/wrapper negative fixture suite.
- Requirement mapping: DLQ-002, DLQ-007, DLQ-009.
- Setup: Separate golden fixtures for missing path, empty file, malformed JSON line, non-object JSON, zero run events, five tests, missing auth package, missing WebSocket package, and filtered-to-zero output; fake `go test` exit codes `1` and `37`.
- Steps: Run verifier once per fixture; run wrapper with fake test process.
- Exact assertions: Every invalid fixture exits non-zero and names its category (`missing`, `empty`, `invalid JSON/schema`, `zero/insufficient tests`, or missing exact package); fake underlying exits are returned exactly as `1` and `37`; verifier is not allowed to turn a failed underlying run green.
- Automation location: `scripts/quality/tests/test_verify_go_test_report.py`, `scripts/quality/tests/test_run_backend_tests.py`; fixtures under `scripts/quality/tests/fixtures/go/negative/`.
- Mutation relevance: Prevents weak/empty Go suites from creating false mutation baselines.

### DLQ-TC-014: Frontend report accepts a valid non-empty run

- Priority / level: P0; verifier contract unit with golden fixture.
- Requirement mapping: DLQ-003, DLQ-007, DLQ-009.
- Setup: Checked-in minimized golden JSON object derived from a real pinned Vitest `4.1.10` report and labeled `vitest-4.1.10`, with `numTotalTestSuites >= 2`, `numTotalTests >= 6`, `numFailedTests = 0`, and result entries for exact UID and WS test files.
- Steps: Invoke the frontend report verifier.
- Exact assertions: Exit status is `0`; output confirms the suite/test minima, zero failures, and both required files.
- Automation location: `scripts/quality/tests/verify-frontend-test-report.test.mjs`; fixture `scripts/quality/tests/fixtures/vitest/valid.json`.
- Mutation relevance: Kills verifier threshold/file-discovery mutants.

### DLQ-TC-015: Frontend report and wrapper fail closed

- Priority / level: P0; verifier/wrapper negative fixture suite.
- Requirement mapping: DLQ-003, DLQ-007, DLQ-009.
- Setup: Separate fixtures for missing path, empty, malformed JSON, array/non-object root, wrong field types, zero suites/tests, one suite, five tests, one failed test, missing UID file, missing WS file, and filtered-to-zero; fake Vitest exits `1` and `29`.
- Steps: Run verifier and wrapper for each condition.
- Exact assertions: Each invalid fixture exits non-zero with its diagnostic category; underlying exit is propagated exactly; `--passWithNoTests=false` is present in the invoked test command; no report text can override a non-zero runner status.
- Automation location: `scripts/quality/tests/verify-frontend-test-report.test.mjs`, `scripts/quality/tests/test_run_frontend_tests.py`; fixtures under `scripts/quality/tests/fixtures/vitest/negative/`.
- Mutation relevance: Prevents weak/empty Vitest suites from creating false Stryker baselines.

### DLQ-TC-016: Integration environment is real, isolated, migrated, and ready

- Priority / level: P0; integration lifecycle/black-box.
- Requirement mapping: DLQ-004, DLQ-009.
- Setup: No pre-existing test resources; invoke the integration wrapper from a clean process.
- Steps: Observe build/up and inspect resolved ports/Compose configuration; bounded-wait for PostgreSQL and Redis health independently; then bounded-wait for backend HTTP readiness, run the registration flow, and inspect resources before teardown.
- Exact assertions: Project name matches `social-app-it-<pid>-<random>` and is unique; PostgreSQL and Redis have no published host ports; their Compose health checks each reach `healthy` within 90 seconds or the gate fails, independently naming the dependency; backend binds a dynamic `127.0.0.1` host port and that exact URL reaches tests via `INTEGRATION_BASE_URL`; credentials/database/JWT secret are test-only; backend readiness accepts only HTTP `415` from `/api/v1/auth/login` within 90 seconds with each request capped at 3 seconds, but that `415` is only router/process readiness; successful DLQ-TC-011 registration against the fresh database proves PostgreSQL usability and auth user-table/AutoMigrate readiness; network and volumes are scoped to this project. Redis participation is proved by DLQ-TC-017, not by health or `415` alone.
- Automation location: `scripts/quality/tests/test_integration_lifecycle.py` plus `backend/tests/integration/auth_flow_test.go`.
- Mutation relevance: Ensures mutation/integration evidence is not produced against mocks or shared state.

### DLQ-TC-017: Authenticated real WebSocket protobuf ping/pong is correlated

- Priority / level: P0; black-box HTTP + WebSocket integration.
- Requirement mapping: DLQ-004, WS-HTTP-001.
- Setup: Use token and UID created through DLQ-TC-011 in the same isolated environment; fixed request ID; 3-second dial/read/write deadlines and a 10-second total bounded poll deadline. The wrapper exposes a test-only inspection command that runs the pinned Redis CLI/client inside the project-scoped Compose network, addressing the Redis service name; Redis remains unpublished to the host.
- Steps: Run named authentication rows `valid`, `missing-token`, and `invalid-token`. For `valid`, dial the real WS endpoint with UID and encoded token; after upgrade bounded-poll Redis in-network with `SISMEMBER online:users <UID>`, send a binary protobuf ping, decode pong, close the socket, then bounded-poll the same membership for removal. For each negative row, attempt the handshake without a token or with a corrupt token.
- Exact assertions: Valid upgrade succeeds; within 10 seconds `SISMEMBER online:users <UID>` returns integer `1` from the isolated Redis service, proving application participation rather than container health; pong is a binary protobuf frame with type `WS_TYPE_PONG`, the identical request ID, a positive timestamp, and arrives before the read deadline; after a clean close, within 10 seconds the same command returns integer `0`, and every inspection command itself must succeed. The `missing-token` and `invalid-token` rows each receive HTTP `401` before upgrade and receive no WS frame. Any Redis inspection timeout/error, wrong membership result, or missing removal fails the case.
- Automation location: `backend/tests/integration/websocket_flow_test.go`, test name contains `DLQ_TC_017_WS_HTTP_001`.
- Mutation relevance: Kills WS auth, dispatch type, correlation ID, and pong response mutations.

### DLQ-TC-018: Integration report is non-empty and contains both required flows

- Priority / level: P0; report verifier with golden fixtures.
- Requirement mapping: DLQ-004, DLQ-007, DLQ-009.
- Setup: Valid Go JSONL report with at least two run events containing exact AUTH-HTTP and WS-HTTP suite IDs; negative fixtures missing/empty/malformed/zero/one flow.
- Steps: Run integration report verifier on each fixture.
- Exact assertions: Valid fixture exits `0`; every negative fixture exits non-zero; verifier requires total `>= 2` and both suite IDs independently, so duplicate events for one suite cannot satisfy the gate.
- Automation location: `scripts/quality/tests/test_verify_integration_report.py`; fixtures under `scripts/quality/tests/fixtures/integration/`.
- Mutation relevance: Prevents mutation/CI success when the real cross-layer suite did not execute.

### DLQ-TC-019: Every integration failure path captures diagnostics before cleanup

- Priority / level: P0; lifecycle wrapper component test with fake Compose/curl/go executables.
- Requirement mapping: DLQ-004, DLQ-007, DLQ-009.
- Setup: Injectable `DOCKER_COMPOSE_BIN` fake recording calls and controllable failures at build, up, port discovery, readiness timeout, test command, report verification, log capture, and cleanup.
- Steps: Run one scenario per failure point.
- Exact assertions: Each scenario exits non-zero; after allocation, every exit attempts `ps`, then `logs`, then `down --volumes --remove-orphans` in that order; no test runs after an earlier failed state; the first primary failure code is preserved when cleanup also fails; if the primary flow succeeds but cleanup fails, final status is non-zero; diagnostic files remain under the run's explicit `quality/integration/<COMPOSE_PROJECT_NAME>/` directory.
- Automation location: `scripts/quality/tests/test_integration_lifecycle.py`.
- Mutation relevance: Kills trap, ordering, error-propagation, and cleanup-flag mutations.

### DLQ-TC-020: INT and TERM perform the same bounded cleanup

- Priority / level: P0; lifecycle wrapper signal test.
- Requirement mapping: DLQ-004, DLQ-009.
- Setup: Fake lifecycle paused after `up`; record Compose calls.
- Steps: Send `INT` in one run and `TERM` in another; wait with a bounded timeout.
- Exact assertions: Both runs terminate non-zero without hanging; each captures `ps` and `logs` before exactly one `down --volumes --remove-orphans`; resources allocated by the run are targeted using its exact project name.
- Automation location: `scripts/quality/tests/test_integration_lifecycle.py`.
- Mutation relevance: Kills missing signal trap and wrong-project cleanup mutants.

### DLQ-TC-021: Pre-existing similarly named resources are untouched

- Priority / level: P0; Compose isolation integration.
- Requirement mapping: DLQ-004, DLQ-009.
- Setup: Create a sentinel network/volume/container outside the generated project with similar human-readable names and a marker value.
- Steps: Run a complete integration gate and teardown.
- Exact assertions: The run uses only its generated project-scoped resources; after teardown no resources for that generated project remain; sentinel resources still exist and marker value is unchanged.
- Automation location: `scripts/quality/tests/test_integration_isolation.sh`.
- Mutation relevance: None; catches overly broad cleanup.

### DLQ-TC-022: Two integration runs are parallel-isolated

- Priority / level: P0; parallel integration smoke.
- Requirement mapping: DLQ-004, DLQ-009.
- Setup: Start two wrappers concurrently with distinct account fixtures. Each wrapper allocates `QUALITY_RUN_DIR=quality/integration/<COMPOSE_PROJECT_NAME>/` immediately after its unique project name and passes that exact absolute/normalized directory explicitly to the Go runner, report verifier, `ps`, and log capture; no component may fall back to shared `quality/integration-report.jsonl`, `quality/compose-ps.txt`, or `quality/compose.log` files.
- Steps: Wait for both auth and WS flows and both teardowns.
- Exact assertions: Project names, backend host ports, networks, volumes, databases, and `QUALITY_RUN_DIR` paths are distinct; each directory contains its own integration JSONL, verifier summary, Compose `ps`, Compose logs, and metadata naming project/endpoint; both runs exit `0`; neither sees the other's account; byte markers written to one run's files remain intact after the other completes, proving no shared file is truncated or overwritten; all resources from both projects are removed afterward. A normal single run atomically writes/updates `quality/integration/latest.json` as a small manifest pointing to its completed run directory; CI uploads `quality/integration/**`, so aggregation preserves every per-project directory rather than flattening filenames.
- Automation location: `scripts/quality/tests/test_integration_parallel.sh`.
- Mutation relevance: None; validates deterministic isolation needed for CI concurrency.

### DLQ-TC-023: Mutation target discovery accepts only a trusted diff

- Priority / level: P0; wrapper contract unit using temporary Git repositories.
- Requirement mapping: DLQ-005, DLQ-006, DLQ-007.
- Setup: Commit graphs covering valid ancestor, absent base, non-ancestor, malformed SHA, rename, deletion-only, no diff, inapplicable docs/generated/tests, unknown status, submodule, and path traversal/out-of-repository path.
- Steps: Run both target-discovery wrappers with explicit `MUTATION_BASE_SHA`.
- Exact assertions: Missing/malformed/nonexistent/non-ancestor base, diff parse failure, unknown status, submodule, and path escape exit non-zero before mutation; diff command semantics are `--name-status --find-renames base...HEAD`; eligible Go production files map to their packages while excluding tests/generated proto; eligible frontend JS/JSX becomes an exact file list while excluding tests/proto/setup/assets; no diff, not-applicable, and deleted-only are recorded distinctly and select the fixed smoke targets rather than skip.
- Automation location: `scripts/quality/tests/test_mutation_target_discovery.py`.
- Mutation relevance: Protects mutation scope itself from silent narrowing.

### DLQ-TC-024: Mutation always runs a successful ordinary-test baseline first

- Priority / level: P0; wrapper component test.
- Requirement mapping: DLQ-005, DLQ-006.
- Setup: Fake Make/Gremlins/Stryker executables with call logs; baseline success and baseline failure variants.
- Steps: Invoke backend and frontend mutation wrappers.
- Exact assertions: Backend calls `make test-backend` before Gremlins; frontend calls `make test-frontend` before Stryker; baseline failure returns its non-zero status and mutation tool is never called; baseline success proceeds. Gremlins version must equal `v0.6.0`; lockfile/config resolves Stryker core and Vitest runner `9.6.1` and Vitest `4.1.10`.
- Automation location: `scripts/quality/tests/test_mutation_wrappers.py`.
- Mutation relevance: Directly proves weak/red baseline cannot yield a mutation pass.

### DLQ-TC-025: Gremlins verifier accepts only complete all-killed evidence

- Priority / level: P0; verifier contract with golden fixtures.
- Requirement mapping: DLQ-005, DLQ-006.
- Setup: A checked-in minimized golden JSON derived from a real Gremlins `v0.6.0` report, labeled `gremlins-v0.6.0`, with `mutants_total > 0`, non-negative integer counters satisfying the total invariant, zero lived, zero not-covered, allowed statuses only `KILLED`/`NOT_VIABLE`, and files inside planned packages. Negative fixtures derived from that schema cover missing, empty, malformed, wrong types, negative count, broken invariant, zero total, lived, not-covered, each tool error representation, unknown status, and file outside plan; a contract test checks the required field names/types against a retained version-labeled real-schema sample.
- Steps: Run verifier on every fixture with an explicit planned package set.
- Exact assertions: Only the valid fixture exits `0`; every negative fixture exits non-zero and names the violated field/status/scope; `LIVED`, `NOT_COVERED`, unknown state, zero total, malformed schema, or target mismatch can never be reported as skipped or passed.
- Automation location: `scripts/quality/tests/test_verify_gremlins_report.py`; fixtures under `scripts/quality/tests/fixtures/gremlins/`.
- Mutation relevance: Golden negative fixtures kill weakened comparison/status/parser mutants in the verifier.

### DLQ-TC-026: Stryker verifier accepts only exact-scope all-killed evidence

- Priority / level: P0; verifier contract with golden fixtures.
- Requirement mapping: DLQ-005, DLQ-006.
- Setup: A checked-in minimized mutation-testing report derived from real Stryker `9.6.1` output, labeled `stryker-9.6.1`, with `> 0` mutants, every status `Killed`, and exact planned files. Negative fixtures derived from that schema cover missing, empty, malformed, wrong types, zero mutants, and one fixture each for `Survived`, `NoCoverage`, `Timeout`, `RuntimeError`, `CompileError`, `Ignored`, `Pending`, unknown status, extra file, and missing planned file; a contract test checks required field names/types against a retained version-labeled real-schema sample.
- Steps: Run verifier on every fixture with an explicit mutate list.
- Exact assertions: Only the valid fixture exits `0`; every other fixture exits non-zero with category; actual files equal planned files as sets; no non-`Killed` status is tolerated; Stryker config declares Vitest runner, JSON and clear-text reporters, and break threshold `100`.
- Automation location: `scripts/quality/tests/verify-stryker-report.test.mjs`; fixtures under `scripts/quality/tests/fixtures/stryker/`.
- Mutation relevance: Golden negative fixtures kill weakened status/scope/parser mutants in the verifier.

### DLQ-TC-027: Backend mutation smoke kills representative JWT mutants

- Priority / level: P0; real mutation smoke.
- Requirement mapping: DLQ-005, DLQ-006, AUTH-001.
- Setup: Clean baseline; fixed target `backend/internal/auth/jwt.go`; Gremlins `v0.6.0`.
- Steps: Run backend mutation smoke and verifier.
- Exact assertions: Baseline passes; Gremlins exits `0`; report contains at least one mutant; `mutants_lived == 0`, `mutants_not_covered == 0`; every file lies in the planned auth package and every status is `KILLED` or `NOT_VIABLE`; JSON and human summary remain in the wrapper's per-run evidence directory and are included by the aggregate `quality/**` artifact upload.
- Automation location: `scripts/quality/run-backend-mutation.sh`, exercised by `make mutation-backend` and CI.
- Mutation relevance: This is the direct proof that JWT tests kill production mutations.

### DLQ-TC-028: Frontend mutation smoke kills representative UID mutants

- Priority / level: P0; real mutation smoke.
- Requirement mapping: DLQ-005, DLQ-006, CLIENT-001.
- Setup: Clean baseline; exact target `frontend/src/lib/uid.js`; locked Stryker toolchain.
- Steps: Run frontend mutation smoke and verifier.
- Exact assertions: Baseline passes; Stryker exits `0`; report has at least one mutant; every mutant is `Killed`; actual target set is exactly `frontend/src/lib/uid.js`; JSON and clear-text summary remain in the wrapper's per-run evidence directory and are included by the aggregate `quality/**` artifact upload.
- Automation location: `scripts/quality/run-frontend-mutation.sh`, exercised by `make mutation-frontend` and CI.
- Mutation relevance: Direct proof that UID tests kill production mutations.

### DLQ-TC-029: Deliberately weak tests demonstrably fail both mutation gates

- Priority / level: P0; mutation meta-test in disposable worktrees/sandboxes.
- Requirement mapping: DLQ-006.
- Setup: Checked-in, immutable manifests `scripts/quality/tests/fixtures/weak-suites/backend-jwt-v1.json` and `frontend-uid-v1.json` enumerate target tool/version, target production file, SHA-256 of the strong test file, every relevant protective test full name, the exact names disabled/replaced, the checked-in unified patch path, expected patched-file SHA-256, and allowed weak mutation statuses. Backend patch `backend-jwt-v1.patch` replaces all manifest-listed JWT protective tests in `backend/internal/auth/jwt_test.go` with a named green test that checks token creation returns no error/non-empty token but does not parse it or assert claims. Frontend patch `frontend-uid-v1.patch` replaces all manifest-listed `parseUid`/`isWhitelistUid` protective tests in `frontend/src/lib/uid.test.js` with a named green import/load assertion that never calls `parseUid` or `isWhitelistUid`. No filler test may import/call the mutation target; report minima are satisfied only by existing unrelated tests explicitly listed in each manifest. Disposable copies never modify the caller worktree.
- Steps: In one clean disposable copy per target, first run the unpatched strong suite baseline and real pinned mutation command and retain its report; in a second copy verify the strong-file SHA, apply the single checked-in patch, assert `git diff --binary`/changed paths and patched SHA equal the manifest exactly, assert every relevant protective test is either retained or explicitly disabled/replaced with no unaccounted target-calling test, run the ordinary weak baseline, then run the same mutation target/tool/config and verifier; discard both copies.
- Exact assertions: Strong and weak ordinary baselines both exit `0`; strong mutation control exits `0` with all acceptable production mutants killed (or Gremlins `NOT_VIABLE`); weak mutation wrapper/verifier exits non-zero. Backend weak evidence must contain at least one documented weak outcome from `{LIVED, NOT_COVERED}` and frontend weak evidence at least one from `{Survived, NoCoverage}`; either member is acceptable because pinned tools may choose different viable mutants, while tool errors/timeouts/compile errors are not acceptable proof of weakness. Strong and weak runs use identical pinned version, target, and mutation configuration; weak patches contain no production change; manifest coverage/path/hash mismatch fails before mutation; caller worktree hash/status and production files are unchanged afterward.
- Automation location: `scripts/quality/tests/test_mutation_weak_tests.sh`.
- Mutation relevance: Required end-to-end proof that mutation catches assertion weakness missed by ordinary test success.

### DLQ-TC-030: Workflow triggers every required job without path filters

- Priority / level: P0; workflow YAML static contract.
- Requirement mapping: DLQ-007, DLQ-009.
- Setup: Parse `.github/workflows/develop-quality.yml` as YAML, not regex-only text.
- Steps: Inspect triggers and job graph.
- Exact assertions: Triggers include `pull_request`, push restricted to `master`, and `workflow_dispatch` with required `base_sha`; no trigger contains `paths` or `paths-ignore`; exactly the required semantic jobs exist: static, backend, frontend, integration, backend mutation, frontend mutation; no required job has path-based or change-based `if`, `continue-on-error`, or a `needs` chain that can silently skip it; every job has a timeout.
- Automation location: `scripts/quality/tests/test_workflow_contract.py`.
- Mutation relevance: None; static protection against CI bypass.

### DLQ-TC-031: Workflow uses full history and exact event SHA semantics

- Priority / level: P0; workflow YAML/static expression contract.
- Requirement mapping: DLQ-007.
- Setup: Parsed workflow plus synthetic event fixtures for PR, master push, and manual dispatch.
- Steps: Resolve checkout ref and `MUTATION_BASE_SHA` expressions for each event.
- Exact assertions: Every checkout is `actions/checkout@v4` with `fetch-depth: 0`; PR checks out `pull_request.head.sha` and uses `pull_request.base.sha`; push checks out `github.sha` and uses `github.event.before`; manual uses current ref SHA and required input `base_sha`; zero/missing/non-ancestor push-before and invalid manual/PR base lead to mutation failure, never smoke/skip fallback; fork PR requires only read permission.
- Automation location: `scripts/quality/tests/test_workflow_contract.py` and temporary-Git cases in `test_mutation_target_discovery.py`.
- Mutation relevance: Ensures trustworthy diff selection.

### DLQ-TC-032: Workflow invokes repository gates, pins tools, and uploads evidence always

- Priority / level: P0; workflow YAML static contract.
- Requirement mapping: DLQ-007, DLQ-008, DLQ-009.
- Setup: Parsed workflow, Makefile target list, and lockfile.
- Steps: Inspect each job's run steps, permissions, tool versions, concurrency, and artifact conditions.
- Exact assertions: Jobs invoke the matching Make target rather than duplicate test/mutation logic; permissions are only `contents: read`; Go is `1.25.3`, Node is `20`, Gremlins is `v0.6.0`, npm uses the lockfile, Actions use declared major versions; per-ref concurrency cancels older runs; artifact upload is `always()` and includes `quality/**`, thereby preserving unit JSON, every `quality/integration/<project>/` directory and `latest.json`, mutation JSON, and summaries without filename flattening; cancellation/timeout has no success fallback.
- Automation location: `scripts/quality/tests/test_workflow_contract.py`.
- Mutation relevance: None.

### DLQ-TC-033: All declared events exercise all six jobs, including no-target smoke

- Priority / level: P0; CI contract simulation/static matrix.
- Requirement mapping: DLQ-007.
- Setup: Event matrix fixtures and diffs for backend, frontend, cross-layer, docs-only, deleted-only, and empty diff.
- Steps: Evaluate job eligibility and mutation target result for each event/diff pair.
- Exact assertions: All six jobs are eligible for every event and diff class; docs-only, deleted-only, and empty diffs run fixed backend JWT and frontend UID mutation smoke; no required job resolves to skipped; unknown/untrusted diff resolves to failure.
- Automation location: `scripts/quality/tests/test_workflow_matrix.py`.
- Mutation relevance: Protects required mutation checks from false skips.

### DLQ-TC-034: Local quality DAG is identical to CI and propagates child failures

- Priority / level: P0; Make contract/component test.
- Requirement mapping: DLQ-007, DLQ-009.
- Setup: Inspect Makefile and project profile; fake child commands with selectable exit status.
- Steps: Expand each named target; run `quality` repeatedly with one failing child.
- Exact assertions: `quality-static` has the exact children `git diff --check`, Go build, Go vet, frontend build, default Compose config, integration Compose config, and `verify-traceability`; `verify-traceability` invokes `scripts/quality/verify-traceability.py`; `quality` depends exactly on `quality-static`, backend, frontend, integration, backend mutation, and frontend mutation; named test/mutation targets call their wrappers; every injected child failure, including traceability failure, makes its target and aggregate non-zero; project profile lists the same matrix; every workflow event reaches traceability through the static job's same `make quality-static` DAG.
- Automation location: `scripts/quality/tests/test_make_quality_contract.py`.
- Mutation relevance: None; ensures mutation is a real aggregate dependency.

### DLQ-TC-035: Valid traceability manifest resolves end to end

- Priority / level: P0; traceability verifier contract/integration.
- Requirement mapping: DLQ-008 and all first-batch specification IDs.
- Setup: `docs/specs/traceability.json`, specification documents, this file, and implemented test files.
- Steps: Run traceability verifier.
- Exact assertions: Exit `0`; every manifest entry has unique specification ID/Case ID/test identity as required by schema; referenced spec document, exact Case ID, and test file exist; file contains the exact declared ID and test full name; every first-batch spec (`AUTH-001`, `WS-001`, `CLIENT-001`, `CLIENT-002`, `AUTH-HTTP-001`, `WS-HTTP-001`) has at least one automated test; quality evidence paths are resolvable for Code Review.
- Automation location: `scripts/quality/tests/test_verify_traceability.py` plus `scripts/quality/verify-traceability.py`; `verify-traceability` is an exact child of `make quality-static`, which is an exact child of `make quality` and is invoked by the workflow static job.
- Mutation relevance: Links production mutation evidence back to its behavioral contract.

### DLQ-TC-036: Traceability verifier rejects every broken link

- Priority / level: P0; verifier negative fixture suite.
- Requirement mapping: DLQ-008.
- Setup: One valid golden manifest and independent negative fixtures for malformed/non-object schema, duplicate ID, unknown spec, missing spec document, unknown/missing Case, missing test file, file lacking exact ID, mismatched full test name, and an uncovered first-batch spec.
- Steps: Run verifier once per fixture in a controlled fixture root.
- Exact assertions: Valid fixture exits `0`; every negative fixture exits non-zero and identifies the broken link/category; substring-only ID matches do not satisfy exact ID checks; one valid mapping cannot mask another invalid mapping.
- Automation location: `scripts/quality/tests/test_verify_traceability.py`; fixtures under `scripts/quality/tests/fixtures/traceability/`.
- Mutation relevance: Kills weakened uniqueness/existence/exact-match verifier mutants.

### DLQ-TC-037: Changed group-management controls retain their behavior

- Priority / level: P0; React component behavior and mutation protection.
- Requirement mapping: DLQ-003, DLQ-005, DLQ-006.
- Setup: Render create, profile, join-request, and member-management components with neutral `Chat*` controls and role-specific fixtures.
- Steps: Edit form fields; invoke create/profile/invitation callbacks; exercise owner, admin, member, self, and non-manager branches.
- Exact assertions: Edited values and IDs reach the correct callbacks; each role sees exactly its permitted actions; every selected diff-line mutant is killed.
- Automation location: `frontend/src/components/chat/GroupManagementComponents.test.jsx`.
- Mutation relevance: Kills changed-line callback, equality, logical, and conditional mutants in the group-management components.

### DLQ-TC-038: Changed Chat page controls remain correctly wired

- Priority / level: P0; React page behavior and mutation protection.
- Requirement mapping: DLQ-003, DLQ-005, DLQ-006.
- Setup: Render `Chat` with an authentication-error fixture and an authenticated fake WebSocket fixture; inject a protobuf group invitation.
- Steps: Return to login; open/use/close follow and profile dialogs; accept and reject the invitation.
- Exact assertions: Login navigation uses replacement; follow target and invitation decision payloads preserve exact values; modal cancel controls close their own dialogs; every selected diff-line mutant is killed.
- Automation location: `frontend/src/pages/ChatControls.test.jsx`.
- Mutation relevance: Kills changed-line callback, boolean, object, string, and arrow-function mutants in `Chat.jsx`.

## 4. P1 follow-up cases

### DLQ-TC-101: JWT algorithm confusion is rejected explicitly

- Priority / level: P1; Go security unit.
- Requirement mapping: AUTH-001 hardening.
- Setup: A correctly shaped token signed with a non-HS256 method or `none`, using test-only keys.
- Steps: Parse it with normal configuration.
- Exact assertions: Parse returns error and nil claims; valid HS256 control succeeds.
- Automation location: `backend/internal/auth/jwt_test.go`.
- Mutation relevance: Kills signing-method validation mutants if explicit algorithm enforcement is added.

### DLQ-TC-102: WebSocket manager concurrent access is race-free

- Priority / level: P1; Go concurrency/race test.
- Requirement mapping: WS-001 hardening.
- Setup: Many unique/repeated UIDs and bounded goroutines.
- Steps: Concurrently register, snapshot, send, and unregister; execute with `go test -race`.
- Exact assertions: No race detector finding, panic, deadlock, or map corruption; final entries correspond only to the last non-unregistered client per UID.
- Automation location: `backend/internal/websocket/manager_race_test.go`.
- Mutation relevance: Low; concurrency robustness rather than mutation score.

### DLQ-TC-103: Repeated integration runs leave no credentials or volumes

- Priority / level: P1; integration soak/security.
- Requirement mapping: DLQ-004 hardening.
- Setup: Five sequential runs with unique accounts.
- Steps: Execute and teardown each run; inspect filesystem logs and Docker resources.
- Exact assertions: All pass; no project containers/networks/volumes remain; reports do not contain plaintext passwords, JWT secret, or full bearer tokens.
- Automation location: `scripts/quality/tests/test_integration_soak.sh`.
- Mutation relevance: None.

### DLQ-TC-104: Equivalent-mutant exclusions are narrow and independently approved

- Priority / level: P1; mutation policy static test, required if an exclusion file is introduced.
- Requirement mapping: DLQ-005/006 hardening.
- Setup: Exclusion manifest fixtures.
- Steps: Parse entries and match them to source/tool report locations.
- Exact assertions: Each entry identifies one file and precise location/mutant, includes rationale and independent Review reference; glob, directory-wide, expired/missing location, duplicate, temporary environment override, or unreviewed exclusion exits non-zero.
- Automation location: `scripts/quality/tests/test_mutation_exclusions.py`.
- Mutation relevance: Prevents broad suppression of surviving mutants.

## 5. Test economy and data rules

- Table-driven tests consolidate JWT-invalid, UID-boundary, and report-schema variants while retaining a separate assertion and fixture name for every failure category.
- Report verifiers use small checked-in golden fixtures; they do not invoke Docker or mutation tools. Real tool smokes are limited to the fixed JWT and UID targets.
- Integration tests share one isolated environment per run but use unique records. They must be order-independent and must not depend on development Compose state.
- Integration evidence is allocated per run under `quality/integration/<COMPOSE_PROJECT_NAME>/`; runner, verifier, and diagnostics receive that path explicitly. `quality/integration/latest.json` points to the completed normal single run, while CI aggregation uploads `quality/integration/**` without flattening.
- Lifecycle tests inject fakes for exhaustive error/signal paths; only isolation, register/login, WebSocket, cleanup, and parallel cases require real Docker.
- The weak-test proof runs in disposable worktrees/tool sandboxes and never edits the caller's production or test files.
- P1 algorithm-confusion, race, soak, and exclusion-policy cases are deferred because P0 already establishes the requested delivery contract; they become gates when their related implementation/policy is introduced.

## 6. Coverage matrix

| Risk / requested area | P0 cases | Gate evidence |
|---|---|---|
| Develop Loop ordering and independent roles | 001–002 | Process contract test output |
| Backend JWT auth | 003–004, 011 | Go JSONL + integration report |
| WebSocket manager | 005–006 | Go JSONL package evidence |
| Frontend `uid.js`, `ws.js`, changed group controls and Chat wiring | 007–010, 037–038 | Vitest JSON + Stryker report |
| Real PostgreSQL/Redis/HTTP register-login | 011, 016 | Integration JSONL and Compose logs |
| Real authenticated protobuf WS ping/pong | 017 | Integration JSONL and WS assertions |
| Non-empty unit/integration reports and negative fixtures | 012–015, 018 | Verifier fixture test reports |
| Lifecycle, cleanup, signals, sentinel safety, parallel isolation | 019–022 | Wrapper call logs + Docker smoke results |
| Trusted mutation target/baseline | 023–024 | Target plan and wrapper test output |
| Gremlins/Stryker parsing and golden negatives | 025–026 | Fixture test output |
| Real mutation kill evidence and weak-test failure proof | 027–029 | Mutation JSON/summary + meta-test logs |
| GitHub event/base SHA/full history/no path filter | 030–033 | Workflow contract/matrix tests |
| Local/CI command parity | 034 | Make/workflow contract test |
| Traceability verifier | 035–036 | Traceability report and negative fixtures |

## 7. Implementation handoff checklist

- [ ] Preserve all stable Case IDs and include them plus mapped spec IDs in automated test names.
- [ ] Implement every P0 automation location or document an equivalent path in `traceability.json`; do not silently omit negative fixtures.
- [ ] Keep golden fixtures minimal, immutable, labeled with their source version (`go-1.25.3`, `vitest-4.1.10`, `gremlins-v0.6.0`, or `stryker-9.6.1`), and derived from retained real-schema samples; never generate expected failure fixtures from the parser under test.
- [ ] Make backend, frontend, and integration wrappers preserve runner exit codes and run report verification only after runner success.
- [ ] Make integration cleanup diagnostic-first on EXIT/INT/TERM; primary success plus cleanup failure must fail, while dual failure preserves the primary code and records cleanup failure.
- [ ] Use dynamic backend ports, project-scoped resources, dedicated credentials, and no PostgreSQL/Redis host ports.
- [ ] Pin Gremlins/Stryker/Vitest exactly as designed; emit planned and actual mutation targets and reject mismatches.
- [ ] Run the real fixed-target mutation smokes for empty, not-applicable, and deleted-only diffs.
- [ ] Keep weak-test proof inside disposable worktrees/sandboxes and assert the caller worktree is unchanged.
- [ ] Ensure workflow has no path filters or skip conditions, fetches full history, passes exact event base SHA, and always uploads evidence.
- [ ] Add traceability entries only after the exact test file and full test name exist; run all traceability negative fixtures.
- [ ] Record `make quality` results and report/artifact paths for independent Code Review; P0 is not complete if any required report is missing even when a command returned zero.

## 8. Revision Notes

- B1 → DLQ-TC-011 now asserts registration nickname equals the submitted username and login preserves it.
- B2 → DLQ-TC-016 independently health-checks PostgreSQL/Redis and scopes registration as the PostgreSQL readiness proof; DLQ-TC-017 proves Redis online participation and bounded cleanup through in-network inspection after authenticated WS.
- B3 → Section 1 and DLQ-TC-005/006 explicitly permit only the documented same-package manager seam and direct minimal `Server` construction without `NewServer` or external/global state.
- B4 → DLQ-TC-029 defines checked-in versioned manifests/patches, exact sandbox-diff/accounting checks, strong/weak controls, and acceptable documented weak outcome sets.
- RC1 → DLQ-TC-003 uses exact inclusive second-truncated bounds and an exact 168-hour expiry delta.
- RC2 → DLQ-TC-034/035 make `verify-traceability` an exact `quality-static` child reached by `make quality` and the workflow static job.
- RC3 → DLQ-TC-019/022 and the evidence rules define explicit per-project run directories, overwrite checks, `latest.json`, and non-flattening CI aggregation.
