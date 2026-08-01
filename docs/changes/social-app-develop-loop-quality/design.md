# Technical Design: social-app-develop-loop-quality

## 0. Inputs

- Requirement: `docs/changes/social-app-develop-loop-quality/requirement.md`
- Project Profile: `.engineering-loop/project.md`
- Related behavior specifications: `docs/specs/auth.md`, `docs/specs/websocket.md`, `docs/specs/frontend-client.md`
- Code entry points: `backend/internal/auth`, `backend/internal/websocket`, `frontend/src/lib`, `Makefile`, `docker-compose.yml`

## 1. Current Code Map

| Area | File | Current responsibility | Change |
|---|---|---|---|
| Develop core | `.engineering-loop/develop/README.md` | 通用需求交付流程 | 明确独立 Test Design/Review、测试实现、unit/integration/mutation 和 CI 同构门禁为代码变更必经环节 |
| Project profile | `.engineering-loop/project.md` | social_app 适配 | 用可执行测试命令替换泛化的“按设计验证”入口 |
| Backend auth | `backend/internal/auth/jwt.go`、`handler.go` | JWT 与 HTTP 认证行为 | 以规格 ID 建立 Go 测试；不因测试引入业务行为变化 |
| WebSocket | `backend/internal/websocket/handler.go`、`manager.go` | 握手、心跳和连接管理 | 建立包级测试与真实 WS 黑盒测试 |
| Frontend client | `frontend/src/lib/uid.js`、`ws.js` | UID 解析和 WS 请求构造 | 建立 Vitest 模块测试，作为 Stryker 首批目标 |
| Integration | `docker-compose.yml` | 开发环境 | 新增 `docker-compose.integration.yml`，使用隔离服务、临时 volume 和独立宿主端口 |
| Commands | `Makefile` | 构建和运行 | 增加 unit、integration、mutation、quality 分层入口 |
| CI | `.github/workflows/` | 当前无 PR 检查 | 新增 Develop Quality workflow，调用与本地相同入口 |

## 2. Data / API / Schema / Configuration

| Type | Location | Change | Compatibility strategy |
|---|---|---|---|
| Go tests | `backend/internal/**/*_test.go` | 使用标准 `testing`，测试名包含规格 ID 的下划线形式 | 不引入测试框架运行时依赖 |
| Frontend tests | `frontend/src/**/*.test.jsx?` | Vitest + Testing Library + jsdom | 仅 devDependencies；生产 bundle 不包含测试依赖 |
| Frontend config | `frontend/vite.config.js`、`frontend/src/test/setup.js` | 注册 jsdom、setup 和非 watch 测试入口 | 保持现有 Vite build 行为 |
| Integration compose | `docker-compose.integration.yml` | PostgreSQL、Redis、backend；独立 project/port/credentials | 不复用开发 volume，不连接共享环境 |
| Integration tests | `backend/tests/integration/*.go` | `integration` build tag；黑盒调用 HTTP/WS | 普通 `go test ./...` 不启动外部依赖 |
| Mutation config | `frontend/stryker.config.mjs` | Stryker + Vitest、100% break threshold、JSON/clear-text report | 由 wrapper 注入可信 diff 的 mutate 列表 |
| Mutation wrappers | `scripts/quality/run-*-mutation.*` | baseline、目标发现、执行和摘要验证 | 生成物输出到忽略的 `quality/`，不修改生产源码 |
| CI | `.github/workflows/develop-quality.yml` | backend/frontend/integration/mutation jobs | `fetch-depth: 0`，PR 使用 base SHA，push 使用父提交 |

## 3. Behavior Changes

