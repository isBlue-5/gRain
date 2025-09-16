// Package processor 提供下一代注解处理器的实现
package processor

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"

	graintypes "github.com/isBlue-5/grain/pkg/annotation/types"
)

// NextGenParser 下一代类型感知注解解析器接口
type NextGenParser interface {
	// ParsePackageWithTypes 使用类型信息解析包中的注解
	ParsePackageWithTypes(pkgPath string) (*EnhancedPackageInfo, error)

	// GetTypeInfo 获取表达式的类型信息
	GetTypeInfo(expr ast.Expr) types.Type

	// ValidateTypeCompatibility 验证类型兼容性
	ValidateTypeCompatibility(expected, actual types.Type) error

	// GetMethodSignature 获取方法签名
	GetMethodSignature(method *ast.FuncDecl) (*types.Signature, error)

	// ResolveTypeAlias 解析类型别名
	ResolveTypeAlias(typeName string) types.Type
}

// EnhancedPackageInfo 增强的包信息，包含类型信息
type EnhancedPackageInfo struct {
	// 标准包信息
	Package *packages.Package

	// 类型信息
	TypesInfo *types.Info

	// 注解列表
	Annotations []graintypes.Annotation

	// 依赖关系图
	Dependencies map[string]*packages.Package

	// 类型映射
	TypeMapping map[string]types.Type
}

// DefaultNextGenParser 默认的下一代解析器实现
type DefaultNextGenParser struct {
	// 包配置
	config *packages.Config

	// 包缓存
	packageCache map[string]*packages.Package

	// 类型信息缓存
	typeInfoCache map[string]*types.Info

	// 注解前缀
	prefix string

	// 原始解析器（用于向下兼容）
	legacyParser *DefaultAnnotationParser
}

// NewNextGenParser 创建新的下一代解析器
func NewNextGenParser(prefix string) *DefaultNextGenParser {
	config := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedTypesSizes |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
		Tests: false, // 不加载测试文件
	}

	return &DefaultNextGenParser{
		config:        config,
		packageCache:  make(map[string]*packages.Package),
		typeInfoCache: make(map[string]*types.Info),
		prefix:        prefix,
		legacyParser:  NewAnnotationParser(prefix),
	}
}

// ParsePackageWithTypes 使用类型信息解析包中的注解
func (p *DefaultNextGenParser) ParsePackageWithTypes(pkgPath string) (*EnhancedPackageInfo, error) {
	// 检查缓存
	if pkg, exists := p.packageCache[pkgPath]; exists {
		return p.buildEnhancedInfo(pkg)
	}

	// 加载包
	pkgs, err := packages.Load(p.config, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("加载包失败: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("未找到包: %s", pkgPath)
	}

	pkg := pkgs[0]

	// 检查包错误
	if len(pkg.Errors) > 0 {
		for _, err := range pkg.Errors {
			fmt.Printf("包错误: %v\n", err)
		}
		// 继续处理，即使有错误
	}

	// 缓存包信息
	p.packageCache[pkgPath] = pkg
	p.typeInfoCache[pkgPath] = pkg.TypesInfo

	return p.buildEnhancedInfo(pkg)
}

// buildEnhancedInfo 构建增强的包信息
func (p *DefaultNextGenParser) buildEnhancedInfo(pkg *packages.Package) (*EnhancedPackageInfo, error) {
	info := &EnhancedPackageInfo{
		Package:      pkg,
		TypesInfo:    pkg.TypesInfo,
		Dependencies: make(map[string]*packages.Package),
		TypeMapping:  make(map[string]types.Type),
	}

	// 构建依赖映射
	for importPath, importPkg := range pkg.Imports {
		info.Dependencies[importPath] = importPkg
	}

	// 构建类型映射
	if pkg.TypesInfo != nil {
		for ident, obj := range pkg.TypesInfo.Defs {
			if obj != nil && ident.Name != "" {
				info.TypeMapping[ident.Name] = obj.Type()
			}
		}
	}

	// 解析注解
	annotations, err := p.extractAnnotationsWithTypes(pkg)
	if err != nil {
		return nil, fmt.Errorf("提取注解失败: %w", err)
	}

	info.Annotations = annotations

	return info, nil
}

// extractAnnotationsWithTypes 使用类型信息提取注解
func (p *DefaultNextGenParser) extractAnnotationsWithTypes(pkg *packages.Package) ([]graintypes.Annotation, error) {
	var annotations []graintypes.Annotation

	// 为每个语法文件提取注解
	for _, file := range pkg.Syntax {
		fileAnnotations := p.extractFromFile(file, pkg)
		annotations = append(annotations, fileAnnotations...)
	}

	return annotations, nil
}

// extractFromFile 从单个文件提取注解
func (p *DefaultNextGenParser) extractFromFile(file *ast.File, pkg *packages.Package) []graintypes.Annotation {
	var annotations []graintypes.Annotation

	// 创建增强的解析上下文
	ctx := &EnhancedParserContext{
		PackageName: file.Name.Name,
		FileName:    pkg.Fset.Position(file.Pos()).Filename,
		Imports:     extractImportsWithTypes(file, pkg),
		TypesInfo:   pkg.TypesInfo,
		Package:     pkg.Types,
		TypeChecker: nil, // 可以后续添加
	}

	// 遍历AST节点
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.FuncDecl:
			// 处理函数/方法注解
			if n.Doc != nil {
				for _, comment := range n.Doc.List {
					if ann := p.parseCommentWithTypes(comment, n, ctx); ann != nil {
						annotations = append(annotations, ann)
					}
				}
			}

		case *ast.GenDecl:
			// 处理类型声明注解
			if n.Doc != nil {
				for _, comment := range n.Doc.List {
					if ann := p.parseCommentWithTypes(comment, n, ctx); ann != nil {
						annotations = append(annotations, ann)
					}
				}
			}

		case *ast.StructType:
			// 处理结构体字段注解
			if n.Fields != nil {
				for _, field := range n.Fields.List {
					if field.Tag != nil {
						tagAnnotations := p.parseStructTagWithTypes(field.Tag.Value, field, n, ctx)
						annotations = append(annotations, tagAnnotations...)
					}
				}
			}
		}

		return true
	})

	return annotations
}

