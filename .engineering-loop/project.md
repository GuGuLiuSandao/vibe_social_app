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

| Change scope | Command | Required result |
|---|---|---|
| Any tracked change | `git diff --check` | Exit 0 |
| Backend Go | `cd backend && go build ./...` | Exit 0 |
| Backend Go | `cd backend && go vet ./...` | Exit 0 |
| Frontend | `cd frontend && npm run build` | Exit 0 |
| Protocol Buffers | `make proto-go && make proto-ts` | 生成成功，并继续执行后端和前端门禁 |
| Compose / deployment configuration | `docker compose config --quiet` | Exit 0 |
| Behavior change | 执行 Technical Design 和 Test Cases 声明的回归验证 | 所有适用验证通过并记录证据 |

## Test Isolation

- Local test data: 使用测试自身创建的临时或专用数据，保证可重复执行和顺序独立。
- External systems: 默认使用本地 PostgreSQL、Redis、mock、fake 或专用测试环境。
- Credentials and sensitive data: 使用测试专用凭据和环境变量；质量验证不写入共享或生产数据。

## Delivery

- Commit convention: `<type>: <summary>`
- Remote platform: GitHub
- Target branch: `master`
- MR/PR creation: 完成独立 Code Review 和本地质量门禁后创建 PR。
- CI policy: 等待仓库已配置的 PR 检查；PR 记录本地质量证据。
- Merge authority: 由用户决定，Agent 不自动合并。

## Critical Change Controls

- Security and permissions: 明确鉴权主体、权限矩阵、失败行为和越权用例。
- Data and migrations: 明确兼容、回滚、数据保全和部署顺序。
- External writes: 优先使用 mock、dry-run 或专用测试环境；真实写入需要明确授权。
- Additional approvals: Requirement Contract 对关键边界完成确认，Design/Test/Code Review 均无 blocker。