| Interface / flow | Input | Output | Error behavior | Permissions |
|---|---|---|---|---|
| `make test-backend` | Go module | 非空 Go 测试结果 | 零测试或失败非零退出 | 本地进程，无外部写入 |
| `make test-frontend` | frontend source/tests | 非空 Vitest 结果 | 零测试或失败非零退出 | jsdom |
| `make test-integration` | integration compose + black-box tests | HTTP 注册/登录及 WS 鉴权/ping 证据 | 服务未就绪、零测试或断言失败非零退出并打印 compose logs | 专用 DB/Redis/凭据 |
| `make mutation-backend` | `MUTATION_BASE_SHA` 到 HEAD 的可信 diff | Gremlins JSON + 摘要 | baseline、祖先校验、零测试、LIVED/NOT COVERED、工具错误失败；无目标记录 skipped | 隔离工具执行 |
| `make mutation-frontend` | 同上 | Stryker JSON + 摘要 | baseline、祖先校验、零测试、survived/noCoverage/error、低于 100% 失败；无目标记录 skipped | Stryker sandbox |
| PR workflow | GitHub PR SHA/base SHA | 四层 Check | 任一必需 Job 失败则 PR Check 失败 | GitHub Actions 最小权限 `contents: read` |

Mutation 目标规则：

- 后端只选择 `backend/**/*.go` 生产文件，排除 `*_test.go`、`backend/internal/proto/**` 和生成文件；映射为唯一 Go package，由 Gremlins 执行。
- 前端只选择 `frontend/src/**/*.{js,jsx}`，排除 `*.test.*`、`frontend/src/proto/**`、测试 setup 和纯样式/资源；Stryker 使用精确文件列表。
- base SHA 必须存在且为 HEAD 祖先；CI 显式传入 `github.event.pull_request.base.sha`。本地必须显式传入或能解析 `origin/master`。
- 当前变更没有生产 diff 时，另设 `mutation-smoke` 固定验证 `backend/internal/auth/jwt.go` 和 `frontend/src/lib/uid.js`，证明工具、测试和阈值有效。

## 4. Implementation Constraints

- Preserve: 现有 HTTP、WebSocket、Proto、数据库模型和前端产品行为；测试基础设施不得改变响应契约。
- Reuse: Go `testing`、现有 Protobuf 类型、Gorilla WebSocket、Dockerfiles 和 Makefile。
- Change boundaries:
  - Develop Loop 的代码变更不再允许跳过独立 Test Designer/Test Reviewer；quick 只允许压缩文档篇幅。
  - Coding 必须实现已评审 Cases；Code Review 必须逐项追溯。
  - CI 调用 Makefile/scripts，不复制测试逻辑。
  - 集成测试拥有独立 Compose project、数据库、Redis 和端口；退出时总是清理。
  - Mutation 先跑普通 baseline；任何无法信任的结果 fail closed。
  - Proto 生成物不直接 mutation，源契约由生成校验和黑盒测试保护。
  - 工具版本固定：Stryker core/vitest runner 同一锁定版本；Gremlins 在 CI 使用固定版本安装。

## 5. Test Mapping

| Acceptance / specification ID | Test level | Target | Notes |
|---|---|---|---|
| DLQ-001 | process/static | Develop core、Agent TOML、交付文档 | Test Review 通过前不得出现生产/测试实现 |
| DLQ-002 / AUTH-001 | Go unit | `backend/internal/auth/jwt_test.go` | token round-trip、错误 secret、非法 token |
| DLQ-002 / WS-001 | Go package | `backend/internal/websocket/manager_test.go` | 同 UID 替换、陈旧连接注销保护、快照 |
| DLQ-003 / CLIENT-001 | Vitest | `frontend/src/lib/uid.test.js` | parse/whitelist 边界和非法输入 |
| DLQ-003 / CLIENT-002 | Vitest | `frontend/src/lib/ws.test.js` | token URL 编码、请求 ID 和消息类型 |
| DLQ-004 / AUTH-HTTP-001 | integration | `backend/tests/integration/auth_flow_test.go` | register→login，断言 protobuf 响应和 token |
| DLQ-004 / WS-HTTP-001 | integration | `backend/tests/integration/websocket_flow_test.go` | token handshake→protobuf ping/pong |
| DLQ-005/006 | mutation smoke + negative wrapper tests | JWT、UID、mutation wrapper | 代表 mutant 被杀；伪造 survivor/零测试摘要会失败 |
| DLQ-007/009 | CI/static | workflow + Make targets | CI 与本地调用同一入口 |
| DLQ-008 | review traceability | `code-review.md` | 规格→Case→测试文件→命令结果 |

