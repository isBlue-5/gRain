// Package processor 提供注解处理器实现，用于生成代码
package processor

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/grain-framework/grain/pkg/annotation/registry"
	"github.com/grain-framework/grain/pkg/annotation/types"
)

// CacheProcessor 缓存注解处理器
type CacheProcessor struct {
	registry   registry.Registry
	parser     AnnotationParser
	outputPath string
	tmpl       *template.Template
	verbose    bool
}

// 缓存包装模板
const cacheTmpl = `
// 由缓存注解处理器自动生成
package {{.Package}}

import (
	"context"
	"fmt"
	"time"
	"encoding/json"

	"github.com/grain-framework/grain/pkg/cache"
)

{{if .IsMethod}}
// {{.WrappedName}} 是{{.OriginalName}}的缓存包装方法
func ({{.ReceiverName}} {{.ReceiverType}}) {{.WrappedName}}({{.Params}}) ({{.Returns}}) {
	// 获取缓存管理器
	cacheManager := {{.ReceiverName}}.{{.CacheManagerField}}
	if cacheManager == nil {
		// 如果缓存管理器未设置，直接调用原始方法
		return {{.ReceiverName}}.{{.OriginalName}}({{.CallArgs}})
	}

	// 获取缓存实例
	cacheInstance := cacheManager.GetCache("{{.CacheName}}")
	if cacheInstance == nil {
		// 创建缓存实例
		var err error
		cacheInstance, err = cacheManager.CreateCache("{{.CacheName}}", 
			cache.WithPrefix("{{.CachePrefix}}"),
			cache.WithDefaultTTL({{.TTL}}),
		)
		if err != nil {
			// 创建缓存失败，直接调用原始方法
			return {{.ReceiverName}}.{{.OriginalName}}({{.CallArgs}})
		}
	}

	// 生成缓存键
	cacheKey := fmt.Sprintf("{{.KeyPrefix}}{{if .KeyGenerator}}%s{{else}}{{range $i, $param := .KeyParams}}{{if $i}}:%v{{else}}%v{{end}}{{end}}{{end}}"{{if .KeyGenerator}}, {{.ReceiverName}}.{{.KeyGenerator}}({{.KeyGenArgs}}){{else}}{{range .KeyParams}}, {{.}}{{end}}{{end}})

	// 尝试从缓存获取
	{{if .HasReturn}}
	var result {{.ReturnType}}
	err := cacheInstance.Get(ctx, cacheKey, &result)
	if err == nil {
		// 缓存命中
		{{if .HasError}}
		return result, nil
		{{else}}
		return result
		{{end}}
	}
	{{end}}

	// 缓存未命中，调用原始方法
	{{if .HasReturn}}result, {{end}}{{if .HasError}}err{{else}}_{{end}} := {{.ReceiverName}}.{{.OriginalName}}({{.CallArgs}})

	{{if .HasError}}
	if err != nil {
		// 方法调用失败，不缓存错误结果
		return {{if .HasReturn}}result, {{end}}err
	}
	{{end}}

	{{if .HasReturn}}
	// 缓存结果
	{{if .Condition}}
	if {{.ReceiverName}}.{{.Condition}}({{.ConditionArgs}}) {
		cacheInstance.Set(ctx, cacheKey, result, {{.TTL}})
	}
	{{else}}
	cacheInstance.Set(ctx, cacheKey, result, {{.TTL}})
	{{end}}
	{{end}}

	// 返回结果
	return {{if .HasReturn}}result{{if .HasError}}, nil{{end}}{{end}}
}
{{else}}
// {{.WrappedName}} 是{{.OriginalName}}的缓存包装函数
func {{.WrappedName}}({{.Params}}) ({{.Returns}}) {
	// 函数级缓存包装实现（与方法类似，但无接收者）
	cacheKey := fmt.Sprintf("{{.CacheKey}}", {{range $i, $param := .Params}}{{if $i}}, {{end}}{{$param}}{{end}})
	
	// 尝试从缓存获取
	if cached, err := cache.Get(context.Background(), cacheKey); err == nil {
		// 缓存命中，返回缓存值
		return cached.({{.Returns}})
	}
	
	// 缓存未命中，执行原函数
	result := {{.OriginalName}}({{range $i, $param := .Params}}{{if $i}}, {{end}}{{$param}}{{end}})
	
	// 将结果存入缓存
	cache.Set(context.Background(), cacheKey, result, {{.TTL}})
	
	return result
}
{{end}}

// {{.EvictMethodName}} 清除{{.OriginalName}}的缓存
{{if .IsMethod}}
func ({{.ReceiverName}} {{.ReceiverType}}) {{.EvictMethodName}}({{.EvictParams}}) {
	// 获取缓存管理器
	cacheManager := {{.ReceiverName}}.{{.CacheManagerField}}
	if cacheManager == nil {
		return
	}

	// 获取缓存实例
	cacheInstance := cacheManager.GetCache("{{.CacheName}}")
	if cacheInstance == nil {
		return
	}

	// 生成缓存键
	cacheKey := fmt.Sprintf("{{.KeyPrefix}}{{if .KeyGenerator}}%s{{else}}{{range $i, $param := .KeyParams}}{{if $i}}:%v{{else}}%v{{end}}{{end}}{{end}}"{{if .KeyGenerator}}, {{.ReceiverName}}.{{.KeyGenerator}}({{.KeyGenArgs}}){{else}}{{range .KeyParams}}, {{.}}{{end}}{{end}})

	// 删除缓存
	cacheInstance.Delete(context.Background(), cacheKey)
}
{{else}}
func {{.EvictMethodName}}({{.EvictParams}}) {
	// 函数级缓存清除实现
	cacheKey := fmt.Sprintf("{{.CacheKey}}", {{range $i, $param := .EvictParams}}{{if $i}}, {{end}}{{$param}}{{end}})
	
	// 清除缓存
	cache.Delete(context.Background(), cacheKey)
}
{{end}}
`

