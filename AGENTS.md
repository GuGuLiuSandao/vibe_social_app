# Repository Guidelines

## Project Structure & Module Organization
This repository is a split frontend/backend social app with shared Protocol Buffers contracts.

- `backend/`: Go services. Main entry is `backend/cmd/api/main.go`; domain logic lives in `backend/internal/<domain>` (for example `auth`, `chat`, `websocket`).
- `frontend/`: React + Vite app. UI routes/pages are in `frontend/src/pages`, reusable UI in `frontend/src/components`, client helpers in `frontend/src/lib`.
- `proto/`: Source `.proto` definitions shared by both apps.
- Generated code targets:
  - Go: `backend/internal/proto`
  - TypeScript: `frontend/src/proto`
- `docs/`: project notes and documentation.
- `docs/features/`: feature-by-feature delivery records. Each requirement gets its own subdirectory containing requirement design, technical design, and code review notes.

## Build, Test, and Development Commands
- `docker compose up -d db redis`: start local PostgreSQL and Redis only.
- `make proto-go`: regenerate Go protobuf bindings.
- `make proto-ts`: regenerate TypeScript protobuf bindings (requires frontend proto plugins).
- `make run-backend`: run backend API server on `:8080`.
- `make run-frontend`: run frontend dev server on `:5173`.
- `make build-backend`: build backend binary to `backend/bin/server`.
- `cd backend && go test ./...`: run all backend tests.
- `cd frontend && npm test`: run frontend Vitest suite once.

## Coding Style & Naming Conventions
- Go: follow `gofmt` defaults (tabs, standard formatting), lowercase package names, and keep new business logic under `backend/internal/<domain>`.
- React: component/page files use PascalCase (for example `Login.jsx`), utility modules use lowercase/camelCase names in `src/lib`.
- Match existing JS style in this repo: ESM imports, double quotes, semicolons.
- Do not hand-edit generated protobuf output under `backend/internal/proto` or `frontend/src/proto`; regenerate from `proto/`.

## Testing Guidelines
- Backend tests use Go’s `testing` package and `*_test.go` naming.
- Frontend tests use Vitest + Testing Library with `*.test.jsx` / `*.test.js`.
- No enforced coverage threshold is configured; add focused tests for new handlers, websocket behavior, and page-level interactions.
- After changing `proto/`, regenerate bindings and run both backend and frontend tests.
- M0 baseline gate (must pass before closing a task):
  - `cd backend && go test ./...`
  - `cd frontend && npm test`
  - If chat flow changed, run `docs/plans/M0_MANUAL_QA.md`.

## Roadmap Execution Rules
- Treat these two planning docs as mandatory execution context for all feature and refactor work:
  - `docs/plans/ROADMAP.md`
  - `docs/plans/M0_TASKS.md`
- Before implementation, map the task to a roadmap milestone and a concrete checklist item in `M0_TASKS.md` (or add one if missing).
- After finishing any task, update both planning docs in the same change:
  - `ROADMAP.md`: reflect milestone progress, timeline/risk changes, or scope changes.
  - `M0_TASKS.md`: mark checklist status and add short outcome notes.
- Do not consider a task complete until code, tests, and both planning docs are updated consistently.

## Feature Delivery Workflow Rules
- All feature development must follow this mandatory sequence:
  1. Requirement design document (`需求设计文档`)
  2. Technical solution document (`技术方案文档`)
  3. Feature implementation
  4. Code review and review note capture
  5. Git commit
- Do not start implementation before the requirement design doc and technical solution doc exist for that feature.
- Do not create the final commit before code review feedback is recorded.
- Every new feature/request must create or reuse a dedicated subdirectory under `docs/features/`, named by requirement scope (example: `docs/features/block-system-v1/`).
- Each feature subdirectory must contain at least:
  - `requirement.md` — requirement design
  - `technical-solution.md` — technical design / implementation plan
  - `code-review.md` — code review findings, decisions, and follow-up notes
- After the commit is created, append the final commit hash and commit message to `technical-solution.md` for traceability.
- If a requirement changes during implementation, update the requirement doc and technical solution doc before continuing code changes.

## Commit & Pull Request Guidelines
- Current history favors short, imperative commit subjects; optional prefixes are already used (for example `init: ...`).
- Preferred commit format: `<type>: <summary>` (for example `feat: add chat reconnect guard`), keep subject lines concise.
- PRs should include:
  - what changed and why,
  - impacted modules (`backend/internal/...`, `frontend/src/...`, `proto/...`),
  - test commands run and results,
  - linked issue/ticket,
  - screenshots for UI changes.
