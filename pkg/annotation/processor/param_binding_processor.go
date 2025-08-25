// Package processor 提供注解处理器实现，用于生成代码
package processor

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/isBlue-5/grain/pkg/annotation/registry"
	"github.com/isBlue-5/grain/pkg/annotation/types"
)

// ParamBindingProcessor 参数绑定注解处理器
// 实现自动解析HTTP请求数据到控制器方法参数的功能
type ParamBindingProcessor struct {
	registry   registry.Registry
	parser     AnnotationParser
	outputPath string
	tmpl       *template.Template
	verbose    bool
}

// ParamBindingInfo 参数绑定信息
type ParamBindingInfo struct {
	// 控制器名称
	ControllerName string
	// 方法名称
	MethodName string
	// 方法参数信息
	Params []*MethodParam
	// 返回值信息
	Returns []*ReturnInfo
	// 注解信息
	Annotations []*types.CommentAnnotation
	// 包信息
	PackageName string
	// 导入信息
	Imports map[string]string
}

// MethodParam 方法参数信息
type MethodParam struct {
	// 参数名称
	Name string
	// 参数类型
	Type string
	// 参数类型反射信息
	TypeReflect reflect.Type
	// 参数标签
	Tags string
	// 绑定来源（json, form, query, uri, header, cookie）
	BindingSource string
	// 绑定字段名
	BindingField string
	// 是否必需
	Required bool
	// 默认值
	DefaultValue string
	// 验证规则
	ValidationRules []string
	// 参数位置（从0开始）
	Index int
}

// ReturnInfo 返回值信息
type ReturnInfo struct {
	// 返回值类型
	Type string
	// 返回值名称
	Name string
	// 是否为错误类型
	IsError bool
	// 返回值位置（从0开始）
	Index int
}

// NewParamBindingProcessor 创建参数绑定注解处理器
func NewParamBindingProcessor(reg registry.Registry, parser AnnotationParser, outputPath string) *ParamBindingProcessor {
	tmpl, err := template.New("paramBinding").Parse(paramBindingTemplate)
	if err != nil {
		panic(fmt.Errorf("解析参数绑定模板失败: %w", err))
	}

	return &ParamBindingProcessor{
		registry:   reg,
		parser:     parser,
		outputPath: outputPath,
		tmpl:       tmpl,
		verbose:    false,
	}
}

// WithVerbose 设置是否启用详细日志
func (p *ParamBindingProcessor) WithVerbose(verbose bool) *ParamBindingProcessor {
	p.verbose = verbose
	return p
}

// ProcessParamBinding 处理参数绑定注解
func (p *ParamBindingProcessor) ProcessParamBinding(paths []string) error {
	// 解析所有路径下的注解
	var annotations []types.Annotation
	for _, path := range paths {
		anns, err := p.parser.ParseDir(path)
		if err != nil {
			return fmt.Errorf("解析目录 %s 失败: %w", path, err)
		}

		for _, ann := range anns {
			// 查找控制器和路由注解
			if ann.GetType() == "controller" || ann.GetType() == "route" {
				annotations = append(annotations, ann)
			}
		}
	}

	if p.verbose {
		fmt.Printf("找到 %d 个注解\n", len(annotations))
	}

	// 按包和控制器组织注解
	packageControllers := make(map[string]map[string]*ParamBindingInfo)

	// 处理每个注解
	for _, ann := range annotations {
		if commentAnn, ok := ann.(*types.CommentAnnotation); ok {
			if err := p.processParamBindingAnnotation(commentAnn, packageControllers); err != nil {
				return err
			}
		}
	}

	// 为每个包和控制器生成参数绑定代码
	for pkgPath, controllers := range packageControllers {
		for controllerName, bindingInfo := range controllers {
			if err := p.generateParamBindingCode(pkgPath, controllerName, bindingInfo); err != nil {
				return fmt.Errorf("生成参数绑定代码失败 for %s.%s: %w", pkgPath, controllerName, err)
			}
		}
	}

	return nil
}

