// 测试NextGenParser的核心功能
package processor

import (
	"go/types"
	"testing"
)

// TestNextGenParserBasics 测试NextGenParser基础功能
func TestNextGenParserBasics(t *testing.T) {
	parser := NewNextGenParser("frame")

	// 测试解析器创建
	if parser == nil {
		t.Fatal("NextGenParser创建失败")
	}

	if parser.prefix != "frame" {
		t.Errorf("期望前缀为'frame'，实际为'%s'", parser.prefix)
	}

	// 测试缓存初始化
	if parser.packageCache == nil {
		t.Error("packageCache未初始化")
	}

	if parser.typeInfoCache == nil {
		t.Error("typeInfoCache未初始化")
	}
}

// TestTypeCompatibilityCheck 测试类型兼容性检查
func TestTypeCompatibilityCheck(t *testing.T) {
	parser := NewNextGenParser("frame")

	// 测试基本类型兼容性
	int64Type := types.Typ[types.Int64]
	int32Type := types.Typ[types.Int32]

	// int32应该可以赋值给int64（在某些情况下）
	err := parser.ValidateTypeCompatibility(int64Type, int32Type)
	if err == nil {
		t.Log("int32与int64兼容性检查通过")
	} else {
		t.Logf("int32与int64兼容性检查失败: %v", err)
	}

	// 相同类型应该兼容
	err = parser.ValidateTypeCompatibility(int64Type, int64Type)
	if err != nil {
		t.Errorf("相同类型应该兼容: %v", err)
	}
}

// TestTypeHelpers 测试类型辅助函数
func TestTypeHelpers(t *testing.T) {
	parser := NewNextGenParser("frame")

	// 测试指针类型检查
	int64Type := types.Typ[types.Int64]
	pointerType := types.NewPointer(int64Type)

	if !parser.IsPointerType(pointerType) {
		t.Error("应该识别为指针类型")
	}

	if parser.IsPointerType(int64Type) {
		t.Error("不应该识别为指针类型")
	}

	// 测试结构体类型检查
	structType := types.NewStruct([]*types.Var{}, []string{})
	namedStructType := types.NewNamed(
		types.NewTypeName(0, nil, "TestStruct", nil),
		structType,
		nil,
	)

	if !parser.IsStructType(namedStructType) {
		t.Error("应该识别为结构体类型")
	}

	// 测试GetStructType
	retrieved := parser.GetStructType(namedStructType)
	if retrieved == nil {
		t.Error("应该能够获取结构体类型")
	}

	// 测试指针解引用
	pointerToStruct := types.NewPointer(namedStructType)
	retrieved = parser.GetStructType(pointerToStruct)
	if retrieved == nil {
		t.Error("应该能够从指针获取结构体类型")
	}
}

// TestEnhancedParserContextCreation 测试增强的解析上下文创建
func TestEnhancedParserContextCreation(t *testing.T) {
	ctx := &EnhancedParserContext{
		PackageName: "testpkg",
		FileName:    "test.go",
		TypesInfo:   &types.Info{},
		Package:     types.NewPackage("testpkg", "testpkg"),
	}

	// 测试基本字段
	if ctx.PackageName != "testpkg" {
		t.Error("包名设置错误")
	}

	if ctx.FileName != "test.go" {
		t.Error("文件名设置错误")
	}

	if ctx.TypesInfo == nil {
		t.Error("TypesInfo应该不为nil")
	}

	if ctx.Package == nil {
		t.Error("Package应该不为nil")
	}
}

// TestMethodAndTypeContext 测试方法和类型上下文
func TestMethodAndTypeContext(t *testing.T) {
	// 测试方法上下文
	methodCtx := &MethodContext{
		IsPointer: true,
	}

	if !methodCtx.IsPointer {
		t.Error("方法上下文IsPointer设置错误")
	}

	// 测试类型上下文
	structType := types.NewStruct([]*types.Var{}, []string{})
	typeCtx := &TypeContext{
		StructType: structType,
	}

	if typeCtx.StructType == nil {
		t.Error("类型上下文StructType设置错误")
	}
}

// TestGenerationOptions 测试代码生成选项
func TestGenerationOptions(t *testing.T) {
	options := &GenerationOptions{
		EnableReflectionOptimization: true,
		EnableTypeValidation:         true,
		OutputFormat:                 "go",
	}

	if !options.EnableReflectionOptimization {
		t.Error("反射优化选项设置错误")
	}

	if !options.EnableTypeValidation {
		t.Error("类型验证选项设置错误")
	}

	if options.OutputFormat != "go" {
		t.Error("输出格式选项设置错误")
	}
}

// TestTypeStringRepresentation 测试类型字符串表示
func TestTypeStringRepresentation(t *testing.T) {
	parser := NewNextGenParser("frame")

	// 测试nil类型
	nilStr := parser.GetTypeString(nil)
	if nilStr != "<nil>" {
		t.Errorf("nil类型字符串应该是'<nil>'，实际为'%s'", nilStr)
	}

	// 测试基本类型
	int64Type := types.Typ[types.Int64]
	int64Str := parser.GetTypeString(int64Type)
	if int64Str != "int64" {
		t.Errorf("int64类型字符串应该是'int64'，实际为'%s'", int64Str)
	}

	// 测试指针类型
	pointerType := types.NewPointer(int64Type)
	pointerStr := parser.GetTypeString(pointerType)
	if pointerStr != "*int64" {
		t.Errorf("指针类型字符串应该是'*int64'，实际为'%s'", pointerStr)
	}
}

// TestInterfaceTypes 测试接口类型处理
func TestInterfaceTypes(t *testing.T) {
	parser := NewNextGenParser("frame")

	// 创建一个简单的接口类型
	methods := []*types.Func{}
	interfaceType := types.NewInterfaceType(methods, nil)
	namedInterface := types.NewNamed(
		types.NewTypeName(0, nil, "TestInterface", nil),
		interfaceType,
		nil,
	)

	// 测试接口类型识别
	if !parser.IsInterfaceType(namedInterface) {
		t.Error("应该识别为接口类型")
	}

	// 测试GetInterfaceType
	retrieved := parser.GetInterfaceType(namedInterface)
	if retrieved == nil {
		t.Error("应该能够获取接口类型")
	}
}

// BenchmarkTypeCompatibilityCheck 类型兼容性检查性能基准
func BenchmarkTypeCompatibilityCheck(b *testing.B) {
	parser := NewNextGenParser("frame")
	int64Type := types.Typ[types.Int64]
	int32Type := types.Typ[types.Int32]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.ValidateTypeCompatibility(int64Type, int32Type)
	}
}

// BenchmarkTypeHelpers 类型辅助函数性能基准
func BenchmarkTypeHelpers(b *testing.B) {
	parser := NewNextGenParser("frame")

	structType := types.NewStruct([]*types.Var{}, []string{})
	namedStructType := types.NewNamed(
		types.NewTypeName(0, nil, "TestStruct", nil),
		structType,
		nil,
	)
	pointerType := types.NewPointer(namedStructType)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.IsPointerType(pointerType)
		_ = parser.IsStructType(namedStructType)
		_ = parser.GetStructType(pointerType)
	}
}