// NewCacheProcessor 创建缓存注解处理器
func NewCacheProcessor(reg registry.Registry, parser AnnotationParser, outputPath string) *CacheProcessor {
	tmpl, err := template.New("cache").Parse(cacheTmpl)
	if err != nil {
		panic(fmt.Errorf("解析缓存模板失败: %w", err))
	}

	return &CacheProcessor{
		registry:   reg,
		parser:     parser,
		outputPath: outputPath,
		tmpl:       tmpl,
		verbose:    false,
	}
}

// WithVerbose 设置是否启用详细日志
func (p *CacheProcessor) WithVerbose(verbose bool) *CacheProcessor {
	p.verbose = verbose
	return p
}

// ProcessCache 处理缓存注解
func (p *CacheProcessor) ProcessCache(paths []string) error {
	// 获取所有缓存注解
	var annotations []types.Annotation

	// 解析所有路径下的注解
	for _, path := range paths {
		anns, err := p.parser.ParseDir(path)
		if err != nil {
			return fmt.Errorf("解析目录 %s 失败: %w", path, err)
		}

		for _, ann := range anns {
			if ann.GetType() == types.CacheType {
				annotations = append(annotations, ann)
			}
		}
	}

	if p.verbose {
		fmt.Printf("找到 %d 个缓存注解\n", len(annotations))
	}

	// 按包进行分组处理
	packageMap := make(map[string][]types.CommentAnnotation)
	for _, ann := range annotations {
		if commentAnn, ok := ann.(*types.CommentAnnotation); ok {
			pkgName := filepath.Base(filepath.Dir(commentAnn.Position.Filename))
			packageMap[pkgName] = append(packageMap[pkgName], *commentAnn)
		}
	}

	// 处理每个包的缓存注解
	for pkgName, anns := range packageMap {
		if err := p.processPackage(pkgName, anns); err != nil {
			return err
		}
	}

	return nil
}

// processPackage 处理单个包的缓存注解
func (p *CacheProcessor) processPackage(pkgName string, annotations []types.CommentAnnotation) error {
	for _, ann := range annotations {
		if ann.TargetType == types.MethodTarget {
			if err := p.processMethodAnnotation(pkgName, ann); err != nil {
				return err
			}
		} else if ann.TargetType == types.FunctionTarget {
			if err := p.processFunctionAnnotation(pkgName, ann); err != nil {
				return err
			}
		}
	}
	return nil
}