// processParamBindingAnnotation 处理单个参数绑定注解
func (p *ParamBindingProcessor) processParamBindingAnnotation(ann *types.CommentAnnotation, packageControllers map[string]map[string]*ParamBindingInfo) error {
	// 获取包路径 - 使用相对路径
	absPath := filepath.Dir(ann.Position.Filename)
	pkgPath, err := filepath.Rel(filepath.Dir(p.outputPath), absPath)
	if err != nil {
		// 如果获取相对路径失败，使用目录名
		pkgPath = filepath.Base(absPath)
	}
	pkgName := p.getPackageName(absPath)

	// 初始化包映射
	if _, ok := packageControllers[pkgPath]; !ok {
		packageControllers[pkgPath] = make(map[string]*ParamBindingInfo)
	}

	// 处理控制器注解
	if ann.GetType() == "controller" {
		controllerName := ann.TargetName
		// 移除"Controller"后缀，保持与路由注解处理一致
		if strings.HasSuffix(controllerName, "Controller") {
			controllerName = strings.TrimSuffix(controllerName, "Controller")
		}
		if _, ok := packageControllers[pkgPath][controllerName]; !ok {
			packageControllers[pkgPath][controllerName] = &ParamBindingInfo{
				ControllerName: controllerName,
				PackageName:    pkgName,
				Imports:        make(map[string]string),
				Annotations:    make([]*types.CommentAnnotation, 0),
			}
		}
		packageControllers[pkgPath][controllerName].Annotations = append(
			packageControllers[pkgPath][controllerName].Annotations, ann)

		// 为控制器注解也添加必要的导入
		p.addRequiredImports(packageControllers[pkgPath][controllerName], nil)
	}

	// 处理路由注解
	if ann.GetType() == "route" {
		// 查找对应的控制器
		controllerName := p.findControllerName(ann)
		if controllerName == "" {
			return fmt.Errorf("无法找到路由 %s 对应的控制器", ann.TargetName)
		}

		if _, ok := packageControllers[pkgPath][controllerName]; !ok {
			packageControllers[pkgPath][controllerName] = &ParamBindingInfo{
				ControllerName: controllerName,
				PackageName:    pkgName,
				Imports:        make(map[string]string),
				Annotations:    make([]*types.CommentAnnotation, 0),
			}
		}

		// 解析方法参数
		methodParams, err := p.parseMethodParams(ann)
		if err != nil {
			return fmt.Errorf("解析方法参数失败: %w", err)
		}

		// 解析方法返回值
		returns, err := p.parseMethodReturns(ann)
		if err != nil {
			return fmt.Errorf("解析方法返回值失败: %w", err)
		}

		// 添加到控制器信息中
		controllerInfo := packageControllers[pkgPath][controllerName]
		controllerInfo.MethodName = ann.TargetName
		controllerInfo.Params = methodParams
		controllerInfo.Returns = returns
		controllerInfo.Annotations = append(controllerInfo.Annotations, ann)

		// 添加必要的导入
		p.addRequiredImports(controllerInfo, methodParams)
	}

	return nil
}

// findControllerName 查找路由对应的控制器名称
func (p *ParamBindingProcessor) findControllerName(routeAnn *types.CommentAnnotation) string {
	// 通过AST节点查找控制器名称
	if routeAnn.Node == nil {
		if p.verbose {
			fmt.Printf("警告: 路由注解 %s 没有关联的AST节点\n", routeAnn.TargetName)
		}
		return ""
	}

	// 查找方法声明节点
	var funcDecl *ast.FuncDecl
	switch node := routeAnn.Node.(type) {
	case *ast.FuncDecl:
		funcDecl = node
	case *ast.CommentGroup:
		// 如果节点是注释组，查找其父节点
		if parent := p.findParentFuncDecl(node); parent != nil {
			funcDecl = parent
		}
	default:
		// 尝试查找父级函数声明
		if parent := p.findParentFuncDecl(node); parent != nil {
			funcDecl = parent
		}
	}

	if funcDecl == nil {
		if p.verbose {
			fmt.Printf("警告: 无法找到路由 %s 对应的函数声明\n", routeAnn.TargetName)
		}
		return ""
	}

	// 检查是否有接收者（方法）
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		if p.verbose {
			fmt.Printf("警告: 路由 %s 不是控制器方法\n", routeAnn.TargetName)
		}
		return ""
	}

	// 获取接收者类型
	recvType := funcDecl.Recv.List[0].Type
	var controllerName string

	// 处理指针接收者
	if starExpr, ok := recvType.(*ast.StarExpr); ok {
		if ident, ok := starExpr.X.(*ast.Ident); ok {
			controllerName = ident.Name
		}
	} else if ident, ok := recvType.(*ast.Ident); ok {
		controllerName = ident.Name
	}

	if controllerName == "" {
		if p.verbose {
			fmt.Printf("警告: 无法解析控制器名称 for 路由 %s\n", routeAnn.TargetName)
		}
		return ""
	}

	// 移除"Controller"后缀（如果存在）
	if strings.HasSuffix(controllerName, "Controller") {
		controllerName = strings.TrimSuffix(controllerName, "Controller")
	}

	if p.verbose {
		fmt.Printf("找到路由 %s 对应的控制器: %s\n", routeAnn.TargetName, controllerName)
	}

	return controllerName
}

// findParentFuncDecl 查找父级函数声明
func (p *ParamBindingProcessor) findParentFuncDecl(node ast.Node) *ast.FuncDecl {
	// 遍历AST树查找父级函数声明
	var funcDecl *ast.FuncDecl
	ast.Inspect(node, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok {
			funcDecl = fd
			return false // 停止遍历
		}
		return true
	})
	return funcDecl
}

