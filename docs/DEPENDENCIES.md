# 项目依赖清单

## 系统依赖（建议）
- Go 1.25.x（项目 `go.mod` 当前为 `go 1.25.3`）
- Node.js 20+（推荐 LTS）
- npm 10+
- PostgreSQL 16+
- Redis 7+
- protoc 3.21+（本地示例为 `libprotoc 33.x`）
- Docker / Docker Compose（可选，用于一键启动）

## 后端核心依赖
- `github.com/gin-gonic/gin`
- `gorm.io/gorm`
- `gorm.io/driver/postgres`
- `github.com/golang-jwt/jwt/v5`
- `github.com/gorilla/websocket`
- `github.com/redis/go-redis/v9`
- `google.golang.org/protobuf`
- `github.com/joho/godotenv`

测试/本地辅助也使用：
- `gorm.io/driver/sqlite`

## 前端核心依赖
- `react` / `react-dom`（当前为 19.x）
- `react-router-dom`（当前为 7.x）
- `vite`（当前为 6.x）
- `@bufbuild/protobuf`（Protobuf 编解码）

测试依赖：
- `vitest`
- `@testing-library/react`
- `@testing-library/jest-dom`
- `jsdom`

## Protobuf 生成工具

Go 代码生成：
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

前端 TypeScript 代码生成（可选）：
```bash
cd frontend
npm install -D @bufbuild/protoc-gen-es @bufbuild/protobuf
```
