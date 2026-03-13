# Docs Index

## 当前有效文档

### 项目总览
- `../AGENTS.md`：仓库协作规则、开发流程、交付要求
- `../README.md`：项目对外说明、技术栈、启动与测试方式
- `../TEMP_STATUS.md`：当前项目状态、已实现能力、剩余风险

### 规划与执行
- `plans/ROADMAP.md`：产品版本口径、里程碑、阶段目标
- `plans/M0_TASKS.md`：稳定性基线与执行清单
- `plans/M0_MANUAL_QA.md`：最小手工回归脚本
- `plans/M0_FAILURE_MATRIX.md`：M0 失败矩阵与修复记录

### 专项说明
- `PROTO_SETUP.md`：Proto 编译与使用说明
- `DEPENDENCIES.md`：依赖说明
- `backend/websocket-implementation.md`：WebSocket 模块实现说明

### 功能交付文档
- `features/README.md`：功能目录规则与模板说明
- `features/_template/`：需求设计 / 技术方案 / Code Review 模板

## 功能文档目录约定
- 每个功能按需求维度归档到 `features/<feature-name>/`
- 每个功能目录至少包含：
  - `requirement.md`
  - `technical-solution.md`
  - `code-review.md`

## 已归档文档
- `deprecated/plans/V1_0_0_TEST_CHECKLIST_2026-03-09.md`
  - 原因：版本口径已过时，当前 1.0/2.0/3.0 定义已调整

## 当前建议阅读顺序
1. `../README.md`
2. `plans/ROADMAP.md`
3. `../TEMP_STATUS.md`
4. `features/README.md`
5. 具体功能目录文档
