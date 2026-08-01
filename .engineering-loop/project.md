# Social App Develop Loop Profile

## Project

- Name: `social_app`
- Repository purpose: Go + React 实时社交应用，前后端共享 Protocol Buffers 契约。
- Primary rules: `AGENTS.md`

## Change Identity

- Change ID format: 小写英文、数字和短横线组成的稳定主题标识；产品版本仅在需求明确属于某个发布版本时加入。
- Branch format: `codex/<change-id>`
- Artifact root: `docs/changes/<change-id>/`

## Repository Map

| Area | Path | Responsibility |
|---|---|---|
| Backend | `backend/` | Go API、领域服务、数据库和 WebSocket |
| Frontend | `frontend/` | React 页面、组件和客户端逻辑 |
| Contracts | `proto/` | 前后端共享 Protocol Buffers 源定义 |
| Go generated | `backend/internal/proto/` | Go Protobuf 生成代码 |
| TypeScript generated | `frontend/src/proto/` | TypeScript Protobuf 生成代码 |
| Product planning | `docs/plans/ROADMAP.md` | 产品版本、能力和建设方向 |

## Long-term Specifications

| Behavior domain | Specification path | Traceability rule |
|---|---|---|
| Stable product behavior | `docs/specs/` | 对外行为变更维护稳定规格 ID；Requirement、Test Cases 和自动化测试引用同一 ID |
| Shared wire contracts | `proto/` | Proto 是通信契约源；生成代码与源定义保持同步 |

## Quality Gates

| Gate | Command | Required result |
|---|---|---|
| Static/build/traceability | `make quality-static` | diff、Go build/vet、frontend build、两份 Compose、流程契约和追溯全部通过 |
| Backend unit | `make test-backend` | Go JSONL 可解析；auth/websocket 均执行，至少 6 个测试事件 |
| Frontend unit | `make test-frontend` | Vitest JSON 可解析；UID/WS 两套测试均执行，至少 6 个 Case |
| Real integration | `make test-integration` | 隔离 PostgreSQL/Redis/HTTP/WS 流程和清理通过 |
| Backend mutation | `make mutation-backend` | Gremlins `v0.6.0` 报告非空且无 lived/uncovered/error 状态 |
| Frontend mutation | `make mutation-frontend` | Stryker `9.6.1` 报告非空且全部 `Killed` |
| Full Develop Loop aggregate | `make quality-develop` | 上述六个门禁全部通过，任何子命令失败均非零退出 |
| Classified local entry | `make quality` | 基于相对 `origin/master` 的实际 diff 选择 docs、engineering 或 develop 门禁；需求变更使用 `CHANGE_LABELS=develop-loop make quality` |
| Protocol Buffers | `make proto-go && make proto-ts` | Proto 变更时生成成功，并继续执行 `CHANGE_LABELS=develop-loop make quality` |

## Test Isolation

- Local test data: 使用测试自身创建的临时或专用数据，保证可重复执行和顺序独立。
- External systems: 默认使用本地 PostgreSQL、Redis、mock、fake 或专用测试环境。
- Credentials and sensitive data: 使用测试专用凭据和环境变量；质量验证不写入共享或生产数据。

## Delivery

- Commit convention: `<type>: <summary>`
- Remote platform: GitHub
- Target branch: `master`
- MR/PR creation: 完成独立 Code Review 和本地质量门禁后创建 PR。
- CI policy: `.github/workflows/develop-quality.yml` 对 PR 实际 diff 做 docs、engineering、develop 分类。docs 和 engineering 分别执行适用门禁；带 `develop-loop` 标签的需求变更执行完整六项门禁。固定 `quality-gate` 汇总结果；master 合并后的 push 再执行完整六项门禁。
- Branch policy: `master` 只接受 PR 合并，`quality-gate` 是必需状态检查，禁止直接 push。
- Merge authority: 由用户决定，Agent 不自动合并。

## Critical Change Controls

- Security and permissions: 明确鉴权主体、权限矩阵、失败行为和越权用例。
- Data and migrations: 明确兼容、回滚、数据保全和部署顺序。
- External writes: 优先使用 mock、dry-run 或专用测试环境；真实写入需要明确授权。
- Additional approvals: Requirement Contract 对关键边界完成确认，Design/Test/Code Review 均无 blocker。
