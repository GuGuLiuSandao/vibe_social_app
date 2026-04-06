# Code Review 记录

## 1. Review 范围
- AI NPC 聊天 MVP（默认魔兽酒馆老板 + 1v1 + 跨 Session 记忆）
- 范围：
  - `proto/chat/chat.proto`
  - `backend/internal/models/chat.go`
  - `backend/internal/db/database.go`
  - `backend/internal/service/chat_service.go`
  - `backend/internal/websocket/handler.go`
  - `frontend/src/pages/Chat.jsx`
  - 对应测试与计划文档

## 2. 发现的问题
- [x] 现有会话类型仅支持私聊/群聊，无法表达 AI NPC 独立类别。
- [x] 会话创建流程依赖 `participant_ids`，无法支持“默认 NPC 一键建会话”。
- [x] 聊天消息流仅支持用户->会话，缺少“服务端自动回话”能力。
- [x] 无 NPC 持久化记忆存储，重登后无法稳定复用用户画像。
- [x] 前端会话分组只有“群聊/私聊”，缺少 AI NPC 分类入口。

## 3. 处理结果
- 协议层新增 `CONVERSATION_TYPE_NPC = 3`，并完成 Go/TS 代码生成。
- 后端新增 `NPCMemory` 模型与迁移，按 `(user_id, npc_key)` 持久化记忆。
- `CreateConversation` 新增 NPC 分支：
  - 自动补齐默认 NPC 系统账号
  - 每用户复用唯一默认 NPC 会话
- 新增 `GenerateNPCReply`：
  - 解析并更新用户记忆（称呼/偏好/主题）
  - 生成酒馆老板风格回复并写入消息表
  - 通过 WS `message_push` 回推给当前用户
- 前端 `Chat.jsx` 新增：
  - `AI NPC` 会话分组
  - `+ AI NPC` 快捷入口
  - NPC 会话 badge 与文案适配
- 测试验证通过：
  - `cd backend && go test ./...`
  - `cd frontend && npm test`

## 4. 遗留事项
- 当前回复策略为规则式引擎，后续可切换可配置模型路由。
- 长期记忆目前只做“追加 + 去重”，尚未实现记忆衰减与用户可视化管理。
- 最终形态中的“自定义人设 + NPC 邀请最多 4 人群聊”未纳入本次 MVP。