// EnhancedParserContext 增强的解析上下文
type EnhancedParserContext struct {
	// 基础信息
	PackageName string
	FileName    string
	ModulePath  string

	// 类型系统信息
	TypesInfo   *types.Info
	Package     *types.Package
	TypeChecker *types.Checker

	// 依赖关系
	Imports      map[string]*packages.Package
	Dependencies []string

	// 注解上下文
	MethodContext *MethodContext
	TypeContext   *TypeContext

	// 生成选项
	GenerationOptions *GenerationOptions
}

// MethodContext 方法上下文
type MethodContext struct {
	ReceiverType types.Type
	Method       *types.Func
	Signature    *types.Signature
	IsPointer    bool
}

// TypeContext 类型上下文
type TypeContext struct {
	Type          types.Type
	StructType    *types.Struct
	InterfaceType *types.Interface
	Methods       []*types.Func
}

// GenerationOptions 代码生成选项
type GenerationOptions struct {
	EnableReflectionOptimization bool
	EnableTypeValidation         bool
	OutputFormat                 string
}

// GetTypeInfo 获取表达式的类型信息
func (p *DefaultNextGenParser) GetTypeInfo(expr ast.Expr) types.Type {
	// 遍历所有缓存的类型信息
	for _, typeInfo := range p.typeInfoCache {
		if typ := typeInfo.TypeOf(expr); typ != nil {
			return typ
		}
	}
	return nil
}

// GetTypeInfoInContext 在特定上下文中获取表达式的类型信息
func (p *DefaultNextGenParser) GetTypeInfoInContext(expr ast.Expr, ctx *EnhancedParserContext) types.Type {
	if ctx.TypesInfo != nil {
		if typ := ctx.TypesInfo.TypeOf(expr); typ != nil {
			return typ
		}
	}

	// 回退到全局查找
	return p.GetTypeInfo(expr)
}

// ValidateTypeCompatibility 验证类型兼容性
func (p *DefaultNextGenParser) ValidateTypeCompatibility(expected, actual types.Type) error {
	// 检查nil类型
	if expected == nil || actual == nil {
		return fmt.Errorf("类型不兼容: 无法验证nil类型")
	}

	if !types.AssignableTo(actual, expected) {
		return fmt.Errorf("类型不兼容: %s 不能赋值给 %s", actual.String(), expected.String())
	}
	return nil
}

// IsPointerType 检查是否为指针类型
func (p *DefaultNextGenParser) IsPointerType(typ types.Type) bool {
	_, ok := typ.(*types.Pointer)
	return ok
}