// parseMethodParams 解析方法参数
func (p *ParamBindingProcessor) parseMethodParams(ann *types.CommentAnnotation) ([]*MethodParam, error) {
	var params []*MethodParam

	// 查找方法声明
	funcDecl := p.findMethodDeclaration(ann)
	if funcDecl == nil {
		return nil, fmt.Errorf("无法找到方法声明")
	}

	// 解析参数列表
	for i, param := range funcDecl.Type.Params.List {
		paramType := p.getTypeString(param.Type)

		// 处理多个参数名共享同一类型的情况
		for _, name := range param.Names {
			methodParam := &MethodParam{
				Name:        name.Name,
				Type:        paramType,
				TypeReflect: p.getReflectType(param.Type),
				Index:       i,
				Required:    true, // 默认必需
			}

			// 解析参数标签和绑定信息
			p.parseParamTags(methodParam, param)

			// 解析验证规则
			p.parseValidationRules(methodParam, ann)

			// 设置默认值
			p.parseDefaultValue(methodParam, ann)

			params = append(params, methodParam)
		}
	}

	if p.verbose {
		fmt.Printf("解析方法 %s 参数: %d 个\n", ann.TargetName, len(params))
	}

	return params, nil
}

// findMethodDeclaration 查找方法声明
func (p *ParamBindingProcessor) findMethodDeclaration(ann *types.CommentAnnotation) *ast.FuncDecl {
	if ann.Node == nil {
		return nil
	}

	// 直接查找函数声明
	var funcDecl *ast.FuncDecl
	ast.Inspect(ann.Node, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok {
			funcDecl = fd
			return false
		}
		return true
	})

	return funcDecl
}

// getTypeString 获取类型的字符串表示
func (p *ParamBindingProcessor) getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + p.getTypeString(t.X)
	case *ast.ArrayType:
		return "[]" + p.getTypeString(t.Elt)
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
		return p.getTypeString(t.X) + "." + t.Sel.Name
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.MapType:
		keyType := p.getTypeString(t.Key)
		valueType := p.getTypeString(t.Value)
		return "map[" + keyType + "]" + valueType
	case *ast.ChanType:
		chanType := p.getTypeString(t.Value)
		if t.Dir == ast.SEND {
			return "chan<- " + chanType
		} else if t.Dir == ast.RECV {
			return "<-chan " + chanType
		}
		return "chan " + chanType
	default:
		return "unknown"
	}
}

// getReflectType 获取反射类型信息
func (p *ParamBindingProcessor) getReflectType(expr ast.Expr) reflect.Type {
	// 这里简化处理，实际应该通过类型解析器获取
	typeStr := p.getTypeString(expr)

	// 基本类型映射
	basicTypes := map[string]reflect.Type{
		"string":  reflect.TypeOf(""),
		"int":     reflect.TypeOf(0),
		"int8":    reflect.TypeOf(int8(0)),
		"int16":   reflect.TypeOf(int16(0)),
		"int32":   reflect.TypeOf(int32(0)),
		"int64":   reflect.TypeOf(int64(0)),
		"uint":    reflect.TypeOf(uint(0)),
		"uint8":   reflect.TypeOf(uint8(0)),
		"uint16":  reflect.TypeOf(uint16(0)),
		"uint32":  reflect.TypeOf(uint32(0)),
		"uint64":  reflect.TypeOf(uint64(0)),
		"float32": reflect.TypeOf(float32(0)),
		"float64": reflect.TypeOf(float64(0)),
		"bool":    reflect.TypeOf(false),
		"error":   reflect.TypeOf((*error)(nil)).Elem(),
	}

	if t, ok := basicTypes[typeStr]; ok {
		return t
	}

	// 对于复杂类型，返回interface{}类型
	return reflect.TypeOf((*interface{})(nil)).Elem()
}

// parseParamTags 解析参数标签
func (p *ParamBindingProcessor) parseParamTags(param *MethodParam, field *ast.Field) {
	// 解析结构体标签
	if field.Tag != nil {
		tagValue := strings.Trim(field.Tag.Value, "`")
		tags := reflect.StructTag(tagValue)

		// 解析binding标签
		if binding := tags.Get("binding"); binding != "" {
			param.BindingSource = "binding"
			param.Tags = binding
			if strings.Contains(binding, "required") {
				param.Required = true
			}
		}

		// 解析json标签
		if json := tags.Get("json"); json != "" {
			if param.BindingSource == "" {
				param.BindingSource = "json"
			}
			param.BindingField = json
		}

		// 解析form标签
		if form := tags.Get("form"); form != "" {
			if param.BindingSource == "" {
				param.BindingSource = "form"
			}
			param.BindingField = form
		}

		// 解析query标签
		if query := tags.Get("query"); query != "" {
			if param.BindingSource == "" {
				param.BindingSource = "query"
			}
			param.BindingField = query
		}

		// 解析uri标签
		if uri := tags.Get("uri"); uri != "" {
			if param.BindingSource == "" {
				param.BindingSource = "uri"
			}
			param.BindingField = uri
		}

		// 解析header标签
		if header := tags.Get("header"); header != "" {
			if param.BindingSource == "" {
				param.BindingSource = "header"
			}
			param.BindingField = header
		}

		// 解析cookie标签
		if cookie := tags.Get("cookie"); cookie != "" {
			if param.BindingSource == "" {
				param.BindingSource = "cookie"
			}
			param.BindingField = cookie
		}

		// 解析validate标签
		if validate := tags.Get("validate"); validate != "" {
			rules := strings.Split(validate, ",")
			param.ValidationRules = append(param.ValidationRules, rules...)
		}
	}

	// 如果没有标签，根据参数位置和类型推断绑定来源
	if param.BindingSource == "" {
		p.inferBindingSource(param)
	}
}

