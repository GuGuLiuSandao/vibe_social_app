# M0 Failure Matrix（2026-03-08 基线收敛）

## 1. 基线执行记录

- 执行命令：`cd backend && go test ./...`
- 首轮结果：失败（`internal/auth`、`internal/proto_test`、`internal/websocket`）
- 执行命令：`cd frontend && npm test`
- 首轮结果：通过（Vitest 6 files / 15 tests）

## 2. 失败矩阵

| 模块 | 失败用例 | 根因 | 修复策略 | 影响范围 | 状态 |
|---|---|---|---|---|---|
| `internal/auth` | `TestRegisterWithProtobuf` | sqlite 不支持 `nextval('user_uid_seq')` | 增加 `nextUserUID()`：Postgres 走 sequence，非 Postgres 回退 `MAX(uid)+1` | 注册、普通登录补 UID 路径 | 已关闭 |
| `internal/auth` | `TestGetCurrentUserWithProtobuf` | 测试里混用了 `ID` 和 `UID`，请求 `uid` 实际传成 `0` | 测试改为统一使用 `UID` 生成 token 与请求参数 | 鉴权测试稳定性 | 已关闭 |
| `internal/proto_test` | `TestWsProto*` | 仍读取过时 `*.pb` 文本路径（仓库已改为 `proto/*.proto`） | 测试切换读取 `.proto`，并更新断言到当前命名 | 协议回归测试可靠性 | 已关闭 |
| `internal/websocket` | `TestWebSocketHandshakeRejectsMissingUID` | 测试假设握手失败返回 protobuf；实现实际为 HTTP `401` 空体 | 测试对齐实现，并新增“无效 token”失败用例 | WS 握手回归 | 已关闭 |

## 3. 收敛结果

- `cd backend && go test ./...`：通过
- `cd frontend && npm test`：通过
- 后续重点：补齐 M0 手工回归脚本与收口发布动作（见 `M0_TASKS.md`）
