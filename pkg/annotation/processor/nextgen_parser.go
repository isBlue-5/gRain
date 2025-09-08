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
	// 需要上下文信息，这里简化实现
	// 实际使用时需要传入正确的TypesInfo
	return nil
}

// ValidateTypeCompatibility 验证类型兼容性
func (p *DefaultNextGenParser) ValidateTypeCompatibility(expected, actual types.Type) error {
	if !types.AssignableTo(actual, expected) {
		return fmt.Errorf("类型不兼容: %s 不能赋值给 %s", actual.String(), expected.String())
	}
	return nil
}

// GetMethodSignature 获取方法签名
func (p *DefaultNextGenParser) GetMethodSignature(method *ast.FuncDecl) (*types.Signature, error) {
	// 需要从TypesInfo中获取
	// 这里需要完善实现
	return nil, nil
}

// ResolveTypeAlias 解析类型别名
func (p *DefaultNextGenParser) ResolveTypeAlias(typeName string) types.Type {
	// 需要从当前包的类型信息中解析
	// 这里需要完善实现
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