// inferBindingSource 推断绑定来源
func (p *ParamBindingProcessor) inferBindingSource(param *MethodParam) {
	// 根据参数位置和类型推断绑定来源
	switch param.Index {
	case 0:
		// 第一个参数通常是gin.Context
		if param.Type == "*gin.Context" || param.Type == "gin.Context" {
			param.BindingSource = "context"
			param.Required = false // context参数不需要验证
		}
	case 1:
		// 第二个参数通常是请求体或URI参数
		if strings.Contains(strings.ToLower(param.Type), "request") {
			param.BindingSource = "json"
		} else if param.Type == "uint" || param.Type == "int" || param.Type == "string" {
			param.BindingSource = "uri"
		}
	default:
		// 其他参数根据类型推断
		if param.Type == "uint" || param.Type == "int" {
			param.BindingSource = "uri"
		} else if param.Type == "string" {
			param.BindingSource = "query"
		}
	}
}

// parseValidationRules 解析验证规则
func (p *ParamBindingProcessor) parseValidationRules(param *MethodParam, ann *types.CommentAnnotation) {
	// 从注解中查找验证规则
	for _, attr := range ann.Attributes {
		if attr.Name == "validate" || attr.Name == "rules" {
			if rules, ok := attr.Value.(string); ok {
				// 解析验证规则字符串
				ruleList := strings.Split(rules, "|")
				for _, rule := range ruleList {
					rule = strings.TrimSpace(rule)
					if rule != "" {
						param.ValidationRules = append(param.ValidationRules, rule)
					}
				}
			} else if ruleSlice, ok := attr.Value.([]string); ok {
				param.ValidationRules = append(param.ValidationRules, ruleSlice...)
			}
		}
	}

	// 如果没有注解规则，使用结构体标签中的规则
	if len(param.ValidationRules) == 0 && param.Tags != "" {
		// 从binding标签中提取验证规则
		if strings.Contains(param.Tags, "required") {
			param.ValidationRules = append(param.ValidationRules, "required")
		}
		if strings.Contains(param.Tags, "email") {
			param.ValidationRules = append(param.ValidationRules, "email")
		}
		if strings.Contains(param.Tags, "min:") {
			// 提取min规则
			if idx := strings.Index(param.Tags, "min:"); idx != -1 {
				endIdx := strings.IndexAny(param.Tags[idx:], " ,")
				if endIdx == -1 {
					endIdx = len(param.Tags[idx:])
				}
				minRule := param.Tags[idx : idx+endIdx]
				param.ValidationRules = append(param.ValidationRules, minRule)
			}
		}
		if strings.Contains(param.Tags, "max:") {
			// 提取max规则
			if idx := strings.Index(param.Tags, "max:"); idx != -1 {
				endIdx := strings.IndexAny(param.Tags[idx:], " ,")
				if endIdx == -1 {
					endIdx = len(param.Tags[idx:])
				}
				maxRule := param.Tags[idx : idx+endIdx]
				param.ValidationRules = append(param.ValidationRules, maxRule)
			}
		}
	}
}

// parseDefaultValue 解析默认值
func (p *ParamBindingProcessor) parseDefaultValue(param *MethodParam, ann *types.CommentAnnotation) {
	// 从注解中查找默认值
	for _, attr := range ann.Attributes {
		if attr.Name == "default" {
			if defaultValue, ok := attr.Value.(string); ok {
				param.DefaultValue = defaultValue
			}
		}
	}

	// 根据类型设置合理的默认值
	if param.DefaultValue == "" {
		switch param.Type {
		case "string":
			param.DefaultValue = `""`
		case "int", "int8", "int16", "int32", "int64":
			param.DefaultValue = "0"
		case "uint", "uint8", "uint16", "uint32", "uint64":
			param.DefaultValue = "0"
		case "float32", "float64":
			param.DefaultValue = "0.0"
		case "bool":
			param.DefaultValue = "false"
		case "[]string":
			param.DefaultValue = "nil"
		case "[]int":
			param.DefaultValue = "nil"
		case "map[string]interface{}":
			param.DefaultValue = "nil"
		}
	}
}

