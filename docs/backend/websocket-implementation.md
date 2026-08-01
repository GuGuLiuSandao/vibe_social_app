# WebSocket 模块实现说明（当前代码）

本文档基于当前 `backend/internal/websocket` 实现整理，描述真实行为，而非设计稿。

## 1. 架构概览

WebSocket 模块采用 Hub-Client 模型：
- `Handler`：HTTP 入口，负责握手与鉴权
- `Server`：管理在线连接与消息分发
- `Client`：单个连接实体，分别启动读写协程

核心文件：
- `backend/internal/websocket/handler.go`
- `backend/internal/websocket/manager.go`

## 2. 握手与鉴权

### 2.1 入口地址
- 路径：`GET /ws`
- 鉴权：必须提供 JWT token
  - 优先读取 query 参数 `token`
  - 若 query 无 token，则尝试 header `Sec-WebSocket-Protocol`

### 2.2 鉴权逻辑
1. token 为空：返回 `401 Unauthorized`
2. token 非法或过期：返回 `401 Unauthorized`
3. 校验通过：使用 token 内 `claims.UserID` 作为连接用户 ID

注意：
- 当前握手阶段不校验 `uid` 参数
- 当前握手失败仅返回 HTTP 状态码（非 Protobuf 错误体）

## 3. 连接生命周期

### 3.1 ReadPump
- `SetReadLimit(8 MiB)`，用于容纳当前允许的 WebSocket 头像上传帧
- `SetReadDeadline(pongWait)`，默认 `60s`
- 收到 Pong 时刷新读超时
- 收到 Ping 时返回 Pong 控制帧
- 消息读取后按 Protobuf `WsMessage` 解码并路由

### 3.2 WritePump
- 从 `Client.Send` channel 读取下行消息并写回连接（二进制帧）
- 每 `pingPeriod`（`54s`）主动发送 Ping
- 写超时 `writeWait`（`10s`）

## 4. 消息协议与路由

统一信封：`proto/ws.proto` 中的 `WsMessage`。

目前路由的业务类型：
- 基础：应用层 `PING` / `PONG`
- Account：连接内认证、用户搜索、资料更新、头像上传
- Relation：follow、unfollow、following、followers、friends、block、unblock、blocked list
- Chat：发送消息、会话列表、消息列表、已读和创建会话
- Group：群详情、资料与公告、成员与角色、邀请、入群申请审批、群主转让、移除成员、退群和解散
- Topic room：房间列表、加入、离开、成员列表和消息

说明：
- 业务消息采用二进制 Protobuf 协议

## 5. 在线状态与 Redis

`manager.go` 在连接注册/注销时会调用 Redis：
- 上线：`SetUserOnline(userID)`
- 下线：`SetUserOffline(userID)`

并提供：
- `SendToUser(userID, message)`
- `SendToConversation(conversationID, message)`（基于 Redis 订阅集合）

## 6. 关系推送与会话推送

- follow / unfollow 后，向目标用户推送 `WS_TYPE_RELATION_PUSH`
- 创建会话后，向其他参与者推送 `WS_TYPE_CHAT_CONVERSATION_PUSH`
- 普通消息、群资料和群成员变化会向相关在线用户发送对应 push

## 7. 参数与默认值

| 参数 | 默认值 | 说明 |
| :--- | :--- | :--- |
| `writeWait` | 10s | 单次写操作超时 |
| `pongWait` | 60s | 读取超时窗口 |
| `pingPeriod` | 54s | 服务端主动心跳间隔 |
| `maxWSMessageSize` | 8 MiB | 单个入站 WebSocket 帧上限 |
| `ReadBufferSize` | 1024 | Gorilla Upgrader 读缓冲 |
| `WriteBufferSize` | 1024 | Gorilla Upgrader 写缓冲 |

## 8. 排查建议
- 401 握手失败：先检查 token 是否传入、是否过期
- 连接建立后无消息：检查客户端是否发送二进制 Protobuf
- 关系推送未收到：确认目标用户在线（Redis 在线状态 + Server.Clients）
