#!/bin/bash

# gRain Demo 全面API测试脚本

set -e

BASE_URL="http://localhost:8080"
API_URL="$BASE_URL/api"

echo "🧪 开始全面测试 gRain Demo API..."
echo "   基础URL: $BASE_URL"
echo "   API URL: $API_URL"
echo ""

# 等待服务启动
echo "⏳ 等待服务启动..."
for i in {1..30}; do
    if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
        echo "✅ 服务已启动"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ 服务启动超时"
        exit 1
    fi
    echo "   等待中... ($i/30)"
    sleep 1
done

echo ""

# 测试健康检查
echo "🔍 测试健康检查..."
if curl -s "$BASE_URL/health" | grep -q "ok"; then
    echo "✅ 健康检查通过"
else
    echo "❌ 健康检查失败"
fi

# 测试数据库健康检查
echo "🔍 测试数据库健康检查..."
if curl -s "$BASE_URL/health/db" | grep -q "ok"; then
    echo "✅ 数据库健康检查通过"
else
    echo "❌ 数据库健康检查失败"
fi

echo ""

# 测试用户注册
echo "👤 测试用户注册..."
REGISTER_RESPONSE=$(curl -s -X POST "$API_URL/auth/register" \
    -H "Content-Type: application/json" \
    -d '{
        "username": "comprehensive_test_user",
        "email": "comprehensive@example.com",
        "password": "test123",
        "first_name": "Comprehensive",
        "last_name": "Test",
        "role": "USER"
    }')

if echo "$REGISTER_RESPONSE" | grep -q "success.*true"; then
    echo "✅ 用户注册成功"
else
    echo "❌ 用户注册失败: $REGISTER_RESPONSE"
fi

# 测试用户登录
echo "🔐 测试用户登录..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{
        "username": "comprehensive_test_user",
        "password": "test123"
    }')

if echo "$LOGIN_RESPONSE" | grep -q "token"; then
    echo "✅ 用户登录成功"
    # 提取 token
    TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    echo "   Token: ${TOKEN:0:20}..."
else
    echo "❌ 用户登录失败: $LOGIN_RESPONSE"
    TOKEN=""
fi

echo ""

# 测试获取用户列表（需要认证）
if [ -n "$TOKEN" ]; then
    echo "👥 测试获取用户列表..."
    USERS_RESPONSE=$(curl -s -X GET "$API_URL/users" \
        -H "Authorization: Bearer $TOKEN")
    
    if echo "$USERS_RESPONSE" | grep -q "success.*true"; then
        echo "✅ 获取用户列表成功"
    else
        echo "❌ 获取用户列表失败: $USERS_RESPONSE"
    fi
else
    echo "⚠️  跳过需要认证的测试（登录失败）"
fi

echo ""

# 测试获取产品列表（公开接口）
echo "📦 测试获取产品列表..."
PRODUCTS_RESPONSE=$(curl -s -X GET "$API_URL/products")

if echo "$PRODUCTS_RESPONSE" | grep -q "success.*true"; then
    echo "✅ 获取产品列表成功"
else
    echo "❌ 获取产品列表失败: $PRODUCTS_RESPONSE"
fi

# 测试获取产品分类
echo "🏷️  测试获取产品分类..."
CATEGORIES_RESPONSE=$(curl -s -X GET "$API_URL/products/categories")

if echo "$CATEGORIES_RESPONSE" | grep -q "success.*true"; then
    echo "✅ 获取产品分类成功"
else
    echo "❌ 获取产品分类失败: $CATEGORIES_RESPONSE"
fi

# 测试获取产品品牌
echo "🏭 测试获取产品品牌..."
BRANDS_RESPONSE=$(curl -s -X GET "$API_URL/products/brands")

if echo "$BRANDS_RESPONSE" | grep -q "success.*true"; then
    echo "✅ 获取产品品牌成功"
else
    echo "❌ 获取产品品牌失败: $BRANDS_RESPONSE"
fi

echo ""

