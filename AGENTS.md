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

## Build and Development Commands
- `docker compose up -d db redis`: start local PostgreSQL and Redis only.
- `make proto-go`: regenerate Go protobuf bindings.
- `make proto-ts`: regenerate TypeScript protobuf bindings (requires frontend proto plugins).
- `make run-backend`: run backend API server on `:8080`.
- `make run-frontend`: run frontend dev server on `:5173`.
- `make build-backend`: build backend binary to `backend/bin/server`.
- `cd backend && go build ./...`: compile all backend packages.
- `cd frontend && npm run build`: create the frontend production bundle.

## Coding Style & Naming Conventions
- Go: follow `gofmt` defaults (tabs, standard formatting), lowercase package names, and keep new business logic under `backend/internal/<domain>`.
- React: component/page files use PascalCase (for example `Login.jsx`), utility modules use lowercase/camelCase names in `src/lib`.
- Frontend UI components must default to `shadcn/ui` primitives under `frontend/src/components/ui`; avoid introducing new third-party UI kits for core controls.
- Match existing JS style in this repo: ESM imports, double quotes, semicolons.
- Do not hand-edit generated protobuf output under `backend/internal/proto` or `frontend/src/proto`; regenerate from `proto/`.

## Develop Loop

Requirement-driven changes follow the repository Develop Loop:

- Project adapter: `.engineering-loop/project.md`
- Change artifacts: `docs/changes/<change-id>/`
- Long-term behavior specifications: `docs/specs/`

The general methodology is maintained in the separate `engineering-loop` repository. When adopting or updating the methodology, provide that repository URL or path to the AI so it can read the source process, roles, and templates alongside this repository. This repository keeps only its project adapter, executable Agent definitions, and completed change artifacts.

The standard project flow is:

```text
Requirement Contract
→ Technical Design
→ Design Review PASS
→ Test Design
→ Test Review PASS
→ Implementation
→ Code Review
→ Local Quality Gates
→ PR / CI
```

Before implementation, establish the change ID and rigor level, then create the artifacts required by the core process. Standard and critical changes proceed through Requirement Contract, Technical Design Review, Test Cases Review, implementation, independent Code Review, local quality gates, and PR/CI delivery. Quick changes may combine design and test notes into the requirement, while independent Code Review and applicable quality gates remain required.

Requirement clarification is not a routine questionnaire. Use `$grilling` one decision at a time only when an unresolved choice would materially change user experience, architecture, data, permissions, acceptance boundaries, or important failure behavior. Investigate facts available from the repository, runtime, and existing specifications directly. Record confirmed decisions and rationale in the Requirement Contract, not the full question-and-answer transcript.

When implementation would change confirmed product rules, scope, or acceptance criteria, update the Requirement Contract and reconfirm the affected decision before continuing.

## Change Classification and Quality Policy

Local gates and GitHub Actions classify the actual diff against the trusted master base. A label is a declaration that may increase checks; it never reduces checks.

| Classification | Diff and declaration | Required gates |
|---|---|---|
| `docs` | Only ordinary documentation such as `README.md`, `docs/*.md`, `docs/backend/**`, or `docs/plans/**` | `make quality-docs` |
| `engineering` | Process, CI, agent, quality tooling, stable specifications, change artifacts, Make/Compose, or repository configuration; no product paths | `make quality-engineering` |
| `develop` | PR has the `develop-loop` label and all changed paths are recognized | Complete six-gate suite through `make quality-develop` |

`backend/**`, `frontend/**`, and `proto/**` are product paths and require the `develop-loop` label. A product change without that label, an unknown path, or an empty comparison is an invalid classification and must fail closed. `docs/specs/**` and `docs/changes/**` are engineering artifacts rather than ordinary documentation.

Run `make quality` for normal local validation. It compares the branch and worktree with `origin/master` (override with `BASE_SHA=<sha>`), classifies the paths, and dispatches the matching gate. For an established requirement change, use `CHANGE_LABELS=develop-loop make quality`. This local declaration is mirrored by the PR label; CI independently reads the PR diff and labels.

Every PR runs the classifier and exactly one applicable quality branch. The fixed `quality-gate` job evaluates the classifier and all required branch results and is the required master status check. Master accepts changes through PRs only; direct pushes are protected by GitHub branch protection. A push produced by merging to master reruns the complete suite as post-merge verification.
