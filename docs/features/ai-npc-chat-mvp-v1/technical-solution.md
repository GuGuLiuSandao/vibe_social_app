# 技术方案文档

## 1. 需求关联
- 对应目录：`docs/features/ai-npc-chat-mvp-v1/`
- 对应需求文档：`requirement.md`
- 里程碑映射：`docs/plans/ROADMAP.md` 的 M1「聊天 MVP 完整化」扩展项（新增 AI NPC 聊天类别）
- 清单映射：`docs/plans/M0_TASKS.md` 新增 T9（AI NPC 聊天 MVP）

## 2. 目标与非目标
- 目标：
  - 新增会话类型 `NPC`，与现有聊天类别并列。
  - 提供默认 NPC（魔兽酒馆老板）1v1 会话能力。
  - 用户发送消息后由服务端自动生成 NPC 回复并写入消息流。
  - 实现跨 Session 记忆持久化。
- 非目标：
  - 不实现用户自定义 NPC 人设。
  - 不实现 NPC 邀请最多 4 人群聊。
  - 不接入外部 LLM 平台。

## 3. 方案概述
1. 协议层：在 `ConversationType` 增加 `CONVERSATION_TYPE_NPC = 3`。
2. 数据层：新增 `npc_memories` 表，按 `(user_id, npc_key)` 唯一存储记忆摘要与偏好。
3. 会话层：`CreateConversation(type=NPC)` 返回用户唯一默认 NPC 会话（存在即复用）。
4. 消息层：用户向 NPC 会话发送消息后，后端同步触发 NPC 回答：
   - 读取记忆 + 最近消息上下文
   - 生成酒馆老板风格回复
   - 持久化为 NPC 发送消息
5. 推送层：在原有 `message_push` 通道向用户补发 NPC 回复，无需新增 WS 事件类型。
6. 前端层：聊天列表新增“AI NPC”分组和“+ AI NPC”按钮，展示/进入 NPC 会话。

## 4. 影响范围
- Proto:
  - `proto/chat/chat.proto`
  - 生成文件：`backend/internal/proto/chat/chat.pb.go`、`frontend/src/proto/chat/chat_pb.ts`
- Backend:
  - `backend/internal/models/chat.go`
  - `backend/internal/db/database.go`
  - `backend/internal/service/chat_service.go`
  - `backend/internal/service/chat_service_test.go`
  - `backend/internal/websocket/handler.go`
- Frontend:
  - `frontend/src/pages/Chat.jsx`
- Docs/Test:
  - `docs/features/ai-npc-chat-mvp-v1/*`
  - `docs/plans/ROADMAP.md`
  - `docs/plans/M0_TASKS.md`

## 5. 数据结构 / 协议变更
- `ConversationType` 新增：`CONVERSATION_TYPE_NPC = 3`。
- 新增模型 `NPCMemory`：
  - `id`
  - `user_id`
  - `npc_key`（MVP 固定为 `wow_tavern_keeper`）
  - `memory_json`（json string）
  - `summary`
  - `updated_at`
- 会话创建约束：
  - `NPC` 类型会话固定成员为 `[user, npc_system_user]`
  - 每用户仅允许一个默认 NPC 会话（通过查询复用）

## 6. 实现步骤
1. 先修改 proto 枚举并生成 Go/TS 代码。
2. 扩展后端模型与迁移，新增 `NPCMemory` 表。
3. 在 `ChatService` 中实现：
   - 默认 NPC 用户自动创建（缺失时自动补齐）
   - NPC 会话创建/复用
   - NPC 记忆读写
   - NPC 回复生成与落库
4. 在 WS handler 里接入“用户发言后自动推送 NPC 回复”。
5. 在前端 `Chat.jsx` 增加 NPC 分类与快速入口。
6. 增加/更新单测并执行全量测试。
7. 回填 code review 记录。

## 7. 测试方案
- 自动化：
  - `cd backend && go test ./...`
  - `cd frontend && npm test`
- 新增重点断言：
  - `CreateConversation(type=NPC)` 可重复调用且复用同一会话
  - NPC 会话发送消息后会自动产生 NPC 回复
  - NPC 记忆在服务重建后仍可读取

## 8. 风险与回滚
- 风险：
  - 规则式回复可读性有限，可能被感知为“模板化”。
  - 若消息发送路径异常，可能导致用户消息成功但 NPC 回复失败。
- 回滚：
  - 回退 `ConversationType=NPC` 入口按钮与后端 NPC 自动回复逻辑即可；不影响原有私聊/群聊/话题房。

## 9. Commit 记录
- Commit Hash:
- Commit Message:
