# Requirement Contract: social-app-develop-loop-quality

## 0. Metadata

- Change ID: `social-app-develop-loop-quality`
- Rigor: `standard`
- Areas: engineering process / backend / frontend / integration / CI
- Behavior specification impact: establish the initial specification and traceability baseline

## 1. Requirement Facts

- Original request: 在 `social_app` 中完整落实从需求到 MR/PR 的 Develop Loop，纳入独立测试设计、测试评审、测试实现、单元/集成门禁和 Mutation Testing，而不是只记录文档流程。
- Goal and successful outcome:
  - Requirement 确认后，由独立 Test Designer 编写 Cases，并由独立 Test Reviewer 放行。
  - Implementer 同时实现生产变更与 Cases 对应的自动化测试。
  - 本地和 GitHub Actions 执行同构的后端、前端、集成和 Mutation 门禁。
  - Code Reviewer 能从规格/验收 ID 追溯到 Case、测试实现和质量证据。
- Confirmed scope:
  - 修正 Engineering Loop 的 Develop Loop 核心，使独立测试设计和测试评审成为代码变更的必经环节。
  - 为 Go 后端建立单元/包级回归测试入口。
  - 为 React 前端建立 Vitest 组件/模块测试入口。
  - 建立使用隔离 PostgreSQL、Redis 和真实后端进程的黑盒集成测试入口，覆盖代表性的 HTTP 与 WebSocket 链路。
  - 为可测试的生产逻辑建立 Mutation Testing，并在本地和 PR CI 产生明确通过/失败信号。
  - 建立统一 Makefile 命令、GitHub Actions 和质量证据。
  - 建立首批稳定行为规格，并让测试名称引用规格 ID。
- Out of scope:
  - Maintenance Loop、Bug Fix Loop、定时巡检和质量趋势。
  - 部署、发布、镜像推送和生产环境验收。
  - 一次性补齐所有历史业务行为的完整测试覆盖。
  - 访问共享或生产 PostgreSQL、Redis 及其他真实外部系统。
- Dependencies:
  - Go 标准 `testing`。
  - 前端测试运行器及 React 测试工具。
  - 项目技术栈适配的 Mutation Testing 工具。
  - GitHub Actions 和 Docker Compose。

## 2. Functional Contract

### Capability: independent test design and review

- Trigger: Requirement Contract 获得确认并进入代码变更流程。
- Input: Requirement、Technical Design、长期行为规格和仓库现状。
- Behavior: 独立 Test Designer 编写自动化 Cases；独立 Test Reviewer 按风险覆盖、断言有效性、规格追溯、隔离和经济性评审。
- Output / feedback: `testcases.md` 与 `testcases-review.md`；Review 通过前 Implementer 不开始 Coding。
- Permissions: Reviewer 只维护评审产物，不修改 Cases 或代码。
- Errors: blocker 或分数未达阈值时返回 Test Designer 修改，最多三轮。

### Capability: executable test gates

- Trigger: Implementer 完成 Cases 对应实现。
- Input: 后端、前端、集成测试和项目质量配置。
- Behavior: 本地统一入口执行各层测试；CI 使用干净环境重复同一命令。
- Output / feedback: 稳定退出状态、可读日志和 CI 结论。
- Permissions: 测试使用隔离数据和测试凭据。
- Errors: 任一必需门禁失败则变更不可交付。

### Capability: mutation quality gate

- Trigger: PR 修改 Mutation 范围内的可测试生产逻辑。
- Input: 生产代码、对应测试和可信 Git diff。
- Behavior: 先验证普通测试 baseline，再执行 Mutation Testing；存活 mutant、未覆盖 mutant、工具错误或不可信结果使门禁失败；无适用生产目标时输出可审计的跳过原因。
- Output / feedback: 机器可读摘要和人可读诊断。
- Permissions: 在隔离工作目录或工具沙箱中运行，不污染调用者工作区。
- Errors: baseline 失败、目标发现失败、Mutation 工具失败或结果不满足阈值时非零退出。