// parseMethodReturns 解析方法返回值
func (p *ParamBindingProcessor) parseMethodReturns(ann *types.CommentAnnotation) ([]*ReturnInfo, error) {
	var returns []*ReturnInfo

	// 查找方法声明
	funcDecl := p.findMethodDeclaration(ann)
	if funcDecl == nil {
		return nil, fmt.Errorf("无法找到方法声明")
	}

	// 解析返回值列表
	if funcDecl.Type.Results != nil {
		for i, result := range funcDecl.Type.Results.List {
			returnType := p.getTypeString(result.Type)

			// 处理多个返回值名共享同一类型的情况
			if len(result.Names) > 0 {
				for _, name := range result.Names {
					returnInfo := &ReturnInfo{
						Type:    returnType,
						Name:    name.Name,
						Index:   i,
						IsError: returnType == "error",
					}
					returns = append(returns, returnInfo)
				}
			} else {
				// 匿名返回值
				returnInfo := &ReturnInfo{
					Type:    returnType,
					Name:    fmt.Sprintf("ret%d", i),
					Index:   i,
					IsError: returnType == "error",
				}
				returns = append(returns, returnInfo)
			}
		}
	}

	if p.verbose {
		fmt.Printf("解析方法 %s 返回值: %d 个\n", ann.TargetName, len(returns))
	}

	return returns, nil
}

// addRequiredImports 添加必要的导入
func (p *ParamBindingProcessor) addRequiredImports(controllerInfo *ParamBindingInfo, params []*MethodParam) {
	// 添加基础导入
	controllerInfo.Imports["github.com/gin-gonic/gin"] = ""
	controllerInfo.Imports["net/http"] = ""

	// 根据参数类型添加必要的导入
	for _, param := range params {
		if strings.Contains(param.Type, "validator") {
			controllerInfo.Imports["github.com/go-playground/validator/v10"] = "validator"
		}
		if strings.Contains(param.Type, "binding") {
			controllerInfo.Imports["github.com/gin-gonic/gin/binding"] = ""
		}
	}

	// 调试信息
	if p.verbose {
		fmt.Printf("DEBUG: 为控制器 %s 添加导入: %v\n", controllerInfo.ControllerName, controllerInfo.Imports)
	}
}

// getPackageName 获取包名
func (p *ParamBindingProcessor) getPackageName(pkgPath string) string {
	// 解析go.mod文件获取包名
	goModPath := filepath.Join(pkgPath, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		// 读取go.mod文件获取模块名
		content, err := os.ReadFile(goModPath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "module ") {
					moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module "))
					// 提取包名
					parts := strings.Split(moduleName, "/")
					return parts[len(parts)-1]
				}
			}
		}
	}

	// 如果无法获取，使用目录名
	return filepath.Base(pkgPath)
}

// generateParamBindingCode 生成参数绑定代码
func (p *ParamBindingProcessor) generateParamBindingCode(pkgPath, controllerName string, bindingInfo *ParamBindingInfo) error {
	// 创建输出目录 - 使用统一的路径解析
	outputDir := p.resolveOutputPath(pkgPath, "binding")

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 创建输出文件
	outputFile := filepath.Join(outputDir, strings.ToLower(controllerName)+"_param_binding_gen.go")

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer file.Close()

	// 执行模板
	if err := p.tmpl.Execute(file, bindingInfo); err != nil {
		return fmt.Errorf("生成参数绑定代码失败: %w", err)
	}

	if p.verbose {
		fmt.Printf("参数绑定代码生成成功: %s\n", outputFile)
		fmt.Printf("DEBUG: 模板数据: PackageName=%s, ControllerName=%s, Imports=%v\n",
			bindingInfo.PackageName, bindingInfo.ControllerName, bindingInfo.Imports)
	}

	return nil
}

