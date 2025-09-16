package processor

import (
	"strings"
	"testing"

	"go/types"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/packages"

	graintypes "github.com/isBlue-5/grain/pkg/annotation/types"
)

// TestNextGenParser_Basic 测试基本功能
func TestNextGenParser_Basic(t *testing.T) {
	parser := NewNextGenParser("frame:")

	assert.NotNil(t, parser)
	assert.Equal(t, "frame:", parser.prefix)
	assert.NotNil(t, parser.config)
	assert.NotNil(t, parser.packageCache)
	assert.NotNil(t, parser.typeInfoCache)
	assert.NotNil(t, parser.legacyParser)
}

// TestNextGenParser_ValidateTypeCompatibility 测试类型兼容性验证
func TestNextGenParser_ValidateTypeCompatibility(t *testing.T) {
	parser := NewNextGenParser("frame:")

	// 测试基本类型兼容性
	// 注意：这里只是测试接口调用，实际的类型检查需要真实的类型信息
	err := parser.ValidateTypeCompatibility(nil, nil)
	assert.Error(t, err) // 应该返回错误，因为nil类型不兼容

	contains := strings.Contains(err.Error(), "类型不兼容")
	assert.True(t, contains, "错误信息应该包含类型不兼容的提示")
}

// TestNextGenParser_GetMethodSignature 测试方法签名获取
func TestNextGenParser_GetMethodSignature(t *testing.T) {
	parser := NewNextGenParser("frame:")

	// 测试方法签名获取
	sig, err := parser.GetMethodSignature(nil)
	assert.Error(t, err) // 应该返回错误，因为传入了nil
	assert.Nil(t, sig)
}

// TestNextGenParser_ResolveTypeAlias 测试类型别名解析
func TestNextGenParser_ResolveTypeAlias(t *testing.T) {
	parser := NewNextGenParser("frame:")

	// 测试类型别名解析
	typ := parser.ResolveTypeAlias("UserID")
	assert.Nil(t, typ) // 当前实现返回nil
}

// TestNextGenParser_GetTypeInfo 测试类型信息获取
func TestNextGenParser_GetTypeInfo(t *testing.T) {
	parser := NewNextGenParser("frame:")

	// 测试类型信息获取
	typ := parser.GetTypeInfo(nil)
	assert.Nil(t, typ) // 当前实现返回nil
}

// TestEnhancedPackageInfo_Structure 测试增强包信息结构
func TestEnhancedPackageInfo_Structure(t *testing.T) {
	info := &EnhancedPackageInfo{
		Package:      nil,
		TypesInfo:    nil,
		Annotations:  make([]graintypes.Annotation, 0),
		Dependencies: make(map[string]*packages.Package),
		TypeMapping:  make(map[string]types.Type),
	}

	assert.NotNil(t, info)
	assert.NotNil(t, info.Annotations)
	assert.NotNil(t, info.Dependencies)
	assert.NotNil(t, info.TypeMapping)
	assert.Equal(t, 0, len(info.Annotations))
	assert.Equal(t, 0, len(info.Dependencies))
	assert.Equal(t, 0, len(info.TypeMapping))
}

// TestEnhancedParserContext_Structure 测试增强解析上下文结构
func TestEnhancedParserContext_Structure(t *testing.T) {
	ctx := &EnhancedParserContext{
		PackageName:       "test",
		FileName:          "test.go",
		ModulePath:        "github.com/test/module",
		TypesInfo:         nil,
		Package:           nil,
		TypeChecker:       nil,
		Imports:           make(map[string]*packages.Package),
		Dependencies:      make([]string, 0),
		MethodContext:     nil,
		TypeContext:       nil,
		GenerationOptions: nil,
	}

	assert.NotNil(t, ctx)
	assert.Equal(t, "test", ctx.PackageName)
	assert.Equal(t, "test.go", ctx.FileName)
	assert.Equal(t, "github.com/test/module", ctx.ModulePath)
	assert.NotNil(t, ctx.Imports)
	assert.NotNil(t, ctx.Dependencies)
}

// TestMethodContext_Structure 测试方法上下文结构
func TestMethodContext_Structure(t *testing.T) {
	ctx := &MethodContext{
		ReceiverType: nil,
		Method:       nil,
		Signature:    nil,
		IsPointer:    false,
	}

	assert.NotNil(t, ctx)
	assert.False(t, ctx.IsPointer)
}

// TestTypeContext_Structure 测试类型上下文结构
func TestTypeContext_Structure(t *testing.T) {
	ctx := &TypeContext{
		Type:          nil,
		StructType:    nil,
		InterfaceType: nil,
		Methods:       make([]*types.Func, 0),
	}

	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.Methods)
	assert.Equal(t, 0, len(ctx.Methods))
}

// TestGenerationOptions_Structure 测试代码生成选项结构
func TestGenerationOptions_Structure(t *testing.T) {
	opts := &GenerationOptions{
		EnableReflectionOptimization: true,
		EnableTypeValidation:         true,
		OutputFormat:                 "go",
	}

	assert.NotNil(t, opts)
	assert.True(t, opts.EnableReflectionOptimization)
	assert.True(t, opts.EnableTypeValidation)
	assert.Equal(t, "go", opts.OutputFormat)
}

// BenchmarkNextGenParser_Creation 基准测试：解析器创建
func BenchmarkNextGenParser_Creation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parser := NewNextGenParser("frame:")
		_ = parser
	}
}

// BenchmarkNextGenParser_ValidateTypeCompatibility 基准测试：类型兼容性验证
func BenchmarkNextGenParser_ValidateTypeCompatibility(b *testing.B) {
	parser := NewNextGenParser("frame:")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.ValidateTypeCompatibility(nil, nil)
	}
}
