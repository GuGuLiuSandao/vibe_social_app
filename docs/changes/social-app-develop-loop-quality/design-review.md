# Design Review: social-app-develop-loop-quality

## 1. Review Result

- Reviewer role: Independent Design Reviewer
- Score: **72/100**
- Pass threshold: **80/100**
- Result: **FAIL**
- Blockers: **4**

设计方向与 Requirement 基本一致，覆盖 unit、frontend、integration、mutation 和 CI，也明确提出零测试、工具错误和不可信 diff 必须 fail closed。但多个关键门禁目前只有目标描述，没有可执行的判定协议；Implementer 可以做出形式上符合设计、实际仍可能空跑或假绿的实现，因此暂不放行进入 Test Design/Coding。

## 2. Blockers

### B1 — 普通测试“非空执行”缺少统一、可实现的判定协议

影响：DLQ-002、DLQ-003、DLQ-007、DLQ-009。

`go test ./...` 在没有测试时可以成功，Vitest 的退出行为也取决于版本和参数。设计仅声明“零测试失败”，没有定义如何收集、解析和校验测试数量，也没有规定 package/file 粒度和机器可读输出。实现者无法据此唯一实现 fail-closed 门禁。

必须补充：

- 后端入口的确切命令、结构化输出格式，以及“至少多少个测试、哪些目标 package 必须有测试”的计数规则；解析失败必须失败。
- 前端入口的确切 Vitest 参数、机器可读报告及最小 test-file/test-case 数；报告缺失、空文件、schema 不符必须失败。
- wrapper 自测至少覆盖：零测试、测试进程失败、报告缺失、损坏报告、测试被过滤为零，且逐项断言非零退出。
- Make target 必须传播底层命令退出码，不得用日志文本或 shell pipeline 意外吞掉失败。

### B2 — Mutation fail-closed 判定与工具契约未闭合

影响：DLQ-005、DLQ-006、DLQ-007。

设计没有固定 Gremlins/Stryker 的具体版本、调用参数、报告 schema、允许状态集合、mutant 总数下限及超时/进程信号处理。`100% break threshold` 本身不足以证明 `noCoverage`、`error`、空报告或零 mutant 都失败；后端“diff 文件映射唯一 package”也会扩大到整包 mutation，却没有说明如何验证实际被 mutation 的目标与计划目标一致。

此外，“无生产 diff 时按 Requirement 审计后跳过”与“另设 mutation-smoke 固定跑 JWT/UID”的触发关系不清：它是每个 PR 的必需门禁、仅基础设施自测，还是首次落地证据，当前无法判断。

必须补充：

- 两个工具的精确锁定版本、安装校验、命令行和报告 schema/version。
- 明确状态白名单与算法：baseline 成功；目标发现成功；有目标时 `total > 0`；`survived/noCoverage/error/timeout/unknown > 0` 均失败；报告缺失、空、不可解析、未知字段/状态、进程异常均失败。
- 输出“计划目标”和“工具实际处理目标”，两者不一致即失败；明确 Go package 扩张后的可接受边界。
- 明确 diff 为零、仅不适用文件、目标被全部排除三种情况的不同结论和审计字段；规定 smoke 的唯一触发点以及是否为 required check。
- wrapper 负向测试覆盖所有失败分支，而不仅是 survivor 和零测试摘要。

### B3 — Integration 门禁缺少完整生命周期，无法证明隔离、可清理和失败诊断

影响：DLQ-004、DLQ-007、DLQ-009。

设计列出了 Compose、真实依赖、readiness 和 `down -v`，但没有规定数据库 schema/migration 初始化、后端配置注入、端口发现、测试命令、资源唯一命名、并发运行、日志采集顺序和中断清理。随机 Compose project 与“独立宿主端口”并不能自动避免并行冲突。若启动中途失败或进程被取消，当前设计也不能保证诊断先保存、资源后清理。

必须补充：

- 给出从 build/up、健康检查、schema 初始化、执行带 tag 测试、采集日志到 `down --volumes --remove-orphans` 的确切状态机。
- 所有退出路径（启动失败、测试失败、timeout、INT、TERM）都必须执行清理；先采集诊断再清理，并保留原始失败退出码。
- 使用动态宿主端口或明确证明不暴露宿主端口；将实际 endpoint 可靠传给测试，支持本地/CI 并发。
- readiness 必须验证 PostgreSQL、Redis 和可用的后端 API，而不只是容器 running；规定总超时、单次超时和最终诊断。
- 明确测试数据库 schema 的创建方式、专用凭据、volume/network/project 唯一性，并增加“预置同名资源/并发两次运行”隔离测试。