// 参数绑定代码模板
const paramBindingTemplate = `// Code generated by gRain. DO NOT EDIT.
// 自动参数绑定代码

package {{.PackageName}}

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"github.com/gin-gonic/gin"
	{{range $alias, $path := .Imports}}
	{{if and $path (ne $path "")}}
	{{if $alias}}{{$alias}} "{{$path}}"{{else}}"{{$path}}"{{end}}
	{{end}}
	{{end}}
)

// {{.ControllerName}}ParamBinder {{.ControllerName}}参数绑定器
type {{.ControllerName}}ParamBinder struct {
	controller interface{}
}

// New{{.ControllerName}}ParamBinder 创建参数绑定器
func New{{.ControllerName}}ParamBinder(controller interface{}) *{{.ControllerName}}ParamBinder {
	return &{{.ControllerName}}ParamBinder{
		controller: controller,
	}
}

{{if .MethodName}}
// {{.MethodName}}ParamBinder {{.MethodName}}方法参数绑定器
func (b *{{$.ControllerName}}ParamBinder) {{.MethodName}}(c *gin.Context) {
	{{if .Params}}
	// 自动绑定参数
	{{range $index, $param := .Params}}
	var {{$param.Name}} {{$param.Type}}
	{{if eq $param.BindingSource "uri"}}
	// 绑定URI参数
	if value := c.Param("{{$param.BindingField}}"); value != "" {
		{{if eq $param.Type "string"}}
		{{$param.Name}} = value
		{{else if eq $param.Type "int"}}
		if intVal, err := strconv.Atoi(value); err == nil {
			{{$param.Name}} = intVal
		}
		{{else if eq $param.Type "uint"}}
		if uintVal, err := strconv.ParseUint(value, 10, 32); err == nil {
			{{$param.Name}} = uint(uintVal)
		}
		{{else if eq $param.Type "int64"}}
		if int64Val, err := strconv.ParseInt(value, 10, 64); err == nil {
			{{$param.Name}} = int64Val
		}
		{{else if eq $param.Type "uint64"}}
		if uint64Val, err := strconv.ParseUint(value, 10, 64); err == nil {
			{{$param.Name}} = uint64Val
		}
		{{else if eq $param.Type "float64"}}
		if float64Val, err := strconv.ParseFloat(value, 64); err == nil {
			{{$param.Name}} = float64Val
		}
		{{else if eq $param.Type "bool"}}
		if boolVal, err := strconv.ParseBool(value); err == nil {
			{{$param.Name}} = boolVal
		}
		{{end}}
	}
	{{else if eq $param.BindingSource "query"}}
	// 绑定查询参数
	if value := c.Query("{{$param.BindingField}}"); value != "" {
		{{if eq $param.Type "string"}}
		{{$param.Name}} = value
		{{else if eq $param.Type "int"}}
		if intVal, err := strconv.Atoi(value); err == nil {
			{{$param.Name}} = intVal
		}
		{{else if eq $param.Type "uint"}}
		if uintVal, err := strconv.ParseUint(value, 10, 32); err == nil {
			{{$param.Name}} = uint(uintVal)
		}
		{{else if eq $param.Type "int64"}}
		if int64Val, err := strconv.ParseInt(value, 10, 64); err == nil {
			{{$param.Name}} = int64Val
		}
		{{else if eq $param.Type "uint64"}}
		if uint64Val, err := strconv.ParseUint(value, 10, 64); err == nil {
			{{$param.Name}} = uint64Val
		}
		{{else if eq $param.Type "float64"}}
		if float64Val, err := strconv.ParseFloat(value, 64); err == nil {
			{{$param.Name}} = float64Val
		}
		{{else if eq $param.Type "bool"}}
		if boolVal, err := strconv.ParseBool(value); err == nil {
			{{$param.Name}} = boolVal
		}
		{{end}}
	}
	{{else if eq $param.BindingSource "form"}}
	// 绑定表单参数
	if value := c.PostForm("{{$param.BindingField}}"); value != "" {
		{{if eq $param.Type "string"}}
		{{$param.Name}} = value
		{{else if eq $param.Type "int"}}
		if intVal, err := strconv.Atoi(value); err == nil {
			{{$param.Name}} = intVal
		}
		{{else if eq $param.Type "uint"}}
		if uintVal, err := strconv.ParseUint(value, 10, 32); err == nil {
			{{$param.Name}} = uint(uintVal)
		}
		{{else if eq $param.Type "int64"}}
		if int64Val, err := strconv.ParseInt(value, 10, 64); err == nil {
			{{$param.Name}} = int64Val
		}
		{{else if eq $param.Type "uint64"}}
		if uint64Val, err := strconv.ParseUint(value, 10, 64); err == nil {
			{{$param.Name}} = uint64Val
		}
		{{else if eq $param.Type "float64"}}
		if float64Val, err := strconv.ParseFloat(value, 64); err == nil {
			{{$param.Name}} = float64Val
		}
		{{else if eq $param.Type "bool"}}
		if boolVal, err := strconv.ParseBool(value); err == nil {
			{{$param.Name}} = boolVal
		}
		{{end}}
	}
	{{else if eq $param.BindingSource "header"}}
	// 绑定请求头参数
	if value := c.GetHeader("{{$param.BindingField}}"); value != "" {
		{{if eq $param.Type "string"}}
		{{$param.Name}} = value
		{{else if eq $param.Type "int"}}
		if intVal, err := strconv.Atoi(value); err == nil {
			{{$param.Name}} = intVal
		}
		{{else if eq $param.Type "uint"}}
		if uintVal, err := strconv.ParseUint(value, 10, 32); err == nil {
			{{$param.Name}} = uint(uintVal)
		}
		{{else if eq $param.Type "int64"}}
		if int64Val, err := strconv.ParseInt(value, 10, 64); err == nil {
			{{$param.Name}} = int64Val
		}
		{{else if eq $param.Type "uint64"}}
		if uint64Val, err := strconv.ParseUint(value, 10, 64); err == nil {
			{{$param.Name}} = uint64Val
		}
		{{else if eq $param.Type "float64"}}
		if float64Val, err := strconv.ParseFloat(value, 64); err == nil {
			{{$param.Name}} = float64Val
		}
		{{else if eq $param.Type "bool"}}
		if boolVal, err := strconv.ParseBool(value); err == nil {
			{{$param.Name}} = boolVal
		}
		{{end}}
	}
	{{else if eq $param.BindingSource "cookie"}}
	// 绑定Cookie参数
	if cookie, err := c.Cookie("{{$param.BindingField}}"); err == nil && cookie != "" {
		{{if eq $param.Type "string"}}
		{{$param.Name}} = cookie
		{{else if eq $param.Type "int"}}
		if intVal, err := strconv.Atoi(cookie); err == nil {
			{{$param.Name}} = intVal
		}
		{{else if eq $param.Type "uint"}}
		if uintVal, err := strconv.ParseUint(cookie, 10, 32); err == nil {
			{{$param.Name}} = uint(uintVal)
		}
		{{else if eq $param.Type "int64"}}
		if int64Val, err := strconv.ParseInt(cookie, 10, 64); err == nil {
			{{$param.Name}} = int64Val
		}
		{{else if eq $param.Type "uint64"}}
		if uint64Val, err := strconv.ParseUint(cookie, 10, 64); err == nil {
			{{$param.Name}} = uint64Val
		}
		{{else if eq $param.Type "float64"}}
		if float64Val, err := strconv.ParseFloat(cookie, 64); err == nil {
			{{$param.Name}} = float64Val
		}
		{{else if eq $param.Type "bool"}}
		if boolVal, err := strconv.ParseBool(cookie); err == nil {
			{{$param.Name}} = boolVal
		}
		{{end}}
	}
	{{else}}
	// 绑定JSON请求体
	if err := c.ShouldBindJSON(&{{$param.Name}}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数绑定失败",
			"error":   err.Error(),
		})
		return
	}
	{{end}}
	{{end}}

	// 调用原始控制器方法
	{{if .Params}}
	// 使用反射调用方法
	method := reflect.ValueOf(b.controller).MethodByName("{{.MethodName}}")
	if !method.IsValid() {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "方法不存在",
		})
		return
	}

	// 准备参数
	args := []reflect.Value{
		reflect.ValueOf(c),
		{{range $param := .Params}}
		reflect.ValueOf({{$param.Name}}),
		{{end}}
	}

	// 调用方法
	results := method.Call(args)

	// 处理返回值
	{{if .Returns}}
	{{range $index, $return := .Returns}}
	{{if eq $return.IsError true}}
	if results[{{$index}}].Interface() != nil {
		if err, ok := results[{{$index}}].Interface().(error); ok && err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "操作失败",
				"error":   err.Error(),
			})
			return
		}
	}
	{{else}}
	// 处理非错误返回值
	result{{$index}} := results[{{$index}}].Interface()
	{{end}}
	{{end}}
	{{end}}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "操作成功",
		{{if .Returns}}
		{{range $index, $return := .Returns}}
		{{if not $return.IsError}}
		"data": result{{$index}},
		{{end}}
		{{end}}
		{{end}}
	})
	{{else}}
	// 没有参数的方法
	method := reflect.ValueOf(b.controller).MethodByName("{{.MethodName}}")
	if !method.IsValid() {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "方法不存在",
		})
		return
	}

	// 调用方法
	results := method.Call([]reflect.Value{reflect.ValueOf(c)})

	// 处理返回值
	{{if .Returns}}
	{{range $index, $return := .Returns}}
	{{if eq $return.IsError true}}
	if results[{{$index}}].Interface() != nil {
		if err, ok := results[{{$index}}].Interface().(error); ok && err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "操作失败",
				"error":   err.Error(),
			})
			return
		}
	}
	{{else}}
	// 处理非错误返回值
	result{{$index}} := results[{{$index}}].Interface()
	{{end}}
	{{end}}
	{{end}}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "操作成功",
		{{if .Returns}}
		{{range $index, $return := .Returns}}
		{{if not $return.IsError}}
		"data": result{{$index}},
		{{end}}
		{{end}}
		{{end}}
	})
	{{end}}
	{{end}}
}
{{end}}

// 自动参数绑定中间件
func AutoParamBinding(controller interface{}) gin.HandlerFunc {
	binder := New{{.ControllerName}}ParamBinder(controller)
	
	return func(c *gin.Context) {
		// 根据请求路径和方法自动调用对应的绑定器
		path := c.Request.URL.Path
		method := c.Request.Method
		
		// 智能路由到对应的绑定器方法
		methodName := p.inferMethodName(path, method)
		if methodName != "" {
			if method := reflect.ValueOf(binder).MethodByName(methodName); method.IsValid() {
				method.Call([]reflect.Value{reflect.ValueOf(c)})
				c.Abort() // 阻止继续执行
				return
			}
		}
		
		// 如果无法推断方法名，尝试通用的路径匹配
		for _, route := range p.getAvailableRoutes(binder) {
			if strings.Contains(path, route.Path) && method == route.Method {
				if method := reflect.ValueOf(binder).MethodByName(route.MethodName); method.IsValid() {
					method.Call([]reflect.Value{reflect.ValueOf(c)})
					c.Abort() // 阻止继续执行
					return
				}
			}
		}
		
		c.Next()
	}
}
`

