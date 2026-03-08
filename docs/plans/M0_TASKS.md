# M0 Task List: 稳定性基线（2026-03-09 ~ 2026-03-22）

## 1. 目标与退出条件

M0 目标是把当前“能跑但不稳定”的状态收敛成“可持续迭代”的工程基线。

退出条件（全部满足才算完成）：
- `cd backend && go test ./...` 全绿
- `cd frontend && npm test` 全绿
- 聊天主链路回归通过：登录 -> 建会话 -> 发消息 -> 已读
- 文档与 CI 命令统一（开发者按文档可直接复现）

## 2. 任务清单（按执行顺序）

## T0 基线冻结与问题清单
- [ ] 重新执行并保存当前测试失败日志（backend/frontend）
- [ ] 产出失败矩阵：失败用例、根因、修复策略、影响范围
- [ ] 建立 M0 跟踪表（本文件勾选 + PR 链接）

验收：
- 失败清单可追踪，不再“边修边猜”

## T1 修复 Auth 测试与 UID 生成兼容性
- [ ] 解决 sqlite 环境 `nextval('user_uid_seq')` 不可用问题（抽象 UID 生成逻辑或按 DB 方言分支）
- [ ] 修复 `TestGetCurrentUserWithProtobuf` 中 UID/ID 使用不一致
- [ ] 补充注册/登录/鉴权关键路径断言，确保与当前实现一致

涉及文件（预期）：
- `backend/internal/auth/handler.go`
- `backend/internal/auth/handler_test.go`
- `backend/internal/db/database.go`（如需）

验收：
- `go test ./internal/auth -v` 全绿

## T2 修复 Proto 测试基线
- [ ] 将测试读取路径从过时 `*.pb` 文本改为当前 `proto/*.proto`
- [ ] 更新过时断言（如旧消息名 `GetConversationsRequest`）到当前协议命名
- [ ] 明确只验证“信封边界 + 关键 payload 引用”，避免脆弱字符串断言

涉及文件：
- `backend/internal/proto_test/ws_proto_test.go`

验收：
- `go test ./internal/proto_test -v` 全绿

## T3 修复 WebSocket 握手测试与实现一致性
- [ ] 对齐缺少 token 的实际行为（当前为 HTTP 401）
- [ ] 修复测试中对 content-type / protobuf body 的错误假设
- [ ] 增加“无效 token”握手失败用例

涉及文件：
- `backend/internal/websocket/handler_test.go`

验收：
- `go test ./internal/websocket -v` 全绿

## T4 回归与防回退测试补强
- [ ] 为“会话创建后 conversation_push”补回归测试（防止再次丢推送）
- [ ] 为“私聊发送限制（未互关最多 3 条）”增加服务层测试
- [ ] 为“拉黑预留点”写失败用例占位（先红灯，M1 转绿）

涉及文件（预期）：
- `backend/internal/websocket/*_test.go`
- `backend/internal/service/*_test.go`（新增）

验收：
- 新增测试在本地稳定通过，且不依赖手工环境

## T5 命令统一与 CI 校准
- [ ] 明确单一验证命令：`go test ./...`、`npm test`
- [ ] 校验 Jenkins `Test` 阶段可复现本地结果
- [ ] 在 `AGENTS.md`/`README.md` 补充“测试基线要求”

验收：
- 本地与 CI 的测试结果一致

## T6 手工联调回归脚本
- [ ] 按“双账号在线互测”补一份最小回归脚本（步骤 + 预期）
- [ ] 覆盖：登录、私聊、群聊、官方话题房、关注关系刷新

建议文档：
- `docs/plans/M0_MANUAL_QA.md`

验收：
- 任意开发者按脚本 20 分钟内完成回归

## T7 M0 收口与发布准备
- [ ] 更新 `TEMP_STATUS.md`（日期、通过用例、剩余风险）
- [ ] 将 M1 输入风险沉淀到 `ROADMAP.md`
- [ ] 打一个可回滚标签（如 `m0-stable`）

验收：
- M0 完成记录可追溯，M1 可直接开工

## 3. 执行节奏建议（单人 + AI）

Week 1：
- 完成 T0/T1/T2/T3（先把红灯清掉）

Week 2：
- 完成 T4/T5/T6/T7（补强与收口）

每日固定节奏：
- 上午修实现与测试
- 下午全量跑测 + 文档更新
