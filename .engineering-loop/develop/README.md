# Develop Loop

Develop Loop 解决需求驱动研发中的两个问题：

1. 把需求方的目标、规则和边界转化为可验证的实现契约。
2. 让本次变更进入项目长期有效的质量体系。

本文件定义通用流程。项目目录、命令、分支和交付平台由 `.engineering-loop/project.md` 提供。

## 1. 基本概念

- `change-id`：一次变更的稳定标识，格式由项目定义。
- Requirement Contract：需求事实、范围、规则、边界和验收标准。
- Project Profile：项目对 Develop Loop 的适配配置。
- Quality Gate：项目可重复执行、以退出状态表达结果的质量命令。
- Delivery Evidence：评审结论、命令结果、提交和 MR/PR 状态。

## 2. 流程强度

协调者根据影响选择强度，并在 Requirement Contract 中记录依据。

| 强度 | 适用条件 | 必需产物 |
|---|---|---|
| `quick` | 目标明确、影响局部、容易回滚 | requirement、code review、quality evidence |
| `standard` | 跨模块、改变对外行为或存在兼容风险 | 全部产物 |
| `critical` | 权限、安全、数据迁移、资金或真实外部写入 | 全部产物，并采用项目声明的额外审批和门禁 |

存在多种合理产品答案、重要边界不明确或选择会改变架构、数据、权限和外部副作用时，先向需求方澄清。其余可从仓库事实可靠推出的细节由协调者判断，并写入契约。

## 3. 标准流程

```text
Preflight
→ Requirement Contract
→ 需求方确认关键契约
→ Technical Design
→ Design Review
→ Test Design
→ Test Review
→ Implementation
→ Code Review
→ Local Quality Gates
→ Commit / Push / MR or PR / CI
```

`quick` 可以把技术设计和测试设计写入 requirement，由协调者直接实现；代码评审和适用的质量门禁仍然执行。

### 3.1 Preflight

1. 读取 Project Profile 和仓库规则。
2. 检查当前分支、远端基线和工作区状态。
3. 识别已有改动的归属，保护需求范围外的用户工作。
4. 按 Project Profile 建立开发分支或隔离工作区。

### 3.2 Requirement Contract

协调者根据原始诉求、仓库事实和必要澄清编写 requirement：

- 目标和成功结果；
- 已确认范围和不在本次范围内的事项；
- 输入、行为、输出、权限和异常；
- 规则矩阵和边界 Case；
- Given / When / Then 验收标准；
- 与现有行为规格的关系。

契约中不能保留会改变实现方向的开放问题。需求方确认关键契约后进入下游流程。

### 3.3 Technical Design

设计者把 Requirement Contract 映射到当前代码：

- 真实入口和数据流；
- API、Schema、存储和配置变化；
- 兼容策略和实现约束；
- 验收标准到测试类型的映射；
- 风险和任务切分。

Reviewer 独立检查设计是否完整、准确且足以让实现者无需猜测。存在 blocker 时回到设计，默认最多循环三次。

### 3.4 Test Design

测试设计先于实现，聚焦高风险行为、错误路径、兼容性和副作用：

- 每个 Case 必须有明确断言；
- 映射 Requirement Contract 或项目行为规格；
- 选择合适的 unit、integration、contract、UI、smoke 或其他验证层；
- 明确隔离方式和测试数据；
- 合并重复、低价值或不能提供失败信号的 Case。

Test Reviewer 独立检查漏测、滥写、弱断言风险和可自动化程度。存在 blocker 时回到测试设计，默认最多循环三次。

### 3.5 Implementation and Review

实现者只实现已通过评审的契约、设计和测试范围，并同步项目的长期行为规格与回归保护。

Code Reviewer 使用 Requirement Contract、Design、Test Cases、行为规格、代码差异和质量结果进行追溯审查，至少检查：

- 行为一致性和范围控制；
- 安全、权限、数据兼容和错误处理；
- 测试是否真正覆盖设计 Case；
- 弱断言、过度 Mock 和隔离问题；
- 项目质量门禁是否完整执行。

存在 blocker 时回到实现者，默认最多循环三次。

### 3.6 Change Control

实现中发现的新事实分三类处理：

- 不改变目标、边界或对外行为：补充设计后继续。
- 改变已确认规则、范围或验收标准：先更新 Requirement Contract 并重新确认。
- 属于独立需求或缺陷：记录后转入对应 Loop，不静默扩大当前变更。

### 3.7 Delivery

本地门禁通过后，按 Project Profile 完成交付：

1. 记录实际运行的命令和结果。
2. 创建有明确范围的提交。
3. 推送开发分支并创建 MR/PR。
4. 等待 CI；失败时在同一变更中诊断、修复并重新验证。
5. 输出可审查的变更、风险和质量证据。

合并权限由项目声明，Develop Loop 本身不假设自动合并。

## 4. 角色独立性

- Requirement Contract 由协调者维护，需求方确认关键产品契约。
- Design Reviewer 不参与原设计编写。
- Test Reviewer 不参与原测试设计。
- Code Reviewer 不修改被评审的业务代码和测试代码。
- Reviewer 的显式 blocker 优先于分数；评分只帮助稳定评审尺度。

## 5. 项目适配要求

每个接入项目必须维护 `.engineering-loop/project.md`，至少声明：

- `change-id`、分支和产物路径规则；
- 项目结构和长期行为规格位置；
- 按影响范围选择的质量命令；
- 外部系统和数据隔离边界；
- Commit、MR/PR、CI 和合并策略；
- `critical` 变更需要的额外控制。
