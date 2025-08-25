#!/bin/bash

# gRain Demo 快速启动脚本

set -e

echo "🚀 启动 gRain Demo 项目..."

# 检查 Go 版本
if ! command -v go &> /dev/null; then
    echo "❌ 错误: Go 未安装，请先安装 Go 1.21 或更高版本"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "✅ 检测到 Go 版本: $GO_VERSION"

# 检查必要的工具
echo "🔍 检查必要的工具..."

if ! command -v sqlite3 &> /dev/null; then
    echo "⚠️  警告: sqlite3 未安装，将使用内存数据库"
    USE_MEMORY_DB=true
else
    USE_MEMORY_DB=false
fi

# 安装依赖
echo "📦 安装项目依赖..."
go mod tidy
go mod download

# 创建必要的目录
echo "📁 创建必要的目录..."
mkdir -p bin
mkdir -p logs

# 设置环境变量
if [ "$USE_MEMORY_DB" = true ]; then
    export DB_TYPE=sqlite
    export DB_PATH=:memory:
else
    export DB_TYPE=sqlite
    export DB_PATH=./demo.db
fi

export LOG_LEVEL=debug
export GRAIN_DEBUG=true

echo "🔧 环境配置:"
echo "   数据库类型: $DB_TYPE"
echo "   数据库路径: $DB_PATH"
echo "   日志级别: $LOG_LEVEL"

# 检查是否需要生成代码
if [ -d "generated" ]; then
    echo "✅ 检测到已生成的代码"
else
    echo "🔨 生成代码..."
    if command -v ginframe-gen &> /dev/null; then
        ginframe-gen -input . -output ./generated
        echo "✅ 代码生成完成"
    else
        echo "⚠️  警告: ginframe-gen 工具未安装，跳过代码生成"
        echo "   请运行: go install github.com/grain-framework/grain/cmd/ginframe-gen@latest"
    fi
fi

# 运行测试
echo "🧪 运行测试..."
if go test -v ./...; then
    echo "✅ 测试通过"
else
    echo "❌ 测试失败，但继续启动..."
fi

# 启动应用
echo "🚀 启动应用..."
echo "   访问地址: http://localhost:8080"
echo "   健康检查: http://localhost:8080/health"
echo "   API文档: http://localhost:8080/api"
echo ""
echo "按 Ctrl+C 停止应用"

# 启动应用
go run main.go 