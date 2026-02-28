.PHONY: proto build clean help

# Protobuf 编译
PROTO_DIR = proto
GO_OUT_DIR = backend/internal/proto
FRONTEND_PROTO_DIR = frontend/src/proto
FRONTEND_PROTO_BIN = frontend/node_modules/.bin
PROTO_FILES = $(shell find $(PROTO_DIR) -name "*.proto")

# 检查 protoc 编译器是否安装
check-protoc:
	@which protoc > /dev/null || (echo "protoc not found, please install from https://grpc.io/docs/protoc-installation/"; exit 1)
	@which protoc-gen-go > /dev/null || (echo "protoc-gen-go not found, installing..."; go install google.golang.org/protobuf/cmd/protoc-gen-go@latest)

# 编译 Protobuf 为 Go 代码
proto-go: check-protoc
	@rm -rf $(GO_OUT_DIR)
	@mkdir -p $(GO_OUT_DIR)
	@echo "Compiling proto files to Go..."
	protoc --proto_path=$(PROTO_DIR) \
		--go_out=$(GO_OUT_DIR) \
		--go_opt=paths=source_relative \
		$(PROTO_FILES)
	@echo "Go proto files compiled successfully!"

# 编译 Protobuf 为 TypeScript 代码（可选，用于前端）
proto-ts: check-protoc
	@PATH=$(FRONTEND_PROTO_BIN):$$PATH; which protoc-gen-es > /dev/null || (echo "protoc-gen-es not found, install with: npm install -D @bufbuild/protoc-gen-es @bufbuild/protobuf"; exit 1)
	@mkdir -p $(FRONTEND_PROTO_DIR)
	@echo "Compiling proto files to TypeScript..."
	PATH=$(FRONTEND_PROTO_BIN):$$PATH protoc --proto_path=$(PROTO_DIR) \
		--es_out=$(FRONTEND_PROTO_DIR) \
		--es_opt=target=ts,import_extension=none \
		$(PROTO_FILES)
	@echo "TypeScript proto files compiled successfully!"

# 编译所有
proto: proto-go

# 安装 Go 依赖
deps:
	cd backend && go mod tidy

# 运行后端
run-backend:
	cd backend && go run cmd/api/main.go

# 运行前端
run-frontend:
	cd frontend && npm run dev

# 构建后端
build-backend:
	cd backend && go build -o bin/server cmd/api/main.go

# 清理生成的文件
clean:
	rm -rf backend/internal/proto
	rm -rf frontend/src/proto
	rm -rf backend/bin

# 帮助
help:
	@echo "Available targets:"
	@echo "  proto         - Compile proto files to Go"
	@echo "  proto-go      - Compile proto files to Go"
	@echo "  proto-ts       - Compile proto files to TypeScript"
	@echo "  deps          - Install Go dependencies"
	@echo "  run-backend   - Run backend server"
	@echo "  run-frontend  - Run frontend dev server"
	@echo "  build-backend - Build backend binary"
	@echo "  clean         - Clean generated files"
	@echo "  help          - Show this help message"
