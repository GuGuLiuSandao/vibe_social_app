# 技术方案文档

## 1. 需求关联
- 对应目录：`docs/features/community-management-v1-1/`
- 对应需求文档：`requirement.md`

## 2. 目标与非目标
- 目标：
  - 在当前“群聊会话”能力上，扩展出可长期存在、可管理的自建群模型。
  - 支持群资料管理：头像、简介、公告。
  - 支持三层角色：群主 / 管理员 / 普通成员，并支持管理员任免、群主转让。
  - 支持三种加入模式：私密群、申请加入群、公开群。
  - 支持邀请、申请、审批、退群、移除成员、解散群等完整成员生命周期。
  - 统一官方群与玩家自建群的数据模型，为 2.0 社群系统后续阶段铺路。
- 非目标：
  - 不在本期引入频道/子房间（channels）能力。
  - 不在本期实现封禁、禁言、群黑名单、风控审核。
  - 不在本期实现推荐、搜索排序、社群发现页。
  - 不在本期实现官方后台运营系统。
  - 不在本期拆出完整独立“社群”前端产品线，仍优先在现有 Chat 体系内演进。

## 3. 方案概述
- 当前系统已有两套接近但割裂的能力：
  - 持久化的群聊会话：`conversations + conversation_participants + messages`
  - 内存态的官方群聊天室：`topic_room`
- 本期不新造一套完全独立的“community”域模型，而是采用**“先基于现有 ConversationTypeGroup 演进为 Community-like Group”** 的低风险方案：
  1. 继续以 `conversations` 作为群的主表，避免推翻当前消息链路；
  2. 在 `conversations` 上补充群运营字段（简介、公告、群类型、加入模式、状态等）；
  3. 在 `conversation_participants` 上继续承载成员与角色；
  4. 新增“申请表”“邀请表”来承接入群流程；
  5. 将当前官方群话题房间视为后续并轨对象，本期先完成数据模型兼容，不强行一次性替换全部 topic room 运行时逻辑。
- 这样可以最大化复用：
  - 现有消息持久化、未读、会话列表、消息历史；
  - 现有群聊 UI 与 WebSocket 分发链路；
  - 现有 participant.role 字段。
- 同时把 1.1 的重点放在“群管理”，而不是“社群架构大重写”。

## 4. 影响范围
- Backend:
  - `backend/internal/models/chat.go`：扩展群字段 / 新增申请与邀请模型
  - `backend/internal/service/chat_service.go`：扩展群管理、成员管理、申请审批、邀请处理等服务逻辑
  - `backend/internal/websocket/handler.go`：新增群管理相关 WS 指令
  - `backend/internal/db/database.go`：新增表自动迁移
- Frontend:
  - `frontend/src/pages/Chat.jsx`：新增群详情/设置/成员管理/申请管理/邀请流
  - 视复杂度拆分新组件，如 `components/chat/GroupSettingsPanel.jsx`、`GroupMembersPanel.jsx`
  - 状态管理扩展：群详情、成员列表、申请列表、邀请状态
- Proto:
  - `proto/chat/chat.proto`：扩展 `Conversation` 结构，并新增群管理相关请求/响应
  - `proto/ws.proto`：新增对应消息类型
- Docs/Test:
  - 更新需求、技术方案、代码评审记录
  - 新增后端 service 层测试、WS handler 测试、前端交互测试

## 5. 数据结构 / 接口 / 协议变更

### 5.1 Conversation 主表扩展
- 当前 `Conversation` 已承载群聊主信息：`type/name/avatar/owner_id`。
- 本期对群聊追加以下字段：
  - `description`：群简介，`text` 或 `varchar(500)`
  - `announcement`：群公告，`text`
  - `announcement_updated_by`：最后更新公告的用户 ID
  - `announcement_updated_at`：最后更新时间
  - `group_kind`：群类型，建议枚举值：`official` / `player_created`
  - `join_mode`：加入模式，建议枚举值：`private` / `approval` / `public`
  - `status`：群状态，建议枚举值：`active` / `dissolved`
  - `dissolved_at`：解散时间
  - `dissolved_by`：解散操作者（仅允许群主）
- 说明：
  - 仅对 `type=GROUP` 的记录使用这些字段；私聊忽略。
  - `owner_id` 继续保留，作为群主角色的单一事实来源；`conversation_participants.role=owner` 作为冗余映射，需要事务内保持一致。

### 5.2 ConversationParticipant 表复用
- 当前字段已足够承载本期核心成员信息：
  - `conversation_id`
  - `user_id`
  - `role`（owner/admin/member）
  - `joined_at`
- 本期暂不新增过多成员资料字段。
- 若前端需要展示群内昵称，可继续沿用 `display_name`，但不是本期必须能力。

### 5.3 新增申请表
- 新增 `group_join_requests`：
  - `id`
  - `conversation_id`
  - `applicant_id`
  - `status`：`pending` / `approved` / `rejected` / `cancelled`
  - `message`：申请附言，可选
  - `reviewed_by`
  - `reviewed_at`
  - `created_at`
  - `updated_at`
- 关键约束：
  - 同一用户对同一群同一时间只能存在一条 `pending` 申请；
  - 已是成员不可申请；
  - 私密群不可申请；
  - 解散群不可申请。

### 5.4 新增邀请表
- 新增 `group_invitations`：
  - `id`
  - `conversation_id`
  - `inviter_id`
  - `invitee_id`
  - `status`：`pending` / `accepted` / `rejected` / `cancelled` / `expired`
  - `created_at`
  - `updated_at`
  - `responded_at`
