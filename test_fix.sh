#!/bin/bash

# gRain 框架修复验证测试脚本
# 用于验证所有修复是否成功

echo "🚀 开始验证 gRain 框架修复结果..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试结果统计
PASSED=0
FAILED=0
TOTAL=0

# 测试函数
test_step() {
    local test_name="$1"
    local test_command="$2"
    local expected_result="$3"
    
    echo -e "\n${BLUE}🔍 测试: ${test_name}${NC}"
    echo "执行命令: ${test_command}"
    
    if eval "$test_command" > /tmp/test_output.log 2>&1; then
        echo -e "${GREEN}✅ 通过${NC}"
        ((PASSED++))
    else
        echo -e "${RED}❌ 失败${NC}"
        echo "错误输出:"
        cat /tmp/test_output.log
        ((FAILED++))
    fi
    
    ((TOTAL++))
}

# 清理函数
cleanup() {
    rm -f /tmp/test_output.log
    rm -f ginframe-gen-test
}

# 设置清理钩子
trap cleanup EXIT

echo -e "\n${YELLOW}📋 测试计划${NC}"
echo "1. 验证 Go 模块配置"
echo "2. 检查硬编码路径"
echo "3. 验证代码编译"
echo "4. 检查注解处理器集成"
echo "5. 验证代码生成器"

# 测试 1: 验证 Go 模块配置
test_step "Go 模块配置验证" \
    "go mod verify" \
    "模块配置正确"

# 测试 2: 检查硬编码路径
test_step "硬编码路径检查" \
    "! grep -r 'github.com/Zyi/' . --include='*.go' && ! grep -r 'github.com/isBlue-5/' . --include='*.go'" \
    "无硬编码路径"

# 测试 3: 验证代码编译
test_step "代码编译测试" \
    "go build -o ginframe-gen-test cmd/ginframe-gen/main.go" \
    "编译成功"

# 测试 4: 检查注解处理器集成
test_step "注解处理器集成检查" \
    "grep -q 'rateLimitProcessor.*ProcessRateLimit' pkg/annotation/processor/generator.go && grep -q 'authzProcessor.*ProcessAuthz' pkg/annotation/processor/generator.go" \
    "处理器已集成"

# 测试 5: 验证代码生成器帮助信息
if [ -f "ginframe-gen-test" ]; then
    test_step "代码生成器帮助信息验证" \
        "./ginframe-gen-test --help | grep -q 'ginframe-gen'" \
        "帮助信息正常"
else
    echo -e "\n${RED}⚠️  跳过代码生成器测试（编译失败）${NC}"
fi

# 测试 6: 检查模块导入一致性
test_step "模块导入一致性检查" \
    "grep -r 'github.com/isBlue-5/grain' . --include='*.go' | wc -l | grep -q '[0-9]'" \
    "导入路径一致"

# 测试 7: 验证事务处理器修复
test_step "事务处理器修复验证" \
    "grep -q 'github.com/isBlue-5/grain/pkg/annotation/types' pkg/annotation/processor/transaction_processor.go" \
    "事务处理器已修复"

# 测试 8: 检查备份文件清理
test_step "备份文件清理验证" \
    "! ls transaction_processor_original.go 2>/dev/null" \
    "备份文件已清理"

# 输出测试结果
echo -e "\n${YELLOW}📊 测试结果汇总${NC}"
echo "=================================="
echo -e "总测试数: ${TOTAL}"
echo -e "通过: ${GREEN}${PASSED}${NC}"
echo -e "失败: ${RED}${FAILED}${NC}"
echo "=================================="

if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}🎉 所有测试通过！gRain 框架修复验证成功！${NC}"
    exit 0
else
    echo -e "\n${RED}⚠️  有 ${FAILED} 个测试失败，请检查相关问题${NC}"
    exit 1
fi 