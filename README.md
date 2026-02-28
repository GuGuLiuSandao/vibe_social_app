# 社交应用 (Social App)

前后端分离的实时社交应用，通信协议为 HTTP + WebSocket，业务载荷统一使用 Protocol Buffers。

## 技术栈
- 后端：Go、Gin、GORM、PostgreSQL、Redis、JWT、Gorilla WebSocket
- 前端：React、Vite、React Router
- 协议：Protocol Buffers（`proto/`）

## 目录结构
```
proto/      Protobuf 定义
backend/    Go 后端
frontend/   React 前端
docs/       项目文档
```

## 快速开始（本地）

### 1) 启动基础依赖（PostgreSQL + Redis）
可使用 Docker Compose：
```bash
docker compose up -d db redis
```

### 2) 编译 Protobuf（修改 proto 后必须执行）
```bash
brew install protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

make proto-go
make proto-ts
```

### 3) 启动后端
```bash
cd backend
cp .env.example .env
go mod tidy
go run cmd/api/main.go
```
- HTTP API: `http://localhost:8080/api/v1`
- WebSocket: `ws://localhost:8080/ws?token=<JWT>`

### 4) 启动前端
```bash
cd frontend
npm install
npm run dev
```
- 前端地址：`http://localhost:5173`

## 使用 Docker Compose 一键启动
```bash
docker compose up --build
```
默认端口：
- 前端：`5173`
- 后端：`8080`
- PostgreSQL：`5432`
- Redis：`6379`

## 协议说明
- HTTP 接口使用 `application/x-protobuf`
- WebSocket 使用二进制 `WsMessage` 信封（见 `proto/ws.proto`）
- 业务模块载荷：
  - `account.AccountPayload`
  - `chat.ChatPayload`
  - `relation.RelationPayload`

## 常用命令
```bash
make proto-go
make proto-ts
make run-backend
make run-frontend

cd backend && go test ./...
cd frontend && npm test
```

## 许可证
MIT