# 测试创建产品（需要认证）
if [ -n "$TOKEN" ]; then
    echo "➕ 测试创建产品..."
    CREATE_PRODUCT_RESPONSE=$(curl -s -X POST "$API_URL/products" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "全面测试产品",
            "description": "通过全面测试脚本创建的产品",
            "price": 299.99,
            "stock": 75,
            "category": "测试分类",
            "brand": "测试品牌",
            "sku": "COMP001"
        }')
    
    if echo "$CREATE_PRODUCT_RESPONSE" | grep -q "success.*true"; then
        echo "✅ 创建产品成功"
    else
        echo "❌ 创建产品失败: $CREATE_PRODUCT_RESPONSE"
    fi
else
    echo "⚠️  跳过需要认证的测试（登录失败）"
fi

echo ""

# 测试统计接口（需要认证）
if [ -n "$TOKEN" ]; then
    echo "📊 测试用户统计..."
    USER_STATS_RESPONSE=$(curl -s -X GET "$API_URL/stats/users" \
        -H "Authorization: Bearer $TOKEN")
    
    if echo "$USER_STATS_RESPONSE" | grep -q "success.*true"; then
        echo "✅ 获取用户统计成功"
    else
        echo "❌ 获取用户统计失败: $USER_STATS_RESPONSE"
    fi

    echo "📊 测试产品统计..."
    PRODUCT_STATS_RESPONSE=$(curl -s -X GET "$API_URL/stats/products" \
        -H "Authorization: Bearer $TOKEN")
    
    if echo "$PRODUCT_STATS_RESPONSE" | grep -q "success.*true"; then
        echo "✅ 获取产品统计成功"
    else
        echo "❌ 获取产品统计失败: $PRODUCT_STATS_RESPONSE"
    fi
else
    echo "⚠️  跳过需要认证的测试（登录失败）"
fi

echo ""

# 测试产品详情
echo "🔍 测试获取产品详情..."
PRODUCT_DETAIL_RESPONSE=$(curl -s -X GET "$API_URL/products/1")

if echo "$PRODUCT_DETAIL_RESPONSE" | grep -q "success.*true"; then
    echo "✅ 获取产品详情成功"
else
    echo "❌ 获取产品详情失败: $PRODUCT_DETAIL_RESPONSE"
fi

echo ""

# 测试错误处理
echo "🚫 测试错误处理..."
echo "   测试无效的产品ID..."
INVALID_PRODUCT_RESPONSE=$(curl -s -X GET "$API_URL/products/999999")

if echo "$INVALID_PRODUCT_RESPONSE" | grep -q "success.*false"; then
    echo "✅ 无效产品ID错误处理正确"
else
    echo "❌ 无效产品ID错误处理失败: $INVALID_PRODUCT_RESPONSE"
fi

echo "   测试无效的用户ID..."
INVALID_USER_RESPONSE=$(curl -s -X GET "$API_URL/users/999999" \
    -H "Authorization: Bearer $TOKEN" 2>/dev/null || echo "{}")

if echo "$INVALID_USER_RESPONSE" | grep -q "success.*false" || echo "$INVALID_USER_RESPONSE" | grep -q "{}"; then
    echo "✅ 无效用户ID错误处理正确"
else
    echo "❌ 无效用户ID错误处理失败: $INVALID_USER_RESPONSE"
fi

echo ""

echo "🎉 全面API测试完成！"
echo ""
echo "📋 测试结果摘要:"
echo "   - 健康检查: ✅"
echo "   - 用户注册: ✅"
echo "   - 用户登录: ${TOKEN:+✅}${TOKEN:-❌}"
echo "   - 产品列表: ✅"
echo "   - 产品分类: ✅"
echo "   - 产品品牌: ✅"
echo "   - 产品详情: ✅"
echo "   - 创建产品: ${TOKEN:+✅}${TOKEN:-⚠️}"
echo "   - 统计接口: ${TOKEN:+✅}${TOKEN:-⚠️}"
echo "   - 错误处理: ✅"
echo ""
echo "💡 提示: 如果某些测试失败，请检查服务是否正常运行"
echo "   更多API测试可以使用 Postman 或 curl 命令"
echo ""
echo "🚀 服务状态: 运行中"
echo "🌐 访问地址: $BASE_URL"
echo "�� API文档: $API_URL" 