### B4 — CI 的适用性矩阵和可信 diff 来源不完整，可能漏跑或错误跳过

影响：DLQ-007、DLQ-009。

设计写了 PR 使用 base SHA、push 使用父提交，但没有定义 workflow triggers、事件级 SHA/HEAD 语义、merge commit、首次 push/root commit、force-push、fork PR、重跑旧 workflow、路径过滤和 job 条件。也未明确 backend/frontend/integration/mutation/build/vet/config 各自在什么变更下必须执行。GitHub job 被 `if` 或 path filter 跳过时，Requirement 要求的“四层门禁均执行”可能被静默绕过。

必须补充：

- 用表格定义 PR/push/manual 各事件的 checkout ref、HEAD SHA、base SHA 获取方式、祖先校验和失败行为；无法建立可信 diff 时必须失败，不得退化为跳过。
- 定义每类文件变更对应的 required jobs；若目标是所有 PR 均跑四层，删除模糊的“适用时”。若允许跳过，必须由一个始终运行的判定 job 输出机器可读理由，并使未知路径 fail closed。
- 明确 shallow clone、base 不存在、root commit、重命名/删除、submodule、生成文件和 merge queue 的处理。
- CI 直接调用与本地相同的 Make targets；增加静态/自动测试证明 workflow 未复制逻辑且所有必需 job 依赖关系不会因 `needs`/`if` 被意外跳过。
- 固定 Actions 与工具供应链版本，并规定超时、并发取消及 artifact/日志保留；取消或超时不能形成成功结论。

## 3. Non-blocking Findings

### N1 — Project Profile 与本设计尚未形成最终命令契约

`.engineering-loop/project.md` 仍以“Technical Design 和 Test Cases 声明的回归验证”作为 Behavior change 入口，未列出本设计计划新增的 backend/frontend/integration/mutation/quality targets。Task 1 表示会更新 Profile，但设计应直接给出更新后的命令矩阵和 `make quality` 的包含关系，避免本地与 CI 对“统一入口”理解不同。

### N2 — build/vet/config 与四层测试的聚合关系不明确

Task 6 提到 build/vet/config，Behavior Changes 的 PR workflow 只称“四层 Check”，没有说明 `make quality` 是否包含 `git diff --check`、Go build/vet、frontend build、Compose config、unit、integration 和 mutation。建议定义唯一 DAG，并规定任一子门禁失败时聚合目标非零退出。

### N3 — 规格追溯的机器校验不足

Test Mapping 给出了 ID 到目标文件的计划，但没有定义测试名称 ID 格式、重复/不存在 ID、Case 与测试多对多关系如何校验。建议提供 traceability manifest 或检查脚本，并对未知 ID、缺失 Case、缺失测试文件、缺失命令证据 fail closed。

### N4 — “等价 mutant”处理仍有裁量泄漏

“精确、带理由的项目映射/排除”没有规定谁批准、存放位置、有效期及如何防止宽泛 glob。建议排除项逐 mutant/逐源码位置记录，独立评审，禁止临时 CI 参数豁免；排除清单解析失败或命中范围扩大时失败。

### N5 — 安装源与项目副本一致性缺少判定细节

风险项要求同步 `engineering-loop` 源并用 installer `--update` 校验，但没有定义比较范围、允许差异及失败输出。应规定可重复的检查命令和字节级/语义级一致性规则，否则 Develop Loop 核心仍可能漂移。

## 4. Score Breakdown

| Dimension | Weight | Score | Review |
|---|---:|---:|---|
| Requirement coverage and traceability | 20 | 17 | 主体覆盖完整，命令与证据追溯尚未闭合 |
| Unit/backend executability | 15 | 10 | 有目标和测试映射，缺少零测试机器判定 |
| Frontend executability | 15 | 11 | 技术选择合理，报告与空跑判定不明确 |
| Integration isolation/executability | 15 | 9 | 架构正确，生命周期和并发隔离不足 |
| Mutation effectiveness/fail-closed | 20 | 12 | 风险意识强，工具结果协议和目标一致性不足 |
| CI/local parity and fail-closed | 15 | 13 | 同构方向明确，事件矩阵和跳过语义缺失 |
| **Total** | **100** | **72** | **FAIL** |

## 5. Conditions for Re-review

修订 Design 后重新评审；至少应新增以下可核验内容：

1. 每个 Make target 的精确命令、输入、结构化输出、最小执行数量和退出码契约。
2. Integration 生命周期状态机及异常/取消清理测试。
3. Backend/Frontend mutation 报告 schema、状态判定伪代码、目标一致性校验和完整负向测试表。
4. CI 事件 × 变更类型 × required job 矩阵，以及可信 base/HEAD 获取与失败规则。
5. 更新后的 Project Profile 命令矩阵和唯一 `quality` 聚合 DAG。

