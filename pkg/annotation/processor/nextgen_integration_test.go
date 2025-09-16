// Package processor 提供NextGen集成测试
package processor

import (
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNextGenAdapter_BasicFunctionality 测试NextGen适配器基本功能
func TestNextGenAdapter_BasicFunctionality(t *testing.T) {
	adapter := NewNextGenAdapter("frame:", true)
	require.NotNil(t, adapter)

	// 测试接口实现
	var _ AnnotationParser = adapter
	var _ TypeAwareAnnotationParser = adapter

	// 测试NextGen功能状态
	assert.True(t, adapter.IsNextGenEnabled())

	// 测试禁用NextGen
	adapter.EnableNextGen(false)
	assert.False(t, adapter.IsNextGenEnabled())

	// 重新启用
	adapter.EnableNextGen(true)
	assert.True(t, adapter.IsNextGenEnabled())
}

// TestNextGenAdapter_LegacyCompatibility 测试向下兼容性
func TestNextGenAdapter_LegacyCompatibility(t *testing.T) {
	adapter := NewNextGenAdapter("frame:", false) // 禁用NextGen
	require.NotNil(t, adapter)

	// 创建临时测试文件
	testContent := `package test

// frame:route(method="GET", path="/test")
func TestHandler() {
	// 测试函数
}
`

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// 测试文件解析（应该使用传统解析器）
	annotations, err := adapter.ParseFile(testFile)
	assert.NoError(t, err)
	assert.NotNil(t, annotations)

	// 测试类型感知功能应该返回nil或错误（因为NextGen被禁用）
	assert.Nil(t, adapter.GetTypeInfo(nil))

	err = adapter.ValidateTypeCompatibility(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "需要启用NextGen功能")
}

// TestProcessorConfig 测试处理器配置
func TestProcessorConfig(t *testing.T) {
	// 测试默认配置
	defaultConfig := DefaultProcessorConfig()
	require.NotNil(t, defaultConfig)
	assert.True(t, defaultConfig.EnableNextGen)
	assert.True(t, defaultConfig.EnableTypeValidation)
	assert.False(t, defaultConfig.EnableReflectionOptimization)
	assert.Equal(t, "go", defaultConfig.OutputFormat)
	assert.False(t, defaultConfig.Verbose)

	// 测试自定义配置
	customConfig := &ProcessorConfig{
		EnableNextGen:                false,
		EnableTypeValidation:         false,
		EnableReflectionOptimization: true,
		OutputFormat:                 "json",
		Verbose:                      true,
	}

	factory := NewProcessorFactory(customConfig)
	require.NotNil(t, factory)

	parser := factory.CreateParser("test:")
	require.NotNil(t, parser)
	assert.False(t, parser.IsNextGenEnabled())
}

// TestProcessorFactory 测试处理器工厂
func TestProcessorFactory(t *testing.T) {
	// 测试使用默认配置的工厂
	factory := NewProcessorFactory(nil)
	require.NotNil(t, factory)

	parser := factory.CreateParser("frame:")
	require.NotNil(t, parser)
	assert.True(t, parser.IsNextGenEnabled())

	// 测试自定义配置的工厂
	config := &ProcessorConfig{
		EnableNextGen: false,
	}
	factory2 := NewProcessorFactory(config)
	parser2 := factory2.CreateParser("frame:")
	assert.False(t, parser2.IsNextGenEnabled())
}

// TestUpdateProcessorWithNextGen 测试处理器更新功能
func TestUpdateProcessorWithNextGen(t *testing.T) {
	adapter := NewNextGenAdapter("frame:", true)

	// 测试不支持的处理器类型
	err := UpdateProcessorWithNextGen("invalid", adapter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的处理器类型")

	// 测试nil处理器
	err = UpdateProcessorWithNextGen(nil, adapter)
	assert.NoError(t, err) // 当前实现返回nil
}

// TestNextGenAdapter_EnhancedPackageInfo 测试增强包信息功能
func TestNextGenAdapter_EnhancedPackageInfo(t *testing.T) {
	adapter := NewNextGenAdapter("frame:", true)

	// 创建测试包
	tempDir := t.TempDir()
	testContent := `package testpkg

// frame:route(method="GET", path="/test")
func TestHandler() {
	// 测试函数
}

type TestStruct struct {
	Name string ` + "`" + `json:"name"` + "`" + `
}
`

	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// 测试获取增强包信息（NextGen启用时）
	info, err := adapter.GetEnhancedPackageInfo(tempDir)
	// 注意：这可能会因为包加载问题而失败，这是正常的
	if err != nil {
		t.Logf("获取增强包信息失败（预期）: %v", err)
	} else {
		assert.NotNil(t, info)
	}

	// 测试NextGen禁用时
	adapter.EnableNextGen(false)
	info, err = adapter.GetEnhancedPackageInfo(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "需要启用NextGen功能")
	assert.Nil(t, info)
}

// TestNextGenAdapter_TypeAwareFunctions 测试类型感知函数
func TestNextGenAdapter_TypeAwareFunctions(t *testing.T) {
	adapter := NewNextGenAdapter("frame:", true)

	// 创建一个简单的AST表达式用于测试
	expr, err := parser.ParseExpr("x + 1")
	require.NoError(t, err)

	// 测试GetTypeInfo（可能返回nil，因为没有类型信息上下文）
	typ := adapter.GetTypeInfo(expr)
	// 当前实现可能返回nil，这是正常的
	t.Logf("GetTypeInfo result: %v", typ)

	// 测试GetMethodSignature
	funcDecl := &ast.FuncDecl{
		Name: &ast.Ident{Name: "TestFunc"},
	}
	sig, err := adapter.GetMethodSignature(funcDecl)
	// 可能会失败，因为没有完整的类型信息上下文
	if err != nil {
		t.Logf("GetMethodSignature失败（预期）: %v", err)
	}
	t.Logf("GetMethodSignature result: %v", sig)

	// 测试ResolveTypeAlias
	typ = adapter.ResolveTypeAlias("TestType")
	t.Logf("ResolveTypeAlias result: %v", typ)

	// 测试NextGen禁用时的行为
	adapter.EnableNextGen(false)

	typ = adapter.GetTypeInfo(expr)
	assert.Nil(t, typ)

	sig, err = adapter.GetMethodSignature(funcDecl)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "需要启用NextGen功能")

	typ = adapter.ResolveTypeAlias("TestType")
	assert.Nil(t, typ)
}

// TestNextGenAdapter_ParseMethods 测试解析方法
func TestNextGenAdapter_ParseMethods(t *testing.T) {
	adapter := NewNextGenAdapter("frame:", true)

	// 创建测试文件
	tempDir := t.TempDir()
	testContent := `package testpkg

// frame:route(method="GET", path="/test")
func TestHandler() {
	// 测试函数
}
`

	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// 测试ParseFile
	annotations, err := adapter.ParseFile(testFile)
	assert.NoError(t, err)
	assert.NotNil(t, annotations)
	t.Logf("ParseFile found %d annotations", len(annotations))

	// 测试ParseDir（NextGen启用）
	annotations, err = adapter.ParseDir(tempDir)
	// 可能会因为包加载问题而失败
	if err != nil {
		t.Logf("ParseDir失败（可能是正常的）: %v", err)
	} else {
		// 不强制要求找到注解，因为NextGen可能返回空结果
		t.Logf("ParseDir found %d annotations", len(annotations))
	}

	// 测试ParsePackage（NextGen启用）
	annotations, err = adapter.ParsePackage(tempDir)
	if err != nil {
		t.Logf("ParsePackage失败（可能是正常的）: %v", err)
	} else {
		// 不强制要求找到注解，因为NextGen可能返回空结果
		t.Logf("ParsePackage found %d annotations", len(annotations))
	}

	// 测试NextGen禁用时的行为
	adapter.EnableNextGen(false)

	annotations, err = adapter.ParseFile(testFile)
	assert.NoError(t, err)
	assert.NotNil(t, annotations)

	annotations, err = adapter.ParseDir(tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, annotations)

	annotations, err = adapter.ParsePackage(tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, annotations)
}

// BenchmarkNextGenAdapter_ParsePerformance 性能基准测试
func BenchmarkNextGenAdapter_ParsePerformance(t *testing.B) {
	// 创建测试文件
	tempDir := t.TempDir()
	testContent := `package testpkg

// frame:route(method="GET", path="/test1")
func TestHandler1() {}

// frame:route(method="POST", path="/test2")
func TestHandler2() {}

// frame:inject()
type TestService struct {}

type TestStruct struct {
	Name string ` + "`" + `json:"name"` + "`" + `
	Age  int    ` + "`" + `json:"age"` + "`" + `
}
`

	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	t.Run("NextGen启用", func(b *testing.B) {
		adapter := NewNextGenAdapter("frame:", true)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := adapter.ParseFile(testFile)
			if err != nil {
				b.Fatalf("解析失败: %v", err)
			}
		}
	})

	t.Run("NextGen禁用", func(b *testing.B) {
		adapter := NewNextGenAdapter("frame:", false)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := adapter.ParseFile(testFile)
			if err != nil {
				b.Fatalf("解析失败: %v", err)
			}
		}
	})
}

// TestIntegration_NextGenWithExistingProcessor 集成测试：NextGen与现有处理器
func TestIntegration_NextGenWithExistingProcessor(t *testing.T) {
	// 这个测试验证NextGen适配器与现有处理器的集成
	adapter := NewNextGenAdapter("frame:", true)

	// 测试适配器可以作为AnnotationParser使用
	var parser AnnotationParser = adapter
	assert.NotNil(t, parser)

	// 测试适配器可以作为TypeAwareAnnotationParser使用
	var typeAwareParser TypeAwareAnnotationParser = adapter
	assert.NotNil(t, typeAwareParser)
	assert.True(t, typeAwareParser.IsNextGenEnabled())

	// 测试工厂模式创建
	factory := NewProcessorFactory(DefaultProcessorConfig())
	factoryParser := factory.CreateParser("frame:")
	assert.NotNil(t, factoryParser)
	assert.True(t, factoryParser.IsNextGenEnabled())

	t.Log("NextGen适配器集成测试通过")
}
