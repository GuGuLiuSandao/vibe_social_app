# 项目健康检查与精简 V1 — 技术方案文档

## 1. 里程碑映射

- Roadmap：M1「聊天 MVP 完整化」后的工程收口，以及 M3「发布准备」的前置健康检查。
- 执行清单：`docs/plans/M0_TASKS.md` 的 T10「项目健康检查与仓库精简」。

## 2. 审查方法

- 使用 `git status/branch/log/ls-files` 确认版本状态和跟踪范围。
- 使用 `rg` 检查命令、组件、样式和文档引用。
- 使用文件行数和体积排序定位维护热点。
- 使用 Go 测试、覆盖率、`go vet`、Vitest 和 Vite build 验证工程基线。
- 对 Docker 运行态单独记录；Docker daemon 不可用时不将静态检查等同于联调通过。

## 3. 删除策略

### 3.1 后端命令

保留：

- `backend/cmd/api`：正式服务入口。
- `backend/cmd/seed_social_data`：可配置、可重复执行的数据构造工具。

删除：

- `check_seq`、`fix_seq`、`test_db_check`：一次性数据库序列排查工具。
- `test_register`、`test_uid`：硬编码本地地址的临时接口请求工具，已有自动化测试覆盖。
- `test_chat_client`、`test_chat_verify`：硬编码或半自动聊天验证程序，未被当前 QA 文档引用，且不能替代双账号手工回归。

这些命令均不被 Makefile、README、CI、功能文档或其他源码引用。

### 3.2 前端与产物

- 删除仅自引用的 `components/ui/separator.jsx`。
- 删除确认无 JSX/JS 引用的旧 Discord/theme 辅助样式；保留 Tailwind 可能使用的变量和通用规则。
- 删除 Git 跟踪的 0 字节 `test_register.pb`。

### 3.3 文档

- 删除根目录 `TEMP_STATUS.md`。
- 将实际功能状态、验证状态、未推送提交和剩余风险更新到 `ROADMAP.md`。
- 更新 `docs/README.md`、README 与任务清单中的状态入口和清理说明。

### 3.4 WebSocket 连接管理修复

- 将连接注册、按实例注销和连接快照提取为带锁的小方法。
- 注销时同时比较用户 ID 和 `Client` 实例；旧连接迟到的注销事件只清理自身，不改变当前连接映射或在线状态。
- 新连接替换旧连接时关闭旧 socket，使旧读写循环退出。
- 广播先在读锁下取得连接快照，再发送；慢客户端通过写锁注销，避免在 `RLock` 范围内 `delete`。
- 不再通过关闭 `Send` channel 驱动退出，避免与锁外发送形成 send-on-closed-channel 竞态；统一通过关闭 socket 结束读写循环。
- 增加“旧连接注销不影响新连接”和“当前连接正常移除”回归测试。

### 3.5 前端连接配置修复

- `frontend/src/lib/ws.js` 与 HTTP 客户端保持一致，从 `import.meta.env.VITE_WS_BASE` 读取地址。
- 未配置时继续回退到 `ws://localhost:8080/ws`，不改变本地开发行为。

## 4. 不在本次直接修改的问题

- `frontend/src/pages/Chat.jsx` 约 2458 行、状态与协议处理高度集中。
- `backend/internal/service/chat_service.go` 约 2076 行，同时承载 NPC、私聊、群管理和消息查询。
- 官方话题房仍为内存态，与持久化群模型并存。
- CORS、默认开发密钥、白名单免 token 等开发便利配置不满足生产安全要求。
- 缺少社群完整 WS/页面集成测试、双账号手工回归和长连接稳定性验证。

以上内容需要独立需求、模块边界设计或真实运行环境，不与低风险清理混合实施。

## 5. 验证方案

1. `cd backend && gofmt` 检查剩余 Go 源码。
2. `cd backend && go test ./...` 和 `go test -race ./internal/websocket`。
3. `cd backend && go vet ./...`。
4. `cd frontend && npm test`。
5. `cd frontend && npm run build`。
6. `rg` 复查被删除名称与 `TEMP_STATUS.md` 无残留引用。
7. `git diff --check` 与工作区审查。

## 6. 回滚

- 本次删除均由 Git 跟踪，可通过对应提交恢复。
- 不涉及数据迁移、协议变更或外部状态变更。

## 7. 提交追踪

- 待最终提交完成后补充提交哈希与消息。