// resolveOutputPath 解析输出路径，避免深层嵌套和绝对路径问题
func (p *ParamBindingProcessor) resolveOutputPath(pkgPath, subDir string) string {
	// 如果是测试环境或包含绝对路径，使用简化的输出目录
	if strings.Contains(pkgPath, os.TempDir()) || filepath.IsAbs(pkgPath) {
		return filepath.Join(p.outputPath, subDir)
	}

	// 生产环境：使用相对路径
	cleanPkgPath := strings.TrimPrefix(pkgPath, "./")
	cleanPkgPath = strings.TrimPrefix(cleanPkgPath, "../")

	// 避免太深的嵌套，只使用最后一级目录
	if strings.Contains(cleanPkgPath, "/") {
		cleanPkgPath = filepath.Base(cleanPkgPath)
	}

	return filepath.Join(p.outputPath, cleanPkgPath, subDir)
}

// inferMethodName 从路径和方法推断方法名
func (p *ParamBindingProcessor) inferMethodName(path, method string) string {
	// 从路径中提取关键信息
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(pathParts) == 0 {
		return ""
	}

	// 根据HTTP方法和路径推断方法名
	switch method {
	case "GET":
		if len(pathParts) == 1 {
			return "List" + strings.Title(pathParts[0])
		} else if len(pathParts) == 2 {
			return "Get" + strings.Title(pathParts[0]) + "ByID"
		}
	case "POST":
		if len(pathParts) == 1 {
			return "Create" + strings.Title(pathParts[0])
		}
	case "PUT":
		if len(pathParts) == 2 {
			return "Update" + strings.Title(pathParts[0])
		}
	case "DELETE":
		if len(pathParts) == 2 {
			return "Delete" + strings.Title(pathParts[0])
		}
	case "PATCH":
		if len(pathParts) == 2 {
			return "Patch" + strings.Title(pathParts[0])
		}
	}

	return ""
}

