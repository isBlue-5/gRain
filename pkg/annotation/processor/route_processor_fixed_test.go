package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grain-framework/grain/pkg/annotation/registry"
	"github.com/stretchr/testify/assert"
)

func TestFixedRouteProcessor_ProcessRoutes(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建注册中心
	reg := registry.NewRegistry()

	// 创建路由处理器
	processor := NewFixedRouteProcessor(reg, "@frame:", tempDir)

	// 测试空注册中心
	err := processor.ProcessRoutes()
	assert.NoError(t, err)

	// 验证生成了通用接口文件
	commonFile := filepath.Join(tempDir, "common", "interfaces.go")
	assert.FileExists(t, commonFile)

	// 读取文件内容验证
	content, err := os.ReadFile(commonFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "type Controller interface")
	assert.Contains(t, string(content), "type Validator interface")
	assert.Contains(t, string(content), "func checkRoles")
}

func TestFixedRouteProcessor_GenerateCommonInterfaces(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建注册中心
	reg := registry.NewRegistry()

	// 创建路由处理器
	processor := NewFixedRouteProcessor(reg, "@frame:", tempDir)

	// 测试生成通用接口
	err := processor.generateCommonInterfaces()
	assert.NoError(t, err)

	// 验证文件生成
	commonFile := filepath.Join(tempDir, "common", "interfaces.go")
	assert.FileExists(t, commonFile)

	// 验证文件内容
	content, err := os.ReadFile(commonFile)
	assert.NoError(t, err)

	expectedContent := []string{
		"package common",
		"github.com/gin-gonic/gin",
		"type Controller interface",
		"type Validator interface",
		"func checkRoles",
	}

	for _, expected := range expectedContent {
		assert.Contains(t, string(content), expected)
	}
}

func TestFixedRouteProcessor_ResolvePackagePath(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建注册中心
	reg := registry.NewRegistry()

	// 创建路由处理器
	processor := NewFixedRouteProcessor(reg, "@frame:", tempDir)

	// 测试相对路径
	relPath := "controllers/user_controller.go"
	result := processor.resolvePackagePath(relPath)
	assert.Equal(t, "controllers", result)

	// 测试绝对路径
	absPath := filepath.Join(tempDir, "controllers", "user_controller.go")
	result = processor.resolvePackagePath(absPath)
	assert.Equal(t, "controllers", result)
}