// IsInterfaceType 检查是否为接口类型
func (p *DefaultNextGenParser) IsInterfaceType(typ types.Type) bool {
	_, ok := typ.Underlying().(*types.Interface)
	return ok
}

// IsStructType 检查是否为结构体类型
func (p *DefaultNextGenParser) IsStructType(typ types.Type) bool {
	_, ok := typ.Underlying().(*types.Struct)
	return ok
}

// GetMethodSignature 获取方法签名
func (p *DefaultNextGenParser) GetMethodSignature(method *ast.FuncDecl) (*types.Signature, error) {
	// 检查输入参数
	if method == nil || method.Name == nil {
		return nil, fmt.Errorf("无效的方法声明")
	}

	// 需要从TypesInfo中获取
	for _, typeInfo := range p.typeInfoCache {
		if obj := typeInfo.Defs[method.Name]; obj != nil {
			if fn, ok := obj.(*types.Func); ok {
				if sig, ok := fn.Type().(*types.Signature); ok {
					return sig, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("无法获取方法 %s 的签名", method.Name.Name)
}

// GetMethodSignatureInContext 在特定上下文中获取方法签名
func (p *DefaultNextGenParser) GetMethodSignatureInContext(method *ast.FuncDecl, ctx *EnhancedParserContext) (*types.Signature, error) {
	if ctx.TypesInfo != nil {
		if obj := ctx.TypesInfo.Defs[method.Name]; obj != nil {
			if fn, ok := obj.(*types.Func); ok {
				if sig, ok := fn.Type().(*types.Signature); ok {
					return sig, nil
				}
			}
		}
	}

	// 回退到全局查找
	return p.GetMethodSignature(method)
}

// ResolveTypeAlias 解析类型别名
func (p *DefaultNextGenParser) ResolveTypeAlias(typeName string) types.Type {
	// 在所有包的类型信息中查找
	for _, pkg := range p.packageCache {
		if pkg.TypesInfo != nil {
			for ident, obj := range pkg.TypesInfo.Defs {
				if obj != nil && ident.Name == typeName {
					if typeName, ok := obj.(*types.TypeName); ok {
						return typeName.Type()
					}
				}
			}
		}
	}
	return nil
}

// ResolveTypeAliasInPackage 在特定包中解析类型别名
func (p *DefaultNextGenParser) ResolveTypeAliasInPackage(typeName string, pkgPath string) types.Type {
	if pkg, exists := p.packageCache[pkgPath]; exists && pkg.TypesInfo != nil {
		// 在指定包中查找
		for ident, obj := range pkg.TypesInfo.Defs {
			if ident.Name == typeName && obj != nil {
				if typeDef, ok := obj.(*types.TypeName); ok {
					return typeDef.Type()
				}
			}
		}
	}

	return nil
}

// ResolveEmbeddedTypes 解析嵌入类型
func (p *DefaultNextGenParser) ResolveEmbeddedTypes(structType *types.Struct) []types.Type {
	var embeddedTypes []types.Type

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		if field.Embedded() {
			embeddedTypes = append(embeddedTypes, field.Type())
		}
	}

	return embeddedTypes
}

// ResolveCrossPackageType 解析跨包类型
func (p *DefaultNextGenParser) ResolveCrossPackageType(typeName string, importPath string) types.Type {
	// 检查是否已加载该包
	for pkgPath, pkg := range p.packageCache {
		if strings.Contains(pkgPath, importPath) && pkg.TypesInfo != nil {
			// 在该包中查找类型
			for ident, obj := range pkg.TypesInfo.Defs {
				if ident.Name == typeName && obj != nil {
					if typeDef, ok := obj.(*types.TypeName); ok {
						return typeDef.Type()
					}
				}
			}
		}
	}

	// 如果未找到，尝试加载包
	config := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedTypesInfo,
	}

	pkgs, err := packages.Load(config, importPath)
	if err != nil || len(pkgs) == 0 {
		return nil
	}

	pkg := pkgs[0]

	// 缓存新加载的包
	p.packageCache[importPath] = pkg
	if pkg.TypesInfo != nil {
		p.typeInfoCache[importPath] = pkg.TypesInfo
	}

	// 在新加载的包中查找类型
	if pkg.TypesInfo != nil {
		for ident, obj := range pkg.TypesInfo.Defs {
			if ident.Name == typeName && obj != nil {
				if typeDef, ok := obj.(*types.TypeName); ok {
					return typeDef.Type()
				}
			}
		}
	}

	return nil
}

// extractImportsWithTypes 提取带类型信息的导入
func extractImportsWithTypes(file *ast.File, pkg *packages.Package) map[string]*packages.Package {
	imports := make(map[string]*packages.Package)

	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if importPkg, exists := pkg.Imports[path]; exists {
			// 处理命名导入
			if imp.Name != nil {
				imports[imp.Name.Name] = importPkg
			} else {
				// 使用包名作为键
				imports[importPkg.Name] = importPkg
			}
		}
	}

	return imports
}

// parseCommentWithTypes 使用类型信息解析注释注解
func (p *DefaultNextGenParser) parseCommentWithTypes(comment *ast.Comment, node ast.Node, ctx *EnhancedParserContext) graintypes.Annotation {
	// 首先使用传统解析器解析基本信息
	baseAnnotation := p.legacyParser.parseComment(comment, node, &parserContext{
		pkgName:  ctx.PackageName,
		fileName: ctx.FileName,
		imports:  nil, // 暂时使用nil，后续完善
	})

	if baseAnnotation == nil {
		return nil
	}

	// 使用类型信息增强注解
	enhancedAnnotation := p.enhanceAnnotationWithTypes(baseAnnotation, ctx)

	return enhancedAnnotation
}

// parseStructTagWithTypes 使用类型信息解析结构体标签
func (p *DefaultNextGenParser) parseStructTagWithTypes(tag string, field *ast.Field, structType *ast.StructType, ctx *EnhancedParserContext) []graintypes.Annotation {
	// 使用传统解析器解析基本信息
	baseAnnotations := p.legacyParser.parseStructTag(tag, field, structType)

	// 使用类型信息增强每个注解
	var enhancedAnnotations []graintypes.Annotation
	for _, baseAnn := range baseAnnotations {
		enhanced := p.enhanceAnnotationWithTypes(baseAnn, ctx)
		if enhanced != nil {
			enhancedAnnotations = append(enhancedAnnotations, enhanced)
		}
	}

	return enhancedAnnotations
}

// enhanceAnnotationWithTypes 使用类型信息增强注解
func (p *DefaultNextGenParser) enhanceAnnotationWithTypes(baseAnnotation graintypes.Annotation, ctx *EnhancedParserContext) graintypes.Annotation {
	// 创建增强的注解副本
	switch ann := baseAnnotation.(type) {
	case *graintypes.CommentAnnotation:
		enhanced := *ann // 复制

		// 添加类型信息
		if ctx.TypesInfo != nil {
			if funcDecl, ok := ann.Node.(*ast.FuncDecl); ok {
				// 为方法注解添加精确的类型信息
				if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
					if receiverType := ctx.TypesInfo.TypeOf(funcDecl.Recv.List[0].Type); receiverType != nil {
						enhanced.ReceiverType = receiverType.String()

						// 添加方法上下文
						if signature := ctx.TypesInfo.TypeOf(funcDecl.Name); signature != nil {
							if sig, ok := signature.(*types.Signature); ok {
								enhanced.MethodSignature = sig
							}
						}
					}
				}
			}
		}

		return &enhanced

	case *graintypes.StructTagAnnotation:
		enhanced := *ann // 复制

		// 添加字段类型信息
		if ctx.TypesInfo != nil && ann.StructField != nil {
			if fieldType := ctx.TypesInfo.TypeOf(ann.StructField.Type); fieldType != nil {
				enhanced.FieldType = fieldType
			}
		}

		return &enhanced

	default:
		return baseAnnotation
	}
}