## 6. Risks

| Risk | Trigger | Protection |
|---|---|---|
| Mutation 对大文件成本失控 | UI/handler 大范围变更 | 精确 diff 文件/包作用域、并行独立 Job、机器摘要 |
| 等价 mutant 阻塞 | 工具生成不改变可观察行为的 mutant | 只允许精确、带理由的项目映射/排除；不得降低全局阈值 |
| 集成测试污染开发数据 | 复用默认 compose | 专用 compose、随机 project name、独立凭据和 `down -v` trap |
| CI 假绿 | 零测试、零目标误判 | 测试收集计数、mutation 明确 skipped/failed 状态、wrapper 自测 |
| WS 测试竞态 | 后端启动或异步注册未完成 | 有界 readiness 和消息 deadline，禁止固定长 sleep |
| Develop core 与安装源漂移 | 只改 social_app 副本 | 同步修改 `engineering-loop` 源并用 installer `--update` 校验一致性 |

## 7. Implementation Tasks

- Task 1: 修正 `engineering-loop` Develop 核心/模板/Agent 约束并同步安装副本，更新 social_app Profile。
- Task 2: 建立 `docs/specs` 首批行为规格、Go unit/package tests 和统一 backend test 命令。
- Task 3: 安装 Vitest/Testing Library/Stryker，建立前端测试和 mutation smoke。
- Task 4: 新增 integration compose、带 build tag 的 HTTP/WS 黑盒测试和可靠清理脚本。
- Task 5: 实现可信 diff 的后端/前端 mutation wrappers、机器摘要、零测试与错误结果防护。
- Task 6: 新增 GitHub Actions jobs，运行 backend、frontend、integration、mutation 和 build/vet/config 门禁。
- Task 7: 执行所有门禁、记录质量证据并进入独立 Code Review。

## 8. Executable Gate Contracts

本节是实现门禁的权威协议；与前文概述冲突时以本节为准。

### 8.1 Unit test reports and minimum execution

`scripts/quality/run-backend-tests.sh`：

1. 创建 `quality/`，执行 `cd backend && go test -json ./...`，原始输出写入 `quality/backend-test.jsonl`。
2. 保留 `go test` 原始退出码；非零立即失败。
3. `scripts/quality/verify-go-test-report.py` 逐行解析 JSON：每行必须是对象；统计 `Action=run` 且 `Test` 非空的事件。
4. `social_app/internal/auth` 和 `social_app/internal/websocket` 各至少出现一个测试，总测试数至少 6；报告缺失、空、非法 JSON、缺少必需 package 或计数不足均失败。
5. 校验器只在原始测试通过后执行；Make target 直接调用 wrapper，不使用会吞退出码的 pipe。

`scripts/quality/run-frontend-tests.sh`：

1. 执行 `npm --prefix frontend test -- --reporter=json --outputFile=../quality/frontend-test.json`；`test` 固定为 `vitest run --passWithNoTests=false`。
2. 保留 Vitest 原始退出码；非零立即失败。
3. `scripts/quality/verify-frontend-test-report.mjs` 校验 JSON object、`numTotalTestSuites >= 2`、`numTotalTests >= 6`、`numFailedTests = 0`，并确认 UID 与 WS 两个测试文件均出现在结果中。
4. 报告缺失、空、schema/类型不符、过滤后零测试或计数不足均失败。

两个 verifier 各有 fixture 驱动的负向测试，覆盖：报告缺失、空/损坏、零测试、缺少必需 suite/package、底层测试失败和成功报告。wrapper 测试确认退出码原样传播。

### 8.2 Integration lifecycle state machine

唯一入口 `scripts/quality/run-integration-tests.sh`：

```text
INIT
→ allocate unique COMPOSE_PROJECT_NAME = social-app-it-<pid>-<random>
→ docker compose -f docker-compose.integration.yml build
→ up -d
→ discover backend endpoint with `docker compose port backend 8080`
→ readiness loop (maximum 90s, each curl 3s)
→ go test -tags=integration -json ./tests/integration
→ verify report: AUTH-HTTP and WS-HTTP suites present, total >= 2
→ capture compose ps/logs
→ down --volumes --remove-orphans
```

