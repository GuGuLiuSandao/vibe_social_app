# 项目健康检查与精简 V1 — Code Review 记录

## 1. Review 范围

- Git 分支、远端差异、目录、依赖、CI/CD 与文档状态。
- 后端认证、上传、数据库初始化、聊天服务、WebSocket 连接管理和官方群话题房。
- 前端入口、存储、HTTP/WS 客户端、聊天页、群管理组件、主题样式和构建配置。
- 本轮删除、WebSocket 修复、前端连接配置和状态文档收敛。

## 2. 发现的问题

### 2.1 本轮已修复

- [x] **P1：旧 WebSocket 连接会误删同用户的新连接。** `Clients` 只按用户 ID 查找，旧连接迟到的 `Unregister` 会删除已经替换进去的新实例，同时错误标记用户离线。
- [x] **P1：广播路径在 `RLock` 下删除 `Clients`。** 这是错误的锁语义，在并发读写时存在数据竞争或运行时异常风险。
- [x] **P1：前端忽略 `VITE_WS_BASE`。** Compose 已注入该变量，但客户端一直硬编码 `ws://localhost:8080/ws`，远程环境会连接访问者自己的 localhost。
- [x] **P2：仓库包含大量一次性调试代码。** 7 个命令共约 635 行，没有 Makefile、CI、文档或源码引用，并与自动化测试/手工 QA 职责重复。
- [x] **P2：状态来源互相矛盾。** `TEMP_STATUS.md` 停留在 2026-03-13，Roadmap 仍称拉黑“实现中”，与实际提交不符。
- [x] **P3：存在空测试产物和未使用前端代码。** 根目录跟踪了 0 字节 `test_register.pb`；`Separator` 组件、旧 Discord/theme 样式只有定义没有调用。

### 2.2 后续重构

- [ ] **P1：聊天前后端职责过度集中。** `frontend/src/pages/Chat.jsx` 约 2458 行、`backend/internal/service/chat_service.go` 约 2076 行；建议分别按连接/会话/关系/群管理/NPC 拆分，并先补集成测试。
- [ ] **P1：官方群与玩家群仍是两套模型。** `topic_room.go` 保存内存消息和成员，服务重启即丢失，且无法满足社群审计要求。
- [ ] **P1：关键边界测试不足。** 当前覆盖率约为 auth 39.3%、service 35.7%、websocket 8.8%；middleware、upload、config、db、redis 等没有直接测试，前端仅 18 个测试。
- [ ] **P2：生产数据与迁移职责耦合。** API 启动时直接执行 GORM `AutoMigrate`，发布时缺少显式、可回滚的版本化迁移。
- [ ] **P2：NPC 系统用户模型需要收敛。** 固定 UID 和普通 User 记录混用，密码字段写入普通字符串；虽然当前 bcrypt 登录不会通过，但应改为明确的不可登录系统账号模型/状态。

### 2.3 发布前阻塞项

- [ ] **P0：开发便利认证不能直接上生产。** UID 白名单可免密码登录；必须增加环境开关并在生产默认关闭。
- [ ] **P0：跨域策略未收口。** HTTP CORS 同时设置 `Allow-Origin: *` 和 credentials，WebSocket `CheckOrigin` 永远返回 true；需要显式允许域名。
- [ ] **P0：默认密钥和数据库口令仍存在。** Compose 和 config 提供公开默认 JWT/DB 密钥，生产启动必须强制校验非默认值。
- [ ] **P1：头像上传只校验扩展名。** 未验证真实 MIME/图片解码，文件名也应使用服务端生成值而非拼接客户端原名。
- [ ] **P1：前端镜像运行 Vite dev server。** 当前 Dockerfile 没有使用已验证的生产构建产物，需要改为静态服务器多阶段镜像。
- [ ] **P1：发布链路重复且证据不足。** GitHub 定时任务每日推镜像但不运行测试，与 Jenkins 推送路径重复；Jenkins 又引用仓库中不存在的 Kubernetes 清单/部署约定，需要确认唯一发布链路。
- [ ] **P1：缺少运行态验收。** Docker daemon 未运行，双账号手工回归、24h 长连接和回滚标签均未完成。
- [ ] **P2：JWT 保存在 localStorage。** 上线前需要结合 CSP、XSS 面和 token 轮换策略重新评估。

## 3. 处理结果

- 后端命令只保留 `cmd/api` 和 `cmd/seed_social_data`。
- 删除 7 个一次性调试命令、空 `test_register.pb`、未使用的 Separator 和 101 行旧样式。
- 删除重复的 `TEMP_STATUS.md`，状态统一到 Roadmap，并更新 README、文档索引和任务清单。
- WebSocket 连接注册/注销改为按 `Client` 实例判断；连接列表访问统一封装在锁内，广播使用快照后再清理慢连接。
- 新连接会关闭被替换的旧 socket；旧连接的迟到注销不会修改新连接和在线状态。
- 新增 3 个连接管理回归测试，race 检查通过。
- 前端 WebSocket 地址改为读取 `VITE_WS_BASE`，未配置时保留本地默认值。
- 清理约 92 MB 本地忽略构建产物；正式后端二进制可通过 `make build-backend` 重建，旧测试客户端可从 Git 历史恢复。
- `simplify` 复查后，移除了冗余消息类型临时变量和失效说明注释，保留清晰的锁方法边界。

验证结果：

- `cd backend && go test ./...`：通过。
- `cd backend && go vet ./...`：通过。
- `cd backend && go test -race ./...`：通过。
- `cd frontend && npm test`：7 files / 18 tests 通过。
- `cd frontend && npm run build`：通过；保留一条 caniuse-lite 数据过期提示，不影响构建。

## 4. 遗留事项

- 优先级建议：先完成生产认证/跨域/密钥收口，再拆 Chat 和 ChatService，随后统一官方群持久化模型。
- 本轮没有创建 `m0-stable` 标签：当前提交包含尚未推送的 AI NPC 变更，且双账号手工回归未执行，不满足稳定标签条件。
- Docker daemon 不可用，未声称服务启动、数据库迁移或真实聊天联调通过。