// BindingRouteInfo 绑定路由信息
type BindingRouteInfo struct {
	Path       string
	Method     string
	MethodName string
}

// getAvailableRoutes 获取可用的路由信息
func (p *ParamBindingProcessor) getAvailableRoutes(binder interface{}) []BindingRouteInfo {
	var routes []BindingRouteInfo

	// 通过反射获取绑定器的所有方法
	binderType := reflect.TypeOf(binder)
	for i := 0; i < binderType.NumMethod(); i++ {
		method := binderType.Method(i)
		methodName := method.Name

		// 根据方法名推断路由信息
		if strings.HasPrefix(methodName, "List") {
			entityName := strings.TrimPrefix(methodName, "List")
			routes = append(routes, BindingRouteInfo{
				Path:       "/" + strings.ToLower(entityName),
				Method:     "GET",
				MethodName: methodName,
			})
		} else if strings.HasPrefix(methodName, "Get") && strings.HasSuffix(methodName, "ByID") {
			entityName := strings.TrimPrefix(methodName, "Get")
			entityName = strings.TrimSuffix(entityName, "ByID")
			routes = append(routes, BindingRouteInfo{
				Path:       "/" + strings.ToLower(entityName) + "/:id",
				Method:     "GET",
				MethodName: methodName,
			})
		} else if strings.HasPrefix(methodName, "Create") {
			entityName := strings.TrimPrefix(methodName, "Create")
			routes = append(routes, BindingRouteInfo{
				Path:       "/" + strings.ToLower(entityName),
				Method:     "POST",
				MethodName: methodName,
			})
		} else if strings.HasPrefix(methodName, "Update") {
			entityName := strings.TrimPrefix(methodName, "Update")
			routes = append(routes, BindingRouteInfo{
				Path:       "/" + strings.ToLower(entityName) + "/:id",
				Method:     "PUT",
				MethodName: methodName,
			})
		} else if strings.HasPrefix(methodName, "Delete") {
			entityName := strings.TrimPrefix(methodName, "Delete")
			routes = append(routes, BindingRouteInfo{
				Path:       "/" + strings.ToLower(entityName) + "/:id",
				Method:     "DELETE",
				MethodName: methodName,
			})
		} else if strings.HasPrefix(methodName, "Patch") {
			entityName := strings.TrimPrefix(methodName, "Patch")
			routes = append(routes, BindingRouteInfo{
				Path:       "/" + strings.ToLower(entityName) + "/:id",
				Method:     "PATCH",
				MethodName: methodName,
			})
		}
	}

	return routes
}
