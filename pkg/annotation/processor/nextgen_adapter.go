// Package processor 提供NextGen解析器适配器
package processor

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"

	graintypes "github.com/isBlue-5/grain/pkg/annotation/types"
)

// NextGenAdapter NextGen解析器适配器，提供向下兼容的接口
type NextGenAdapter struct {
	// 新一代解析器
	nextGenParser NextGenParser

	// 原始解析器（向下兼容）
	legacyParser AnnotationParser

	// 是否启用NextGen功能
	enableNextGen bool
}

// NewNextGenAdapter 创建NextGen适配器
func NewNextGenAdapter(prefix string, enableNextGen bool) *NextGenAdapter {
	return &NextGenAdapter{
		nextGenParser: NewNextGenParser(prefix),
		legacyParser:  NewAnnotationParser(prefix),
		enableNextGen: enableNextGen,
	}
}

// ParseFile 解析文件（适配器方法）
func (a *NextGenAdapter) ParseFile(filename string) ([]graintypes.Annotation, error) {
	// 对于单个文件，直接使用传统解析器，因为NextGen主要针对包级别优化
	return a.legacyParser.ParseFile(filename)
}

// ParsePackage 解析包（适配器方法）
func (a *NextGenAdapter) ParsePackage(pkgPath string) ([]graintypes.Annotation, error) {
	if a.enableNextGen {
		// 使用NextGen解析器
		info, err := a.nextGenParser.ParsePackageWithTypes(pkgPath)
		if err != nil {
			return nil, err
		}
		return info.Annotations, nil
	}

	// 使用传统解析器
	return a.legacyParser.ParsePackage(pkgPath)
}

// ParseDir 解析目录（适配器方法）
func (a *NextGenAdapter) ParseDir(dirPath string) ([]graintypes.Annotation, error) {
	if a.enableNextGen {
		// NextGen解析器处理目录
		return a.parseDirectoryWithNextGen(dirPath)
	}

	// 使用传统解析器
	return a.legacyParser.ParseDir(dirPath)
}

// parseDirectoryWithNextGen 使用NextGen解析器处理目录
func (a *NextGenAdapter) parseDirectoryWithNextGen(dirPath string) ([]graintypes.Annotation, error) {
	// 使用packages.Load加载目录下的所有包
	info, err := a.nextGenParser.ParsePackageWithTypes(dirPath)
	if err != nil {
		return nil, err
	}
	return info.Annotations, nil
}

// GetTypeInfo 获取类型信息（新功能）
func (a *NextGenAdapter) GetTypeInfo(expr ast.Expr) types.Type {
	if a.enableNextGen {
		return a.nextGenParser.GetTypeInfo(expr)
	}
	return nil // 传统解析器不支持类型信息
}

// ValidateTypeCompatibility 验证类型兼容性（新功能）
func (a *NextGenAdapter) ValidateTypeCompatibility(expected, actual types.Type) error {
	if a.enableNextGen {
		return a.nextGenParser.ValidateTypeCompatibility(expected, actual)
	}
	return fmt.Errorf("类型兼容性检查需要启用NextGen功能")
}

// GetMethodSignature 获取方法签名（新功能）
func (a *NextGenAdapter) GetMethodSignature(method *ast.FuncDecl) (*types.Signature, error) {
	if a.enableNextGen {
		return a.nextGenParser.GetMethodSignature(method)
	}
	return nil, fmt.Errorf("方法签名获取需要启用NextGen功能")
}

// ResolveTypeAlias 解析类型别名（新功能）
func (a *NextGenAdapter) ResolveTypeAlias(typeName string) types.Type {
	if a.enableNextGen {
		return a.nextGenParser.ResolveTypeAlias(typeName)
	}
	return nil // 传统解析器不支持类型别名解析
}

// EnableNextGen 启用NextGen功能
func (a *NextGenAdapter) EnableNextGen(enable bool) {
	a.enableNextGen = enable
}

// IsNextGenEnabled 检查是否启用了NextGen功能
func (a *NextGenAdapter) IsNextGenEnabled() bool {
	return a.enableNextGen
}

// GetEnhancedPackageInfo 获取增强的包信息（仅NextGen支持）
func (a *NextGenAdapter) GetEnhancedPackageInfo(pkgPath string) (*EnhancedPackageInfo, error) {
	if !a.enableNextGen {
		return nil, fmt.Errorf("增强包信息需要启用NextGen功能")
	}
	return a.nextGenParser.ParsePackageWithTypes(pkgPath)
}

// TypeAwareAnnotationParser 类型感知注解解析器接口
// 扩展了原有的AnnotationParser接口，添加了类型感知功能
type TypeAwareAnnotationParser interface {
	AnnotationParser

	// 类型感知功能
	GetTypeInfo(expr ast.Expr) types.Type
	ValidateTypeCompatibility(expected, actual types.Type) error
	GetMethodSignature(method *ast.FuncDecl) (*types.Signature, error)
	ResolveTypeAlias(typeName string) types.Type

	// NextGen功能控制
	EnableNextGen(enable bool)
	IsNextGenEnabled() bool
	GetEnhancedPackageInfo(pkgPath string) (*EnhancedPackageInfo, error)
}

// 确保NextGenAdapter实现了TypeAwareAnnotationParser接口
var _ TypeAwareAnnotationParser = (*NextGenAdapter)(nil)

