# 项目现状临时记录

更新时间：2026-03-13

## 0. 当前版本口径
- 1.0：当前基础功能 + 拉黑系统
- 2.0：完整社群系统
- 3.0：动态 / 朋友圈

其余阶段划分与执行状态保持不变。

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
- 前端：`npm test` 通过（6 files / 15 tests）
- 后端：`go test ./...` 通过（全绿）

## 4. 当前已知问题 / 待修项
- 后续阶段能力仍待完成：
  - 社群系统持久化与权限体系
  - 动态 / 朋友圈
- 当前正在实现中：
  - 拉黑系统 V1（数据模型、关系管理、聊天拦截、前端关系页）
- M0 收口剩余动作：
  - 按 `docs/plans/M0_MANUAL_QA.md` 完成一次双账号手工回归并记录结果
  - 打可回滚标签（建议 `m0-stable`）

## 5. 本轮已修复
1. 修复 Auth 测试与 UID 生成兼容：
   - sqlite 不再依赖 `nextval('user_uid_seq')`
   - 修复 `GetCurrentUser` 测试中的 UID/ID 混用
2. 修复 Proto 测试基线：
   - 测试读取路径切换为 `proto/*.proto`
   - 断言更新到当前协议命名
3. 修复 WebSocket 握手测试：
   - 对齐缺 token / invalid token 的 HTTP 401 行为
4. 补充防回退测试：
   - 创建会话后 `conversation_push` 回归测试
   - 私聊“未互关最多 3 条”服务层测试
   - 拉黑预留 `Skip` 占位测试（M1 转为可执行断言）
5. 补齐 M0 文档工件：
   - `docs/plans/M0_FAILURE_MATRIX.md`
   - `docs/plans/M0_MANUAL_QA.md`

## 6. 建议下一步
1. 执行并记录一轮 `M0_MANUAL_QA.md` 的双账号回归结果。
2. 完成 M0 收口标签（`m0-stable`）并准备进入 M1。