- PostgreSQL/Redis 不发布宿主端口；backend 使用 `127.0.0.1::8080` 动态端口，实际 endpoint 通过 `docker compose port` 注入 `INTEGRATION_BASE_URL`。
- Compose 使用专用 DB 名、用户、密码、JWT secret 和 project-scoped network/volumes。后端启动时执行现有 AutoMigrate，readiness 必须收到 `/api/v1/auth/login` 的预期 `415`，从而证明 DB/Redis 初始化和 HTTP router 可用。
- wrapper 在 `EXIT/INT/TERM` trap 中先保存 `ps` 和 `logs` 到 `quality/`，再执行 `down --volumes --remove-orphans`，最后返回最初失败码。启动、端口发现、readiness、测试、报告验证或清理失败均不得形成成功。
- readiness 每次请求 3s，总上限 90s；测试命令总上限由 CI job `timeout-minutes: 15` 提供。
- wrapper 测试通过可注入的 `DOCKER_COMPOSE_BIN` fake 验证启动失败、测试失败、INT/TERM 与日志/清理顺序；两次并行 smoke 验证 project name 和动态端口不冲突。

### 8.3 Mutation tool and report contracts

固定版本：

- Go Gremlins `v0.6.0`，安装后 `gremlins --version` 必须匹配。
- `@stryker-mutator/core`、`@stryker-mutator/vitest-runner` 固定 `9.6.1`，Vitest 固定 `4.1.10`，由 `package-lock.json` 锁定。

可信目标发现：

1. `MUTATION_BASE_SHA` 必须由调用者传入；wrapper 验证 base/head commit 存在且 `base` 是 `HEAD` 祖先。
2. 使用 `git diff --name-status --find-renames <base>...HEAD`；解析失败、未知状态、submodule 或路径越界失败。
3. Go 将合格生产文件映射为 package；Gremlins 实际目标允许扩张到这些 package 内的非生成生产 `.go` 文件，报告出现计划 package 外文件即失败。
4. Frontend Stryker 的 `mutate` 是精确变更文件列表；报告出现列表外文件或遗漏计划文件即失败。
5. 删除/重命名仅旧路径没有可执行目标时记录 `deleted-only`；仅不适用文件记录 `not-applicable`。这两种和完全无 diff 都必须运行固定 mutation smoke，而不是直接绿灯。

后端执行：先运行 `make test-backend`，再对计划 package 执行：

```bash
gremlins unleash --output quality/backend-mutation.json \
  --threshold-efficacy 100 --threshold-mutant-coverage 100 <packages...>
```

验证 Gremlins JSON：`mutants_total > 0`；顶层计数为非负整数且满足总数不变量；`mutants_lived = 0`、`mutants_not_covered = 0`；文件 mutation status 只允许 `KILLED`、`NOT_VIABLE`，出现未知状态失败；报告缺失/空/非法、工具非零/信号/timeout、计划与实际 package 不一致均失败。

前端执行：先运行 `make test-frontend`，wrapper 生成临时配置并执行：

```bash
npm --prefix frontend exec -- stryker run <config>
```

Stryker 配置包含 `testRunner: vitest`、JSON/clear-text reporter、`break: 100`。验证 mutation-testing report：mutant 总数 `> 0`；每个 mutant 状态只允许 `Killed`；`Survived/NoCoverage/Timeout/RuntimeError/CompileError/Ignored/Pending` 或任何未知状态均失败；报告缺失/空/非法、零 mutant、工具异常、计划与实际文件不一致均失败。

固定 smoke 目标为 `backend/internal/auth/jwt.go` 和 `frontend/src/lib/uid.js`。它在无 diff/仅不适用/删除-only 时作为 required mutation check 运行，也在本次落地验收中运行。等价 mutant 只能通过仓库中的逐文件逐位置映射排除，记录理由和关联 Review；宽泛 glob、临时 CI 参数或解析失败均失败。

