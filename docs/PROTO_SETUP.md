# Protobuf 编译指南

## 目录约定
- 源文件：`proto/**/*.proto`
- Go 生成目录：`backend/internal/proto`
- 前端生成目录：`frontend/src/proto`

## 安装工具

### 1) 安装 protoc
```bash
brew install protobuf
protoc --version
```

### 2) 安装 Go 插件
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

### 3) 安装前端插件（仅生成 TS 时需要）
```bash
cd frontend
npm install -D @bufbuild/protoc-gen-es @bufbuild/protobuf
```

## 编译命令

在仓库根目录执行：
```bash
make proto-go   # 仅生成 Go
make proto-ts   # 仅生成前端 TS
```

说明：
- `make proto` 当前等价于 `make proto-go`
- 若修改了 `proto/*.proto`，请至少重新执行对应目标

## WebSocket 协议说明（当前实现）
- 握手地址：`/ws?token=<JWT_TOKEN>`
- 前端当前会附带 `uid` 查询参数（兼容用），服务端实际鉴权基于 token
- 消息格式：二进制 Protobuf，信封为 `social.WsMessage`（见 `proto/ws.proto`）
- 业务 payload：
  - `social.account.AccountPayload`
  - `social.chat.ChatPayload`
  - `social.relation.RelationPayload`

## 常见问题
- `protoc-gen-go not found`：确认 `$GOPATH/bin` 已在 `PATH`
- `protoc-gen-es not found`：确认 `frontend/node_modules/.bin` 可用，或先执行 `npm install`
- 生成文件导入报错：确认 `proto` 包名与 `go_package` 未被手工改坏
