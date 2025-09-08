#!/bin/bash

echo "开始测试gRain代码生成器..."

# 清理之前的测试输出
rm -rf ./test-output

# 尝试编译代码生成器
echo "编译代码生成器..."
if go build -o ginframe-gen ./cmd/ginframe-gen; then
    echo "✓ 代码生成器编译成功"
    
    # 运行代码生成器
    echo "运行代码生成器..."
    if ./ginframe-gen -verbose -output ./test-output ./examples/annotation; then
        echo "✓ 代码生成成功"
        
        # 显示生成的文件
        echo "生成的文件:"
        find ./test-output -name "*.go" | head -10
        
    else
        echo "✗ 代码生成失败"
        exit 1
    fi
else
    echo "✗ 代码生成器编译失败"
    exit 1
fi

echo "测试完成!" 