// GetStructType 获取结构体类型（如果是指针，则解引用）
func (p *DefaultNextGenParser) GetStructType(typ types.Type) *types.Struct {
	// 如果是指针，解引用
	if ptr, ok := typ.(*types.Pointer); ok {
		typ = ptr.Elem()
	}

	// 获取底层类型
	if structType, ok := typ.Underlying().(*types.Struct); ok {
		return structType
	}

	return nil
}

// GetInterfaceType 获取接口类型
func (p *DefaultNextGenParser) GetInterfaceType(typ types.Type) *types.Interface {
	if interfaceType, ok := typ.Underlying().(*types.Interface); ok {
		return interfaceType
	}
	return nil
}

// GetPointerElementType 获取指针指向的元素类型
func (p *DefaultNextGenParser) GetPointerElementType(typ types.Type) types.Type {
	if ptr, ok := typ.(*types.Pointer); ok {
		return ptr.Elem()
	}
	return typ
}

// IsSliceType 检查是否为切片类型
func (p *DefaultNextGenParser) IsSliceType(typ types.Type) bool {
	_, ok := typ.Underlying().(*types.Slice)
	return ok
}

// IsMapType 检查是否为映射类型
func (p *DefaultNextGenParser) IsMapType(typ types.Type) bool {
	_, ok := typ.Underlying().(*types.Map)
	return ok
}