- 关键约束：
  - 被邀请人已是成员时不可创建邀请；
  - 同一 invitee 对同一群只能有一条有效 `pending` 邀请；
  - 解散群不可邀请；
  - 邀请接受时，需二次校验群状态和成员关系，避免脏写。

### 5.5 Proto 扩展建议
- 扩展现有 `Conversation` message，增加：
  - `description`
  - `announcement`
  - `group_kind`
  - `join_mode`
  - `status`
  - `owner_id`
  - `my_role`
  - `member_count`
- 新增枚举：
  - `GroupKind`
  - `GroupJoinMode`
  - `GroupStatus`
  - `GroupJoinRequestStatus`
  - `GroupInvitationStatus`
- 新增请求/响应：
  - `GetGroupDetailRequest/Response`
  - `UpdateGroupProfileRequest/Response`
  - `UpdateGroupAnnouncementRequest/Response`
  - `GetGroupMembersRequest/Response`
  - `UpdateGroupMemberRoleRequest/Response`
  - `TransferGroupOwnershipRequest/Response`
  - `RemoveGroupMemberRequest/Response`
  - `LeaveGroupRequest/Response`
  - `DissolveGroupRequest/Response`
  - `ApplyToJoinGroupRequest/Response`
  - `GetGroupJoinRequestsRequest/Response`
  - `ReviewGroupJoinRequestRequest/Response`
  - `InviteToGroupRequest/Response`
  - `GetMyGroupInvitationsRequest/Response`
  - `RespondGroupInvitationRequest/Response`
- 推送建议：
  - `GroupUpdatedPush`
  - `GroupMembersUpdatedPush`
  - `GroupJoinRequestUpdatedPush`（只推给群主/管理员）
  - `GroupInvitationUpdatedPush`（只推给被邀请人）

## 6. 实现步骤
1. **数据模型扩展**
   - 扩展 `conversations` 表群字段
   - 新增 `group_join_requests`、`group_invitations`
   - 自动迁移与索引补齐
2. **服务层落地**
   - 群详情查询
   - 群资料更新
   - 成员列表查询
   - 管理员任免、群主转让
   - 退群、移除成员、解散群
   - 申请、审批、邀请、接受/拒绝
3. **协议层扩展**
   - 更新 `proto/chat/chat.proto`
   - 更新 `proto/ws.proto`
   - 重新生成 Go/TS protobuf 代码
4. **WebSocket handler 接入**
   - 接入新增请求路由
   - 对成功操作进行定向推送/全群推送
   - 统一错误返回与权限校验
5. **前端 Chat 体系扩展**
   - 现有群聊详情面板增加“群资料/公告/成员/管理”视图
   - 为不同角色展示不同操作入口
   - 增加入群申请列表、邀请处理入口
   - 对私密群/申请群/公开群展示不同按钮与文案
6. **测试与回归**
   - 后端 service + websocket 测试
   - 前端 Vitest 交互测试
   - 手工验证关键角色流转与加入流程

## 7. 测试方案
- 自动化测试：
  - Backend (`go test ./...`)：
    - 创建群时默认群主角色正确
    - 群主任命管理员 / 撤销管理员
    - 群主转让后原群主降为普通成员
    - 管理员不能处置群主和其他管理员
    - 管理员可编辑资料、审批申请、邀请成员、移除普通成员
    - 私密群不能申请加入
    - 申请群可提交申请并审批通过/拒绝
    - 公开群可直接加入
    - 已解散群不可申请、不可邀请、不可发消息、不可查看历史消息
    - 解散群仅群主可执行
  - Frontend (`npm test`)：
    - 群详情面板按角色展示正确操作
    - 不同加入模式展示正确 CTA
    - 申请审批/邀请接受拒绝流程正确更新 UI
- 手工验证：
  - 群主创建群并编辑头像、简介、公告
  - 群主任命管理员，管理员登录后验证拥有同等前台管理能力（除处置管理员/群主、转让群主、解散群）
  - 私密群：只能通过邀请进入，申请入口不可见
  - 申请群：提交申请后，管理员审批通过/拒绝
  - 公开群：直接加入后可在会话列表与成员列表中看到
  - 群主转让：新群主生效，原群主降为普通成员
  - 解散群：群成员列表、消息区、会话入口均不可继续访问历史内容

## 8. 风险与回滚
- 风险：
  - 当前官方群 topic room 是内存态，而本期群管理基于持久化 group conversation，两套模型会并存一段时间。
  - 现有 `Chat.jsx` 过于集中，继续叠加群管理逻辑会提高维护成本，建议在实现时适度拆组件。
  - `owner_id` 与 participant.role 双写存在一致性风险，必须在事务内更新。
  - 群解散后“不可查看历史”要求会影响现有消息拉取接口，需要统一在 service 层做状态校验。
  - 管理员权限接近群主，若校验实现不严谨，容易出现越权。
- 回滚方式：
  - 数据层采用“新增字段/新增表”方式，不破坏现有私聊/群聊基础能力；如需回滚，可先关闭前端入口与 WS 路由。
  - 对 `conversations` 的新增字段保持兼容默认值，避免影响旧会话读取。
  - 若群管理功能出现问题，可临时退回仅保留原始 `CreateConversation` 群聊能力。

## 9. Commit 记录
- Commit Hash:
- Commit Message:
