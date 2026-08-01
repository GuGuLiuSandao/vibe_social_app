# Code Review 记录

## 1. Review 范围
- 前端 UI 重构 V1（shadcn/ui 统一化 + 留白优化）
- 范围：`frontend/src/pages/Login.jsx`、`frontend/src/pages/Chat.jsx`、`frontend/src/components/chat/*`、`frontend/src/components/ui/*`、样式与 Tailwind 配置

## 2. 发现的问题
- [x] 现有前端基础控件仍依赖 `@vercel/examples-ui`，不符合“统一组件体系”目标。
- [x] 聊天页侧栏/列表/消息区间距偏紧，信息块之间缺少呼吸感。
- [x] 群管理弹窗与卡片区域样式不统一，视觉层级不稳定。

## 3. 处理结果
- 新增 `shadcn/ui` 基础组件目录：`frontend/src/components/ui/`（Button/Input/Textarea/Card/Badge/Label；未使用的 Separator 已在后续健康检查中移除）。
- 聊天页与登录页改为使用 shadcn 组件，并移除 `frontend/src/lib/vercel-ui.js`。
- 已移除 `@vercel/examples-ui` 依赖，Tailwind 配置切换到本地主题变量体系。
- 聊天三栏与弹窗完成留白调整：边距、内边距、列表项间距、按钮尺寸与状态色统一。
- 已在 `AGENTS.md` 增加规则：前端核心控件默认使用 `shadcn/ui`。
- 已验证：
  - `cd frontend && npm test`
  - `cd frontend && npm run build`

## 4. 遗留事项
- `Chat.jsx` 仍是大文件，建议后续继续按模块拆分（侧栏、消息区、关系区、弹窗状态）。
- 当前主题系统保持历史变量兼容，后续可进一步清理不再使用的旧 class 映射样式。
