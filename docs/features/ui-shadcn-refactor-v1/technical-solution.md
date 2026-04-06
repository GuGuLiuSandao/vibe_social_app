# 技术方案文档

## 1. 需求关联
- 对应目录：`docs/features/ui-shadcn-refactor-v1/`
- 对应需求文档：`requirement.md`
- 里程碑映射：`docs/plans/ROADMAP.md` 中 M1「聊天 MVP 完整化」的“私聊/群聊体验收敛”
- 清单映射：在 `docs/plans/M0_TASKS.md` 新增 UI 收敛执行项（T8）

## 2. 目标与非目标
- 目标：
  - 将前端核心交互组件统一到 `shadcn/ui` 风格组件目录（`frontend/src/components/ui/*`）。
  - 保留现有业务逻辑和 WebSocket 行为，仅重构表现层与布局密度。
  - 增强模块间留白和层级节奏，降低“高密度堆叠”带来的压迫感。
- 非目标：
  - 不改动后端协议、接口语义、消息流。
  - 不引入新的产品功能流程。

## 3. 方案概述
1. 引入 `shadcn/ui` 基础依赖与工具函数：`class-variance-authority`、`clsx`、`tailwind-merge`。
2. 新建基础 UI 组件：`Button`、`Input`、`Textarea`、`Card`、`Badge`（必要时含 `CardHeader/CardContent`）。
3. 移除 `frontend/src/lib/vercel-ui.js` 的使用，全部替换为 `components/ui/*`。
4. 登录页重构为卡片化表单布局，扩大区块间距与表单间距。
5. 聊天页重构：
   - 侧栏、列表、消息区统一卡片/分区语义；
   - 关系页与群管理区域提升留白与按钮尺寸一致性；
   - 维持现有文案和行为触发点，保障测试稳定。
6. 全局样式重构为 shadcn 变量命名体系并保持四主题切换。

## 4. 影响范围
- 文档：
  - `AGENTS.md`
  - `docs/features/ui-shadcn-refactor-v1/*`
  - `docs/plans/ROADMAP.md`
  - `docs/plans/M0_TASKS.md`
- 前端：
  - `frontend/src/styles.css`
  - `frontend/tailwind.config.cjs`
  - `frontend/src/components/ui/*`（新增）
  - `frontend/src/components/ThemeSwitcher.jsx`
  - `frontend/src/components/chat/*`
  - `frontend/src/pages/Login.jsx`
  - `frontend/src/pages/Chat.jsx`
  - `frontend/src/components/chat/GroupManagementComponents.test.jsx`（按组件签名调整）
  - `frontend/package.json`（依赖调整）

## 5. 实施步骤
1. 创建 feature 文档（需求/技术方案）并补充 roadmap 任务映射。
2. 增加 `shadcn/ui` 基础组件与 `cn` 工具函数。
3. 更新 Tailwind 配置与全局样式变量。
4. 重构登录页与聊天页主布局（提升间距和卡片化程度）。
5. 重构群管理子组件并清理旧 `Discord*` 组件透传。
6. 运行前端测试并记录 code review 结论。

## 6. 测试方案
- 自动化：
  - `cd frontend && npm test`
- 回归关注点：
  - 登录/注册表单提交流程
  - 聊天页自动连接、路由跳转、白名单 uid 行为
  - 群管理卡片组件基础交互

## 7. 风险与回滚
- 风险：
  - `Chat.jsx` 体量大，UI 调整可能误触交互行为。
  - 样式变量重构可能影响旧 class 映射。
- 回滚：
  - 本次主要为前端表现层变更，可通过回退对应前端提交快速恢复。

## 8. Commit 记录
- Commit Hash:
- Commit Message:
