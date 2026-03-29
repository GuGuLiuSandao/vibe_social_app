# Code Review 记录

## 1. Review 范围
- 社群管理 V1.1 首轮实现
- 范围：数据模型、Proto 协议、后端群管理服务、WS 路由、前端 Chat 群管理界面

## 2. 发现的问题
- [x] 现有群聊能力与官方群话题房能力割裂，若直接重做 community 域，改动面过大。
- [x] `Chat.jsx` 单文件已较大，若一次性拆完整社群页面，交付风险较高。
- [x] 群主/管理员权限边界复杂，若在 service 层之外分散校验，容易出现越权。
- [x] 群解散后“不可查看历史消息”会影响现有消息读取链路，需要在原消息接口统一拦截。

## 3. 处理结果
- 采用“基于现有 `conversations` / `conversation_participants` 演进”的方案，没有新开独立 community 主表。
- 已扩展 proto、后端模型、群管理 service、WS handler，并生成 Go/TS 协议代码。
- 已在前端 Chat 页面补充群管理交互：群资料、群公告、成员列表、邀请处理、申请审批、管理员任免、群主转让、退群/解散群等入口。
- 已进一步将群管理 UI 从 `Chat.jsx` 中拆分出独立组件：`GroupProfileCard`、`GroupAnnouncementCard`、`GroupMembersCard`、`GroupJoinRequestsCard`、`GroupCreateModal`。
- 权限控制统一收敛在 `backend/internal/service/chat_service.go`，由 service 层负责 owner/admin/member 边界判断。
- 已验证：
  - `cd backend && go test ./...`
  - `cd backend && go build ./...`
  - `cd frontend && npm test`
  - `cd frontend && npm run build`

## 4. 遗留事项
- 前端当前仍以 `Chat.jsx` 为主承载，后续建议拆分群详情、群成员、群申请、群邀请子组件。
- 当前前端虽然已补全主要交互入口，但仍缺少更细致的状态反馈（loading/confirm/modal/toast）与更清晰的权限引导。
- 当前官方群话题房 `topic_room` 仍为内存态，与新群管理模型并存，后续需要统一到同一套持久化社群体系。
- 已补充一批群管理 V1.1 自动化测试，但当前仍以 service 层测试和组件级前端测试为主，后续可继续补充更完整的 WS/页面集成测试。
