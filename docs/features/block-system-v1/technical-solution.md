# 技术方案文档

## 1. 需求关联
- 对应目录：`docs/features/block-system-v1/`
- 对应需求文档：`requirement.md`
- 对应里程碑：`docs/plans/ROADMAP.md` 中的 1.0 / M1 黑名单系统 V1

## 2. 目标与非目标
- 目标：
  - 建立可持久化的黑名单数据模型
  - 在 WebSocket 关系链中支持 block/unblock/get-blocked
  - 在聊天服务层阻断私聊创建、私聊发消息、群聊邀请
  - 在前端关系页增加黑名单管理入口
- 非目标：
  - 不处理群消息隔离
  - 不新增 HTTP API
  - 不实现被拉黑来源展示

## 3. 方案概述
- 数据层新增 `BlockRelation` 表，记录 `user_id -> target_id` 的拉黑关系。
- 关系服务新增：拉黑、取消拉黑、获取黑名单、双向拉黑检查。
- 拉黑成功时自动删除双方 follow 关系。
- 聊天服务在创建私聊、发送私聊消息、创建群聊时增加双向拉黑拦截。
- WebSocket 扩展 relation 模块协议与处理逻辑。
- 前端在 Chat 页的关系面板增加“Blocked”分页及操作按钮。

## 4. 影响范围
- Backend:
  - `backend/internal/models/relation.go`
  - `backend/internal/db/database.go`
  - `backend/internal/service/relation_service.go`
  - `backend/internal/service/chat_service.go`
  - `backend/internal/websocket/handler.go`
  - 相关测试文件
- Frontend:
  - `frontend/src/lib/ws.js`
  - `frontend/src/lib/ws.test.js`
  - `frontend/src/pages/Chat.jsx`
- Proto:
  - `proto/relation/relation.proto`
  - `proto/ws.proto`
  - 生成代码：`backend/internal/proto`、`frontend/src/proto`
- Docs/Test:
  - `docs/README.md`
  - `docs/features/block-system-v1/*`
  - `docs/plans/ROADMAP.md`
  - `docs/plans/M0_TASKS.md`
  - `TEMP_STATUS.md`

## 5. 数据结构 / 接口 / 协议变更
- 数据表：新增 `block_relations`
  - `user_id`
  - `target_id`
  - `created_at`
- Proto：新增
  - `BlockRequest` / `BlockResponse`
  - `UnblockRequest` / `UnblockResponse`
  - `GetBlockedRequest` / `GetBlockedResponse`
  - `RelationPush` 新增 `TYPE_BLOCKED` / `TYPE_UNBLOCKED`
  - `ws.proto` 新增对应 relation message types

## 6. 实现步骤
1. 扩展 proto 协议并重新生成 Go / TS 代码。
2. 新增黑名单模型与数据库迁移。
3. 扩展关系服务：block/unblock/get-blocked/has-blocking-relation。
4. 在聊天服务中接入黑名单拦截。
5. 扩展 WebSocket relation handlers 与 push。
6. 扩展前端关系页与 WS builders。
7. 补充服务层 / WS helper 测试，更新状态文档与 code review 记录。

## 7. 测试方案
- 自动化测试：
  - `cd backend && go test ./...`
  - `cd frontend && npm test`
- 手工验证：
  - A 拉黑 B 后：
    - A 无法发起与 B 的私聊
    - A/B 在已有私聊中无法继续发消息
    - A 无法邀请 B 建群
    - A 可在黑名单页取消拉黑

## 8. 风险与回滚
- 风险：
  - 黑名单与 follow/friend 列表状态同步不一致
  - 旧私聊仍显示在前端，但发送被拒绝
- 回滚方式：
  - 回退本次代码与 proto 变更
  - 数据层可保留 `block_relations` 表，不影响旧逻辑运行

## 9. Commit 记录
- Commit Hash: `2daa7d0`
- Commit Message: `feat: add blacklist flow and docs workflow`
