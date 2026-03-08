# 项目现状临时记录

更新时间：2026-03-01

## 1. 运行状态
- Docker Compose 已启动并稳定运行：
  - `db`（PostgreSQL）`Up (healthy)`，端口 `5432`
  - `redis` `Up (healthy)`，端口 `6379`
  - `backend` `Up`，端口 `8080`
  - `frontend` `Up`，端口 `5173`

## 2. 已实现功能模块

### 2.1 账号认证
- 注册（Protobuf）
- 登录（账号密码 / UID 白名单）
- 获取当前用户
- 更新资料

### 2.2 实时聊天（WebSocket + Protobuf）
- WebSocket 握手与 JWT 校验
- 会话列表获取
- 消息列表获取
- 发送消息
- 标记已读
- 创建会话（单聊/群聊基础流程）

### 2.3 关系链
- 关注 / 取关
- 获取关注列表
- 获取粉丝列表
- 获取好友（互关）
- 关系推送（follow/unfollow push）

### 2.4 头像与资料
- HTTP 头像上传接口
- WebSocket 头像上传
- WebSocket 更新个人资料

### 2.5 前端页面
- `/login`：白名单直达、账号登录、注册
- `/chat`：会话、消息、关系、资料编辑、WS 长连接

## 3. 自动化测试现状
- 前端：`npm test` 通过（6 files / 14 tests）
- 后端：`go test ./...` 当前失败（存在实现与测试不一致）

## 4. 当前已知问题 / 待修项
- 后端测试与当前实现有偏差，主要包括：
  - sqlite 测试环境不支持 `nextval(...)`（注册路径依赖 Postgres sequence）
  - WebSocket 握手测试断言与现实现（缺 token 直接 401）不一致
  - proto 测试文件读取路径/文件名与仓库当前结构不匹配

## 5. 本轮已修复
1. 已补齐创建会话后的 `WS_TYPE_CHAT_CONVERSATION_PUSH`：
   - 创建者收到 `create_conversation_response`
   - 其他会话成员收到 `conversation_push`
2. 已统一白名单示例输入为范围内 UID（前端登录页示例改为 `20000000`，与后端范围 `10000000-20000000` 一致）

## 6. 建议下一步
1. 修复后端测试基线（当前测试与实现存在偏差）。
2. 在真实联调场景验证会话创建推送（双账号在线互测）。