Mutation verifier 负向 fixture 覆盖报告缺失、非法、零 mutant、每个失败状态、未知状态、计数不变量、目标越界、base 非祖先和工具异常。

### 8.4 Local quality DAG

| Target | Exact children |
|---|---|
| `quality-static` | `git diff --check`、`cd backend && go build ./...`、`go vet ./...`、`npm --prefix frontend run build`、`docker compose config --quiet`、`docker compose -f docker-compose.integration.yml config --quiet` |
| `test-backend` | backend test wrapper + report verifier |
| `test-frontend` | frontend test wrapper + report verifier |
| `test-integration` | integration lifecycle wrapper + report verifier |
| `mutation-backend` | target discovery + backend baseline + Gremlins + report verifier；无目标运行 smoke |
| `mutation-frontend` | target discovery + frontend baseline + Stryker + report verifier；无目标运行 smoke |
| `quality` | `quality-static test-backend test-frontend test-integration mutation-backend mutation-frontend` |

任一 child 非零时聚合目标非零。Project Profile 更新为以上命令矩阵；CI 只调用这些 target。

### 8.5 GitHub Actions event matrix

workflow 不使用 path filter，所有事件都运行 static、backend、frontend、integration、backend mutation、frontend mutation 六个 jobs。

| Event | Checkout HEAD | Mutation base | Trust rule |
|---|---|---|---|
| `pull_request` | `pull_request.head.sha` | `pull_request.base.sha` | 两 SHA 必须存在且 base 为 head 祖先；fork PR 只用只读 token |
| `push` to `master` | `github.sha` | `github.event.before` | before 为全零、缺失或非祖先时失败 |
| `workflow_dispatch` | 当前 ref SHA | 必填 input `base_sha` | input 必须是存在的祖先 commit |

- `actions/checkout@v4` 使用 `fetch-depth: 0`；无法建立可信 diff 时 mutation job 失败，不降级为跳过。
- 所有 jobs 在所有事件运行；mutation 无适用目标时运行固定 smoke，因此 required checks 不会因 `if`/`needs` 静默 skipped。
- Actions 固定到声明的 major，Go `1.25.3`、Node `20`、npm 使用 lockfile、Gremlins `v0.6.0`；每个 job 有 timeout，workflow concurrency 对同一 ref 取消旧运行，取消不会产生成功结论。
- CI artifacts `when: always` 上传 unit JSON、integration logs、mutation JSON/summary；权限仅 `contents: read`。
- workflow 静态测试读取 YAML，断言 triggers、fetch-depth、六个 job、Make target、timeout、权限和无 path filter/allow-failure。

### 8.6 Traceability contract

`docs/specs/traceability.json` 记录规格 ID、Case ID、测试全名和文件。`scripts/quality/verify-traceability.py` 校验：ID 唯一、规格文档存在、Case 在 `testcases.md` 存在、测试文件存在且包含完全匹配的 ID、当前首批规格均至少有一个自动化测试。未知/重复/缺失项失败。Code Review 的质量证据引用该 manifest 和实际报告。

## 9. Implementation refinements

实现阶段根据锁定工具和真实运行结果补充以下不改变产品契约的细节：

- Gremlins `v0.6.0` 的实际 mutant coverage 参数是 `--threshold-mcover`；`go install` 构建的 CLI 版本字符串为 `dev`，因此通过 `go version -m` 校验模块版本必须为 `v0.6.0`。
- 集成测试仍由宿主发现 backend 随机端口并执行 `415` readiness；带 `integration` tag 的 Go Tester 在同一隔离 Compose 网络内运行，因此无需暴露 PostgreSQL/Redis 端口，并能直接验证 Redis `online:users` 的上线/离线转换。
- Weak-test proof 使用 manifest 锁定 strong/weak SHA、保护性测试清单和 checked-in unified patch；patch 只在临时目录应用，强/弱普通测试都为绿，而 mutation verifier 必须拒绝 weak report。
- 本地未提供 `MUTATION_BASE_SHA` 时执行固定 JWT/UID smoke，以覆盖未提交改动；CI 必须显式提供可信祖先 SHA，缺失或非祖先直接失败。
