 .PHONY: help build run test clean generate migrate seed docker-build docker-run

# 默认目标
help:
	@echo "gRain Demo 项目构建工具"
	@echo ""
	@echo "可用命令:"
	@echo "  build      - 构建项目"
	@echo "  run        - 运行项目"
	@echo "  test       - 运行测试"
	@echo "  clean      - 清理构建文件"
	@echo "  generate   - 生成代码"
	@echo "  migrate    - 数据库迁移"
	@echo "  seed       - 填充测试数据"
	@echo "  docker-build - 构建Docker镜像"
	@echo "  docker-run   - 运行Docker容器"
	@echo "  deps       - 安装依赖"
	@echo "  lint       - 代码检查"
	@echo "  format     - 代码格式化"

# 安装依赖
deps:
	@echo "安装依赖..."
	go mod tidy
	go mod download

# 构建项目
build:
	@echo "构建项目..."
	go build -o bin/demo main.go

# 运行项目
run:
	@echo "运行项目..."
	go run main.go

# 运行测试
test:
	@echo "运行测试..."
	go test -v ./...

# 运行测试并生成覆盖率报告
test-coverage:
	@echo "运行测试并生成覆盖率报告..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 清理构建文件
clean:
	@echo "清理构建文件..."
	rm -rf bin/
	rm -rf coverage.out
	rm -rf coverage.html
	rm -rf demo.db
	rm -rf generated/

# 生成代码
generate:
	@echo "生成代码..."
	@if command -v ginframe-gen >/dev/null 2>&1; then \
		ginframe-gen -input . -output ./generated; \
	else \
		echo "警告: ginframe-gen 工具未安装，跳过代码生成"; \
	fi

# 数据库迁移
migrate:
	@echo "执行数据库迁移..."
	@if command -v sqlite3 >/dev/null 2>&1; then \
		sqlite3 demo.db < database/schema.sql; \
		echo "数据库迁移完成"; \
	else \
		echo "警告: sqlite3 未安装，请手动执行数据库迁移"; \
	fi

# 填充测试数据
seed:
	@echo "填充测试数据..."
	go run main.go seed

# 代码检查
lint:
	@echo "代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "警告: golangci-lint 未安装，跳过代码检查"; \
	fi

# 代码格式化
format:
	@echo "代码格式化..."
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "警告: goimports 未安装，跳过导入格式化"; \
	fi

# 构建Docker镜像
docker-build:
	@echo "构建Docker镜像..."
	docker build -t grain-demo .

# 运行Docker容器
docker-run:
	@echo "运行Docker容器..."
	docker run -p 8080:8080 --name grain-demo-container grain-demo

# 停止Docker容器
docker-stop:
	@echo "停止Docker容器..."
	docker stop grain-demo-container || true
	docker rm grain-demo-container || true

# 开发模式运行（自动重载）
dev:
	@echo "开发模式运行..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "警告: air 未安装，使用普通模式运行"; \
		go run main.go; \
	fi

# 安装开发工具
install-tools:
	@echo "安装开发工具..."
	go install github.com/cosmtrek/air@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@if command -v curl >/dev/null 2>&1; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
	else \
		echo "警告: curl 未安装，无法安装 golangci-lint"; \
	fi

# 完整构建流程
all: deps clean build test

# 开发环境设置
setup-dev: install-tools deps migrate seed
	@echo "开发环境设置完成！"
	@echo "运行 'make run' 启动项目"
	@echo "运行 'make dev' 启动开发模式（自动重载）"

# 生产环境构建
build-prod:
	@echo "生产环境构建..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o bin/demo main.go

# 性能测试
bench:
	@echo "运行性能测试..."
	go test -bench=. -benchmem ./...

# 安全检查
security-check:
	@echo "安全检查..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "警告: gosec 未安装，跳过安全检查"; \
	fi

# 显示项目信息
info:
	@echo "项目信息:"
	@echo "  名称: gRain Demo"
	@echo "  版本: 1.0.0"
	@echo "  Go版本: $(shell go version)"
	@echo "  模块路径: $(shell go list -m)"
	@echo "  工作目录: $(shell pwd)"