## 3. Rules

| Scenario | Condition | Behavior | Expected result |
|---|---|---|---|
| 行为代码变更 | 影响后端或前端对外行为 | 先设计并评审 Cases，再实现代码和测试 | Case、测试实现和规格 ID 可追溯 |
| 后端变更 | 修改 Go 生产代码 | 执行 build、vet、unit；适用时执行 integration 和 mutation | 所有必需命令通过 |
| 前端变更 | 修改 React 或客户端逻辑 | 执行 unit/component test 和 build；适用时执行 mutation | 所有必需命令通过 |
| 跨层链路变更 | 修改 HTTP、WebSocket、Proto、DB 或 Redis 协作 | 执行隔离的真实依赖集成测试 | 黑盒链路通过且不使用共享数据 |
| Mutation 有目标 | 可信 diff 包含受支持生产逻辑 | 对目标执行 Mutation Testing | 无存活、未覆盖或错误 mutant |
| Mutation 无目标 | 仅文档、生成物或不适用文件 | 不运行 Mutation 工具 | 记录目标为零和跳过原因并通过 |
| CI 与本地 | 同一变更 | CI 调用仓库统一入口 | 不维护两套行为不同的命令 |

## 4. Boundary Cases

| Case | Trigger | Expected behavior | Notes |
|---|---|---|---|
| 普通测试 baseline 失败 | Mutation 前已有红灯 | Mutation 立即失败且不报告虚假通过 | fail closed |
| 生成代码变化 | Proto 生成文件进入 diff | 生成物不直接作为 Mutation 目标 | 由源 Proto 与行为测试保护 |
| 外部依赖不可用 | 集成环境未就绪 | 有界等待后失败并保留诊断 | 不无限重试 |
| 测试只断言成功状态 | Reviewer 发现弱断言 | Test Cases Review 或 Code Review 不通过 | 必须断言响应和副作用 |
| 工具没有产生测试 | Mutation 或测试运行器报告零测试 | 门禁失败 | 防止空跑绿灯 |

## 5. Acceptance Criteria

| ID | Given | When | Then |
|---|---|---|---|
| DLQ-001 | 已确认 Requirement 和 Design | 独立 Test Designer/Reviewer 执行流程 | 产生通过评审的 Cases，Coding 在 Review 后开始 |
| DLQ-002 | 当前 Go 后端 | 执行后端测试入口 | 至少一组稳定行为规格测试通过，命令非空执行 |
| DLQ-003 | 当前 React 前端 | 执行前端测试入口 | 模块/组件测试通过，命令非空执行 |
| DLQ-004 | 隔离 PostgreSQL、Redis 和后端 | 执行集成测试入口 | 代表性 HTTP 与 WebSocket 链路通过，环境可清理 |
| DLQ-005 | 有普通测试保护的生产逻辑 | 执行 Mutation Testing | 代表性 mutant 被测试杀死，摘要可审计 |
| DLQ-006 | Mutation 范围引入弱测试 | 执行 Mutation Testing | 存活或未覆盖 mutant 使命令失败 |
| DLQ-007 | PR 修改项目代码 | GitHub Actions 运行 | 后端、前端、集成和适用 Mutation 门禁均执行 |
| DLQ-008 | Code Reviewer 读取交付产物 | 审查测试实现 | 能从规格/验收 ID 追溯到 Case、测试文件和命令结果 |
| DLQ-009 | 开发者使用统一入口 | 本地执行质量命令 | 与 CI 调用的命令一致且结果可重复 |

## 6. Confirmation

- Confirmed decisions:
  - Develop Loop 必须包含独立 Tester、独立 Test Reviewer、测试实现、本地/CI 单元与集成门禁，以及 Mutation Testing。
  - 本次只完成 Develop Loop，不涉及 Maintenance。
- Contract changes after confirmation: none