// ProcessorConfig 处理器配置
type ProcessorConfig struct {
	// 是否启用NextGen功能
	EnableNextGen bool

	// 是否启用类型验证
	EnableTypeValidation bool

	// 是否启用反射优化
	EnableReflectionOptimization bool

	// 输出格式
	OutputFormat string

	// 详细日志
	Verbose bool
}

// DefaultProcessorConfig 默认处理器配置
func DefaultProcessorConfig() *ProcessorConfig {
	return &ProcessorConfig{
		EnableNextGen:                true,  // 默认启用NextGen
		EnableTypeValidation:         true,  // 默认启用类型验证
		EnableReflectionOptimization: false, // 默认关闭反射优化（渐进式启用）
		OutputFormat:                 "go",  // 默认Go格式
		Verbose:                      false, // 默认关闭详细日志
	}
}

// ProcessorFactory 处理器工厂
type ProcessorFactory struct {
	config *ProcessorConfig
}

// NewProcessorFactory 创建处理器工厂
func NewProcessorFactory(config *ProcessorConfig) *ProcessorFactory {
	if config == nil {
		config = DefaultProcessorConfig()
	}
	return &ProcessorFactory{config: config}
}

// CreateParser 创建解析器
func (f *ProcessorFactory) CreateParser(prefix string) TypeAwareAnnotationParser {
	return NewNextGenAdapter(prefix, f.config.EnableNextGen)
}

// CreateParamBindingProcessor 创建参数绑定处理器
func (f *ProcessorFactory) CreateParamBindingProcessor(outputPath string) *ParamBindingProcessor {
	parser := f.CreateParser("frame")
	processor := &ParamBindingProcessor{
		parser:     parser,
		outputPath: outputPath,
		verbose:    f.config.Verbose,
	}
	return processor
}

// CreateEnhancedRouteProcessor 创建增强路由处理器
func (f *ProcessorFactory) CreateEnhancedRouteProcessor(outputPath string) *EnhancedRouteProcessor {
	parser := f.CreateParser("frame")
	processor := &EnhancedRouteProcessor{
		parser:     parser,
		outputPath: outputPath,
		verbose:    f.config.Verbose,
	}
	return processor
}

// UpdateProcessor 更新处理器使用NextGen功能
func (f *ProcessorFactory) UpdateProcessor(processor interface{}) error {
	parser := f.CreateParser("frame")
	return UpdateProcessorWithNextGen(processor, parser)
}

// UpdateProcessorWithNextGen 更新处理器使用NextGen功能
// 这是一个渐进式迁移的辅助函数，根据处理器类型更新其解析器字段
func UpdateProcessorWithNextGen(processor interface{}, parser TypeAwareAnnotationParser) error {
	if processor == nil {
		return fmt.Errorf("处理器不能为nil")
	}
	if parser == nil {
		return fmt.Errorf("解析器不能为nil")
	}

	// 根据处理器类型进行字段更新
	switch p := processor.(type) {
	case *ParamBindingProcessor:
		// ParamBindingProcessor有parser字段
		p.parser = parser
		fmt.Printf("已将ParamBindingProcessor更新为使用NextGen解析器\n")
		return nil

	case *EnhancedRouteProcessor:
		// EnhancedRouteProcessor有parser字段
		p.parser = parser
		fmt.Printf("已将EnhancedRouteProcessor更新为使用NextGen解析器\n")
		return nil

	case *RouteProcessor:
		// RouteProcessor没有parser字段，但可以通过工厂方法重新创建
		// 这里我们记录一个警告，因为RouteProcessor需要特殊处理
		fmt.Printf("警告: RouteProcessor不支持动态更新解析器，建议使用ProcessorFactory重新创建\n")
		return nil

	case *CRUDProcessor:
		// CRUDProcessor没有parser字段，记录警告
		fmt.Printf("警告: CRUDProcessor不支持动态更新解析器，建议使用ProcessorFactory重新创建\n")
		return nil

	case *InjectProcessor:
		// InjectProcessor没有parser字段，记录警告
		fmt.Printf("警告: InjectProcessor不支持动态更新解析器，建议使用ProcessorFactory重新创建\n")
		return nil

	default:
		// 尝试使用反射进行通用处理
		return updateProcessorWithReflection(processor, parser)
	}
}

// updateProcessorWithReflection 使用反射更新处理器的parser字段
func updateProcessorWithReflection(processor interface{}, parser TypeAwareAnnotationParser) error {
	v := reflect.ValueOf(processor)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("处理器必须是指针类型")
	}

	// 获取结构体元素
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("处理器必须是结构体指针")
	}

	// 查找parser字段
	parserField := elem.FieldByName("parser")
	if !parserField.IsValid() {
		// 尝试查找Parser字段（首字母大写）
		parserField = elem.FieldByName("Parser")
		if !parserField.IsValid() {
			return fmt.Errorf("处理器不包含parser或Parser字段")
		}
	}

	// 检查字段是否可设置
	if !parserField.CanSet() {
		return fmt.Errorf("parser字段不可设置")
	}

	// 检查类型兼容性
	parserValue := reflect.ValueOf(parser)
	if !parserValue.Type().AssignableTo(parserField.Type()) {
		return fmt.Errorf("parser类型不兼容: 期望 %v, 得到 %v", parserField.Type(), parserValue.Type())
	}

	// 设置字段值
	parserField.Set(parserValue)
	fmt.Printf("已通过反射将 %T 更新为使用NextGen解析器\n", processor)
	return nil
}