上述 4 个 blocker 全部关闭且复评分达到 80 分后方可通过。

## 6. Re-review

- Reviewer role: Independent Design Reviewer
- Re-review scope: 首轮 B1–B4
- Score: **91/100**
- Pass threshold: **80/100**
- Result: **PASS**
- Remaining blockers: **0**

复审结论：修订后的 Design 新增 8.1–8.6 可执行门禁协议，已将首轮四个 blocker 全部关闭。设计现在明确了结构化报告、最低执行数量、退出码传播、集成测试生命周期、Mutation 工具及状态判定、可信 diff、CI 事件矩阵和本地质量 DAG；实现者可以据此形成确定且 fail-closed 的实现。允许进入后续 Test Design，但实现与评审仍须逐项验证这些协议确实落地。

### B1 Re-review — CLOSED

8.1 给出了后端与前端的确切入口、机器可读报告、必需 package/test file、最低 suite/test 数以及缺失、空、损坏和 schema 异常的失败语义；同时要求保留底层退出码，并规定 verifier fixture 与 wrapper 负向测试。原“非空执行无法唯一判定”的问题已关闭。

### B2 Re-review — CLOSED

8.3 固定 Gremlins `v0.6.0`、Stryker `9.6.1` 和 Vitest `4.1.10`，定义 baseline、可信目标发现、实际目标边界、`total > 0`、状态白名单、报告异常及工具异常的失败规则。无 diff、仅不适用和 deleted-only 均运行固定 smoke，required check 不再因零目标假绿；负向 fixture 覆盖各失败状态、未知状态、计数不变量、越界目标和工具异常。原 Mutation 契约不闭合问题已关闭。

### B3 Re-review — CLOSED

8.2 给出了 build、up、动态端口发现、API readiness、带 tag 测试、报告验证、诊断采集和 `down --volumes --remove-orphans` 的完整状态机。专用凭据、project-scoped network/volume、随机 project、动态 backend 端口和不暴露 PostgreSQL/Redis 端口形成隔离；EXIT/INT/TERM、启动/测试失败均先保存诊断再清理，并有 fake compose 和并行 smoke 验证。原生命周期、并发隔离与清理诊断问题已关闭。

### B4 Re-review — CLOSED

8.5 明确 `pull_request`、master `push`、`workflow_dispatch` 的 checkout HEAD、mutation base、祖先校验与失败行为；所有事件无 path filter 地运行六个 jobs，无适用 mutation 目标时运行 smoke，无法建立可信 diff 时失败。`fetch-depth: 0`、只读权限、timeout、concurrency、always-upload artifacts 和 workflow 静态测试也已定义。8.3 另覆盖重命名、删除、submodule、生成文件和路径越界。原 CI 漏跑或错误跳过风险已关闭。

## 7. Re-review Score Breakdown

| Dimension | Weight | Score | Re-review |
|---|---:|---:|---|
| Requirement coverage and traceability | 20 | 19 | 8.6 建立机器可校验的规格、Case、测试文件映射 |
| Unit/backend executability | 15 | 14 | 命令、报告和最低执行数量已明确 |
| Frontend executability | 15 | 14 | Vitest 报告、suite/file 校验和失败语义已明确 |
| Integration isolation/executability | 15 | 13 | 生命周期和并行隔离闭合；仍需实现阶段验证清理失败组合语义 |
| Mutation effectiveness/fail-closed | 20 | 18 | 工具、目标、状态、smoke 与负向测试契约完整 |
| CI/local parity and fail-closed | 15 | 13 | 事件矩阵与统一 target 已闭合；merge queue 未作为独立事件声明 |
| **Total** | **100** | **91** | **PASS** |

## 8. Remaining Non-blocking Findings

- R1：Integration wrapper 同时出现“最后返回最初失败码”和“清理失败不得形成成功”。实现时应明确：主流程成功而清理失败必须非零；主流程已失败时保留主失败码并记录清理失败，避免实现者对组合失败优先级产生歧义。
- R2：CI 矩阵没有声明 `merge_group`。当前 Requirement 只要求 PR CI，因此不构成 blocker；若仓库启用 GitHub merge queue，应在实现前增加 `merge_group` trigger 及可信 base/HEAD 规则，或明确该仓库不启用 merge queue。
- R3：8.3 要求报告 schema/类型校验，但未给 Gremlins 与 mutation-testing JSON 的正式 schema 标识。实现时 verifier 应锁定实际工具版本产出的字段集合与必需字段类型，并以 golden fixtures 防止版本漂移。
