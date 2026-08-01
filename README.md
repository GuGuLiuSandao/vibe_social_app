# Social App

一个 Web First 的实时社交应用。项目采用 Go + React 前后端分离架构，以 Protocol Buffers 作为共享通信契约，通过 HTTP 完成登录注册，通过 WebSocket 承载账号、关系链、私聊、群聊和社群操作。

## 当前能力

- 账号：注册、登录、JWT 鉴权、用户搜索、资料更新和头像上传
- 关系链：关注、取关、关注列表、粉丝、互关、拉黑和解除拉黑
- 聊天：私聊、群聊、会话列表、消息列表和已读状态
- 社群：建群、群资料、公告、成员角色、邀请、入群申请审批、群主转让、退群和解散
- 话题房：官方固定话题房、加入/离开、成员列表和实时消息
- 实时连接：基于二进制 Protobuf 信封的 WebSocket 请求、响应和推送

产品版本、当前建设重点和后续方向见 [Roadmap](docs/plans/ROADMAP.md)。

## 技术栈

| 层级 | 技术 |
|---|---|
| 后端 | Go 1.25.3、Gin、GORM、PostgreSQL 16、Redis 7、JWT、Gorilla WebSocket |
| 前端 | React 19、Vite 6、React Router 7、Tailwind CSS、shadcn/ui 风格本地组件 |
| 通信契约 | Protocol Buffers、Buf Protobuf JavaScript runtime |
| 测试 | Go test、Vitest、Testing Library、真实 PostgreSQL/Redis/HTTP/WebSocket 集成测试 |
| Mutation Testing | Gremlins 0.6.0、Stryker 9.6.1 |
| CI | GitHub Actions |

## 目录结构

```text
backend/                 Go API、业务服务、数据库和 WebSocket
frontend/                React 页面、组件和客户端逻辑
proto/                   前后端共享的 Protobuf 源定义
docs/plans/              产品 Roadmap
docs/specs/              跨版本稳定的产品行为规格
docs/changes/            每次研发变更的需求、设计、测试与评审记录
.engineering-loop/       Social App 的 Develop Loop 项目适配
scripts/quality/         本地与 CI 共用的质量门禁脚本
```

生成代码位于 `backend/internal/proto/` 和 `frontend/src/proto/`。修改 `proto/` 后应通过 Make 命令重新生成，不直接编辑生成文件。

## 本地启动

### 环境要求

- Go 1.25.3
- Node.js 20 与 npm
- Docker + Docker Compose
- 修改 Protobuf 时需要 `protoc` 和 `protoc-gen-go`

### 1. 安装依赖

```bash
cp backend/.env.example backend/.env
cd backend && go mod download && cd ..
npm ci --prefix frontend
```

### 2. 启动 PostgreSQL 和 Redis

```bash
docker compose up -d db redis
```

### 3. 启动后端和前端

分别在两个终端运行：

```bash
make run-backend
```

```bash
make run-frontend
```

- 前端：`http://localhost:5173`
- HTTP API：`http://localhost:8080/api/v1`
- WebSocket：`ws://localhost:8080/ws?token=<JWT>`

也可以一次启动完整容器环境：

```bash
docker compose up --build
```

## Protocol Buffers

- HTTP 请求和响应使用 `application/x-protobuf`
- WebSocket 使用二进制 `WsMessage` 信封，定义见 `proto/ws.proto`
- 账号、聊天和关系链载荷分别定义在 `proto/account/`、`proto/chat/` 和 `proto/relation/`

修改协议后重新生成两端代码：

```bash
make proto-go
make proto-ts
```

更完整的环境说明见 [Proto Setup](docs/PROTO_SETUP.md)。

## 研发流程

本项目采用 Requirement-driven Develop Loop。通用方法维护在独立的 `engineering-loop` 仓库中；接入或更新时，将该仓库地址或路径与业务仓库一起交给 AI 阅读。本仓库只保存项目适配、Codex Agent 配置和已经填写的变更产物。

标准变更依次经过：

```text
需求确认
→ 技术设计与独立评审
→ 测试用例设计与独立评审
→ 实现
→ 独立代码评审
→ 本地质量门禁
→ PR / GitHub Actions
```

开始变更前需要确定稳定的 change ID 和严谨度，并在 `docs/changes/<change-id>/` 维护对应产物。对外行为变更同步维护 `docs/specs/`，Requirement、Test Case 和自动化测试通过稳定 ID 建立追溯关系。

- 项目适配：[Project Profile](.engineering-loop/project.md)
- Agent 配置：`.codex/agents/`
- 文档索引：[Docs Index](docs/README.md)

## 质量门禁

本地与 GitHub Actions 使用同一套变更分类规则：

| 分类 | 典型变更 | 执行方式 |
|---|---|---|
| `docs` | README、普通说明和 Roadmap | `make quality-docs` |
| `engineering` | CI、质量脚本、流程配置、规格和变更产物 | `make quality-engineering` |
| `develop` | 带 `develop-loop` 标签的产品代码或协议变更 | 完整六项质量门禁 |

`make quality` 会比较当前分支和工作区相对 `origin/master` 的真实 diff，并自动选择适用门禁。需求变更在完成 Develop Loop 产物后运行：

```bash
CHANGE_LABELS=develop-loop make quality
```

完整 `develop` 门禁包括：

| 门禁 | 命令 | 内容 |
|---|---|---|
| 静态与构建 | `make quality-static` | 追溯、diff、Go build/vet、前端 build、Compose 和流程契约 |
| 后端测试 | `make test-backend` | JWT 和 WebSocket manager 单元测试及机器可读报告 |
| 前端测试 | `make test-frontend` | 客户端工具、WebSocket、Chat 控件和群管理组件测试 |
| 集成测试 | `make test-integration` | 隔离 PostgreSQL/Redis/HTTP/WebSocket 流程和并行隔离 |
| 后端变异测试 | `make mutation-backend` | Gremlins 检查实际变更的 Go 行 |
| 前端变异测试 | `make mutation-frontend` | Stryker 检查实际变更的前端行 |

需要直接复验完整六项时也可运行：

```bash
make quality-develop
```

GitHub Actions 对每个 Pull Request 的实际 diff 和标签独立分类，只执行对应分支，并由固定的 `quality-gate` 汇总结果。产品路径没有 `develop-loop` 标签或出现未知路径时分类失败；标签只能增加检查。合并到 `master` 后会再执行完整六项门禁，并上传 `quality/**` 机器可读证据。

## 常用命令

```bash
make run-backend
make run-frontend
make build-backend
make proto-go
make proto-ts
make quality
```

## 许可证

MIT