// processMethodAnnotation 处理方法上的缓存注解
func (p *CacheProcessor) processMethodAnnotation(pkgName string, ann types.CommentAnnotation) error {
	// 提取注解参数
	cacheName := types.GetStringAttribute(ann.Attributes, "name", pkgName)
	cachePrefix := types.GetStringAttribute(ann.Attributes, "prefix", "")
	keyPrefix := types.GetStringAttribute(ann.Attributes, "key", ann.TargetName)
	ttl := types.GetStringAttribute(ann.Attributes, "ttl", "5*time.Minute")
	keyGenerator := types.GetStringAttribute(ann.Attributes, "keyGenerator", "")
	condition := types.GetStringAttribute(ann.Attributes, "condition", "")
	cacheManagerField := types.GetStringAttribute(ann.Attributes, "cacheManager", "cacheManager")

	// 提取键参数
	keyParams, _ := types.GetStringSliceAttribute(ann.Attributes, "keyParams")

	// 分析方法
	methodDecl, ok := ann.Node.(*ast.FuncDecl)
	if !ok {
		return fmt.Errorf("缓存注解只能应用于方法或函数: %s", ann.TargetName)
	}

	// 提取方法签名信息
	signatureInfo, err := extractMethodSignature(methodDecl)
	if err != nil {
		return fmt.Errorf("提取方法签名失败: %w", err)
	}

	// 获取原始方法名和包装方法名
	originalName := methodDecl.Name.Name
	wrappedName := originalName
	originalName = originalName + "Impl" // 原始方法将被重命名
	evictMethodName := "Evict" + wrappedName

	// 如果未指定键参数，使用所有参数作为键
	if len(keyParams) == 0 && keyGenerator == "" {
		// 获取所有参数名
		keyParams = extractParamNames(methodDecl.Type.Params)
	}

	// 创建模板数据
	tmplData := map[string]interface{}{
		"Package":           pkgName,
		"WrappedName":       wrappedName,
		"OriginalName":      originalName,
		"EvictMethodName":   evictMethodName,
		"CacheName":         cacheName,
		"CachePrefix":       cachePrefix,
		"KeyPrefix":         keyPrefix,
		"TTL":               ttl,
		"KeyParams":         keyParams,
		"KeyGenerator":      keyGenerator,
		"KeyGenArgs":        strings.Join(extractParamNames(methodDecl.Type.Params), ", "),
		"Condition":         condition,
		"ConditionArgs":     strings.Join(extractParamNames(methodDecl.Type.Params), ", "),
		"CacheManagerField": cacheManagerField,
		"EvictParams":       extractEvictParams(methodDecl.Type.Params),
	}

	// 合并方法签名信息
	for k, v := range signatureInfo {
		tmplData[k] = v
	}

	// 生成包装代码文件
	outputFile := filepath.Join(p.outputPath, fmt.Sprintf("%s_cache.go", strings.ToLower(ann.TargetName)))

	// 创建输出目录
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 创建输出文件
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer file.Close()

	// 执行模板
	if err := p.tmpl.Execute(file, tmplData); err != nil {
		return fmt.Errorf("生成缓存包装代码失败: %w", err)
	}

	return nil
}

// processFunctionAnnotation 处理函数上的缓存注解
func (p *CacheProcessor) processFunctionAnnotation(pkgName string, ann types.CommentAnnotation) error {
	// 函数级缓存处理逻辑（类似方法处理，但无接收者）
	return nil
}

// extractMethodSignature 从方法声明中提取方法签名信息
func extractMethodSignature(methodDecl *ast.FuncDecl) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 提取接收者信息（如果有）
	if methodDecl.Recv != nil && len(methodDecl.Recv.List) > 0 {
		recvField := methodDecl.Recv.List[0]
		recvType, err := extractType(recvField.Type)
		if err != nil {
			return nil, fmt.Errorf("提取接收者类型失败: %w", err)
		}

		// 确定接收者名称
		var recvName string
		if len(recvField.Names) > 0 {
			recvName = recvField.Names[0].Name
		} else {
			recvName = "r" // 默认接收者名称
		}

		result["ReceiverType"] = recvType
		result["ReceiverName"] = recvName
		result["IsMethod"] = true
	} else {
		result["IsMethod"] = false
	}

	// 提取参数信息
	params, ctxParamName, err := extractParams(methodDecl.Type.Params)
	if err != nil {
		return nil, fmt.Errorf("提取参数失败: %w", err)
	}
	result["Params"] = params
	result["ContextName"] = ctxParamName

	// 提取返回值信息
	returns, hasReturn, hasError, returnType, err := extractReturns(methodDecl.Type.Results)
	if err != nil {
		return nil, fmt.Errorf("提取返回值失败: %w", err)
	}
	result["Returns"] = returns
	result["HasReturn"] = hasReturn
	result["HasError"] = hasError
	result["ReturnType"] = returnType

	// 生成调用参数列表
	callArgs := generateCallArgs(methodDecl.Type.Params)
	result["CallArgs"] = callArgs

	return result, nil
}