// IsChannelType 检查是否为通道类型
func (p *DefaultNextGenParser) IsChannelType(typ types.Type) bool {
	_, ok := typ.Underlying().(*types.Chan)
	return ok
}

// GetSliceElementType 获取切片元素类型
func (p *DefaultNextGenParser) GetSliceElementType(typ types.Type) types.Type {
	if slice, ok := typ.Underlying().(*types.Slice); ok {
		return slice.Elem()
	}
	return nil
}

// GetMapKeyValueTypes 获取映射的键值类型
func (p *DefaultNextGenParser) GetMapKeyValueTypes(typ types.Type) (key, value types.Type) {
	if mapType, ok := typ.Underlying().(*types.Map); ok {
		return mapType.Key(), mapType.Elem()
	}
	return nil, nil
}

// ImplementsInterface 检查类型是否实现了指定接口
func (p *DefaultNextGenParser) ImplementsInterface(typ, interfaceType types.Type) bool {
	return types.Implements(typ, interfaceType.Underlying().(*types.Interface))
}

// GetMethodSet 获取类型的方法集
func (p *DefaultNextGenParser) GetMethodSet(typ types.Type) *types.MethodSet {
	return types.NewMethodSet(typ)
}

// FindMethodByName 在类型中查找指定名称的方法
func (p *DefaultNextGenParser) FindMethodByName(typ types.Type, methodName string) *types.Func {
	methodSet := p.GetMethodSet(typ)
	for i := 0; i < methodSet.Len(); i++ {
		method := methodSet.At(i).Obj().(*types.Func)
		if method.Name() == methodName {
			return method
		}
	}
	return nil
}

// GetFieldByName 在结构体中查找指定名称的字段
func (p *DefaultNextGenParser) GetFieldByName(structType *types.Struct, fieldName string) *types.Var {
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		if field.Name() == fieldName {
			return field
		}
	}
	return nil
}

// GetStructTags 获取结构体字段的标签
func (p *DefaultNextGenParser) GetStructTags(structType *types.Struct, fieldIndex int) string {
	if fieldIndex >= 0 && fieldIndex < structType.NumFields() {
		return structType.Tag(fieldIndex)
	}
	return ""
}

// IsBasicType 检查是否为基础类型
func (p *DefaultNextGenParser) IsBasicType(typ types.Type) bool {
	_, ok := typ.Underlying().(*types.Basic)
	return ok
}

// GetBasicTypeKind 获取基础类型的种类
func (p *DefaultNextGenParser) GetBasicTypeKind(typ types.Type) types.BasicKind {
	if basic, ok := typ.Underlying().(*types.Basic); ok {
		return basic.Kind()
	}
	return types.Invalid
}

// IsNumericType 检查是否为数值类型
func (p *DefaultNextGenParser) IsNumericType(typ types.Type) bool {
	if basic, ok := typ.Underlying().(*types.Basic); ok {
		info := basic.Info()
		return (info & types.IsNumeric) != 0
	}
	return false
}

// IsStringType 检查是否为字符串类型
func (p *DefaultNextGenParser) IsStringType(typ types.Type) bool {
	if basic, ok := typ.Underlying().(*types.Basic); ok {
		return basic.Kind() == types.String
	}
	return false
}

// IsBoolType 检查是否为布尔类型
func (p *DefaultNextGenParser) IsBoolType(typ types.Type) bool {
	if basic, ok := typ.Underlying().(*types.Basic); ok {
		return basic.Kind() == types.Bool
	}
	return false
}

// GetTypeString 获取类型的字符串表示
func (p *DefaultNextGenParser) GetTypeString(typ types.Type) string {
	if typ == nil {
		return "<nil>"
	}
	return typ.String()
}

// GetUnderlyingType 获取类型的底层类型
func (p *DefaultNextGenParser) GetUnderlyingType(typ types.Type) types.Type {
	return typ.Underlying()
}