// extractType 提取类型表达式的字符串表示
func extractType(expr ast.Expr) (string, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, nil
	case *ast.StarExpr:
		baseType, err := extractType(t.X)
		if err != nil {
			return "", err
		}
		return "*" + baseType, nil
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name, nil
		}
		return "", fmt.Errorf("不支持的选择器表达式")
	case *ast.ArrayType:
		elemType, err := extractType(t.Elt)
		if err != nil {
			return "", err
		}
		return "[]" + elemType, nil
	case *ast.MapType:
		keyType, err := extractType(t.Key)
		if err != nil {
			return "", err
		}
		valueType, err := extractType(t.Value)
		if err != nil {
			return "", err
		}
		return "map[" + keyType + "]" + valueType, nil
	case *ast.InterfaceType:
		return "interface{}", nil
	case *ast.FuncType:
		return "func()", nil // 简化函数类型表示
	case *ast.ChanType:
		elemType, err := extractType(t.Value)
		if err != nil {
			return "", err
		}
		if t.Dir == ast.SEND {
			return "chan<- " + elemType, nil
		} else if t.Dir == ast.RECV {
			return "<-chan " + elemType, nil
		}
		return "chan " + elemType, nil
	case *ast.StructType:
		return "struct{}", nil // 简化结构体类型表示
	case *ast.Ellipsis:
		elemType, err := extractType(t.Elt)
		if err != nil {
			return "", err
		}
		return "..." + elemType, nil
	default:
		return "", fmt.Errorf("不支持的类型表达式: %T", expr)
	}
}

// extractParams 提取函数参数信息
func extractParams(fields *ast.FieldList) (string, string, error) {
	if fields == nil || len(fields.List) == 0 {
		return "", "", nil
	}

	var params []string
	var ctxParamName string

	for _, field := range fields.List {
		fieldType, err := extractType(field.Type)
		if err != nil {
			return "", "", err
		}

		// 检查是否为上下文参数
		isCtx := false
		if sel, ok := field.Type.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "context" && sel.Sel.Name == "Context" {
				isCtx = true
			}
		}

		if len(field.Names) == 0 {
			// 匿名参数
			params = append(params, fieldType)
			if isCtx {
				ctxParamName = "_ctx" // 为匿名上下文参数指定默认名称
			}
		} else {
			for _, name := range field.Names {
				params = append(params, name.Name+" "+fieldType)
				if isCtx {
					ctxParamName = name.Name
				}
			}
		}
	}

	return strings.Join(params, ", "), ctxParamName, nil
}

// extractReturns 提取函数返回值信息
func extractReturns(fields *ast.FieldList) (string, bool, bool, string, error) {
	if fields == nil || len(fields.List) == 0 {
		return "", false, false, "", nil
	}

	var returns []string
	hasReturn := false
	hasError := false
	var returnType string

	for _, field := range fields.List {
		fieldType, err := extractType(field.Type)
		if err != nil {
			return "", false, false, "", err
		}

		// 检查是否为error类型
		if ident, ok := field.Type.(*ast.Ident); ok && ident.Name == "error" {
			hasError = true
		} else {
			// 非error类型视为返回值
			hasReturn = true
			returnType = fieldType
		}

		if len(field.Names) == 0 {
			// 匿名返回值
			returns = append(returns, fieldType)
		} else {
			for _, name := range field.Names {
				returns = append(returns, name.Name+" "+fieldType)
			}
		}
	}

	return strings.Join(returns, ", "), hasReturn, hasError, returnType, nil
}

// generateCallArgs 生成调用参数列表
func generateCallArgs(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}

	var args []string
	for _, field := range fields.List {
		if len(field.Names) == 0 {
			// 匿名参数，使用类型名作为参数名
			typeName, _ := extractType(field.Type)
			// 将首字母小写作为参数名
			paramName := strings.ToLower(typeName[:1])
			args = append(args, paramName)
		} else {
			for _, name := range field.Names {
				args = append(args, name.Name)
			}
		}
	}

	return strings.Join(args, ", ")
}

// extractParamNames 提取参数名称
func extractParamNames(fields *ast.FieldList) []string {
	if fields == nil || len(fields.List) == 0 {
		return nil
	}

	var names []string
	for _, field := range fields.List {
		if len(field.Names) == 0 {
			// 匿名参数，使用类型名作为参数名
			typeName, _ := extractType(field.Type)
			// 将首字母小写作为参数名
			paramName := strings.ToLower(typeName[:1])
			names = append(names, paramName)
		} else {
			for _, name := range field.Names {
				names = append(names, name.Name)
			}
		}
	}

	return names
}

// extractEvictParams 提取缓存清除方法的参数
func extractEvictParams(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}

	var params []string
	for _, field := range fields.List {
		fieldType, _ := extractType(field.Type)

		// 检查是否为上下文参数
		isCtx := false
		if sel, ok := field.Type.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "context" && sel.Sel.Name == "Context" {
				isCtx = true
			}
		}

		if !isCtx {
			if len(field.Names) == 0 {
				// 匿名参数
				params = append(params, fieldType)
			} else {
				for _, name := range field.Names {
					params = append(params, name.Name+" "+fieldType)
				}
			}
		}
	}

	return strings.Join(params, ", ")
}
