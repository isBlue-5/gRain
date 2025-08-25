// Package processor 提供增强的路由处理器实现
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

// EnhancedRouteProcessor 增强的路由处理器
// 支持自动参数绑定和自动解析请求数据
type EnhancedRouteProcessor struct {
	registry   registry.Registry
	parser     AnnotationParser
	outputPath string
	tmpl       *template.Template
	verbose    bool
}

// EnhancedRouteInfo 增强的路由信息
type EnhancedRouteInfo struct {
	// 控制器名称
	ControllerName string
	// 控制器路径前缀
	PathPrefix string
	// 路由列表
	Routes []*EnhancedRoute
	// 包信息
	PackageName string
	// 导入信息
	Imports map[string]string
}

// EnhancedRoute 增强的路由信息
type EnhancedRoute struct {
	// HTTP方法
	Method string
	// 路径
	Path string
	// 方法名称
	MethodName string
	// 方法参数
	Params []*RouteParam
	// 返回值
	Returns []*RouteReturn
	// 认证要求
	Auth *RouteAuth
	// 验证规则
	Validation *RouteValidation
	// 绑定信息
	Binding *RouteBinding
}

// RouteParam 路由参数信息
type RouteParam struct {
	// 参数名称
	Name string
	// 参数类型
	Type string
	// 绑定来源
	BindingSource string
	// 绑定字段名
	BindingField string
	// 是否必需
	Required bool
	// 默认值
	DefaultValue string
	// 验证规则
	ValidationRules []string
}

// RouteReturn 路由返回值信息
type RouteReturn struct {
	// 返回类型
	Type string
	// 是否为错误
	IsError bool
}

// RouteAuth 路由认证信息
type RouteAuth struct {
	// 必需角色
	RequiredRoles []string
	// 必需权限
	RequiredPermissions []string
}

// RouteValidation 路由验证信息
type RouteValidation struct {
	// 验证规则
	Rules string
	// 验证组
	Groups []string
}

// RouteBinding 路由绑定信息
type RouteBinding struct {
	// 绑定来源
	Source string
	// 绑定模型
	Model string
}

// NewEnhancedRouteProcessor 创建增强的路由处理器
func NewEnhancedRouteProcessor(reg registry.Registry, parser AnnotationParser, outputPath string) *EnhancedRouteProcessor {
	tmpl, err := template.New("enhancedRoute").Parse(enhancedRouteTemplate)
	if err != nil {
		panic(fmt.Errorf("解析增强路由模板失败: %w", err))
	}

	return &EnhancedRouteProcessor{
		registry:   reg,
		parser:     parser,
		outputPath: outputPath,
		tmpl:       tmpl,
		verbose:    false,
	}
}

// WithVerbose 设置是否启用详细日志
func (p *EnhancedRouteProcessor) WithVerbose(verbose bool) *EnhancedRouteProcessor {
	p.verbose = verbose
	return p
}

// ProcessEnhancedRoutes 处理增强路由
func (p *EnhancedRouteProcessor) ProcessEnhancedRoutes(paths []string) error {
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
	packageControllers := make(map[string]map[string]*EnhancedRouteInfo)

	// 处理每个注解
	for _, ann := range annotations {
		if commentAnn, ok := ann.(*types.CommentAnnotation); ok {
			if err := p.processEnhancedRouteAnnotation(commentAnn, packageControllers); err != nil {
				return err
			}
		}
	}

	// 为每个包和控制器生成增强路由代码
	for pkgPath, controllers := range packageControllers {
		for controllerName, routeInfo := range controllers {
			if err := p.generateEnhancedRouteCode(pkgPath, controllerName, routeInfo); err != nil {
				return fmt.Errorf("生成增强路由代码失败 for %s.%s: %w", pkgPath, controllerName, err)
			}
		}
	}

	return nil
}

// processEnhancedRouteAnnotation 处理单个增强路由注解
func (p *EnhancedRouteProcessor) processEnhancedRouteAnnotation(ann *types.CommentAnnotation, packageControllers map[string]map[string]*EnhancedRouteInfo) error {
	// 获取包路径
	pkgPath := filepath.Dir(ann.Position.Filename)
	pkgName := p.getPackageName(pkgPath)

	// 初始化包映射
	if _, ok := packageControllers[pkgPath]; !ok {
		packageControllers[pkgPath] = make(map[string]*EnhancedRouteInfo)
	}

	// 处理控制器注解
	if ann.GetType() == "controller" {
		controllerName := ann.TargetName
		pathPrefix := types.GetStringAttribute(ann.Attributes, "path", "")

		if _, ok := packageControllers[pkgPath][controllerName]; !ok {
			packageControllers[pkgPath][controllerName] = &EnhancedRouteInfo{
				ControllerName: controllerName,
				PathPrefix:     pathPrefix,
				PackageName:    pkgName,
				Imports:        make(map[string]string),
				Routes:         make([]*EnhancedRoute, 0),
			}
		}

		// 添加必要的导入
		p.addRequiredImports(packageControllers[pkgPath][controllerName])
	}

	// 处理路由注解
	if ann.GetType() == "route" {
		// 查找对应的控制器
		controllerName := p.findControllerName(ann)
		if controllerName == "" {
			return fmt.Errorf("无法找到路由 %s 对应的控制器", ann.TargetName)
		}

		if _, ok := packageControllers[pkgPath][controllerName]; !ok {
			packageControllers[pkgPath][controllerName] = &EnhancedRouteInfo{
				ControllerName: controllerName,
				PackageName:    pkgName,
				Imports:        make(map[string]string),
				Routes:         make([]*EnhancedRoute, 0),
			}
		}

		// 解析路由信息
		route, err := p.parseEnhancedRoute(ann)
		if err != nil {
			return fmt.Errorf("解析路由失败: %w", err)
		}

		// 添加到控制器信息中
		controllerInfo := packageControllers[pkgPath][controllerName]
		controllerInfo.Routes = append(controllerInfo.Routes, route)

		// 添加必要的导入
		p.addRequiredImports(controllerInfo)
	}

	return nil
}

// findControllerName 查找路由对应的控制器名称
func (p *EnhancedRouteProcessor) findControllerName(routeAnn *types.CommentAnnotation) string {
	// 这里需要实现查找控制器的逻辑
	// 可以通过解析AST或者维护注解关系来实现

	// 从注解的目标信息中获取控制器名称
	if routeAnn != nil && routeAnn.GetTargetName() != "" {
		// 如果注解有目标名称，直接使用
		return routeAnn.GetTargetName()
	}

	// 从注解的关联信息中查找
	if routeAnn != nil && routeAnn.GetTargetType() == types.TypeTarget {
		// 如果是类型注解，尝试从类型信息中获取
		return p.extractControllerNameFromType(routeAnn)
	}

	// 从文件路径和包信息中推断
	if routeAnn != nil && routeAnn.GetPosition().Filename != "" {
		return p.inferControllerNameFromPath(routeAnn.GetPosition().Filename)
	}

	// 默认返回
	return "DefaultController"
}

// extractControllerNameFromType 从类型信息中提取控制器名称
func (p *EnhancedRouteProcessor) extractControllerNameFromType(ann *types.CommentAnnotation) string {
	// 实现从类型信息中提取控制器名称的逻辑
	// 这里可以根据实际的注解结构来实现
	return "ExtractedController"
}

// inferControllerNameFromPath 从文件路径推断控制器名称
func (p *EnhancedRouteProcessor) inferControllerNameFromPath(filePath string) string {
	// 从文件路径中提取控制器名称
	// 例如：/path/to/user_controller.go -> UserController
	fileName := filepath.Base(filePath)

	// 移除扩展名
	fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))

	// 转换为驼峰命名
	parts := strings.Split(fileName, "_")
	var result string
	for _, part := range parts {
		if len(part) > 0 {
			result += strings.Title(part)
		}
	}

	if result == "" {
		return "InferredController"
	}

	return result
}

// parseEnhancedRoute 解析增强路由信息
func (p *EnhancedRouteProcessor) parseEnhancedRoute(ann *types.CommentAnnotation) (*EnhancedRoute, error) {
	route := &EnhancedRoute{}

	// 获取基本路由信息
	route.Method = types.GetStringAttribute(ann.Attributes, "method", "GET")
	route.Path = types.GetStringAttribute(ann.Attributes, "path", "")
	route.MethodName = ann.TargetName

	// 解析方法参数（通过AST解析）
	params, err := p.parseMethodParams(ann)
	if err != nil {
		return nil, fmt.Errorf("解析方法参数失败: %w", err)
	}
	route.Params = params

	// 解析方法返回值（通过AST解析）
	returns, err := p.parseMethodReturns(ann)
	if err != nil {
		return nil, fmt.Errorf("解析方法返回值失败: %w", err)
	}
	route.Returns = returns

	// 解析认证信息
	route.Auth = p.parseRouteAuth(ann)

	// 解析验证信息
	route.Validation = p.parseRouteValidation(ann)

	// 解析绑定信息
	route.Binding = p.parseRouteBinding(ann)

	return route, nil
}

// parseMethodParams 解析方法参数
func (p *EnhancedRouteProcessor) parseMethodParams(ann *types.CommentAnnotation) ([]*RouteParam, error) {
	// 这里需要实现通过AST解析方法参数的逻辑

	var params []*RouteParam

	// 从注解的AST节点中解析参数
	if ann.Node != nil {
		if funcDecl, ok := ann.Node.(*ast.FuncDecl); ok && funcDecl.Type != nil {
			if funcDecl.Type.Params != nil {
				for _, param := range funcDecl.Type.Params.List {
					paramType := p.extractTypeName(param.Type)
					for _, name := range param.Names {
						param := &RouteParam{
							Name: name.Name,
							Type: paramType,
						}
						params = append(params, param)
					}
				}
			}
		}
	}

	// 如果AST解析失败，尝试从注解属性中获取
	if len(params) == 0 {
		params = p.parseParamsFromAttributes(ann)
	}

	return params, nil
}

// extractTypeName 提取类型名称
func (p *EnhancedRouteProcessor) extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + p.extractTypeName(t.X)
	case *ast.ArrayType:
		return "[]" + p.extractTypeName(t.Elt)
	case *ast.SelectorExpr:
		return p.extractTypeName(t.X) + "." + t.Sel.Name
	default:
		return "interface{}"
	}
}

// parseParamsFromAttributes 从注解属性中解析参数
func (p *EnhancedRouteProcessor) parseParamsFromAttributes(ann *types.CommentAnnotation) []*RouteParam {
	var params []*RouteParam

	// 尝试从注解属性中获取参数信息
	for _, attr := range ann.GetAttributes() {
		if attr.Name == "params" {
			if paramStr, ok := attr.Value.(string); ok {
				// 解析参数字符串，格式：name:type,name2:type2
				paramPairs := strings.Split(paramStr, ",")
				for _, pair := range paramPairs {
					parts := strings.Split(strings.TrimSpace(pair), ":")
					if len(parts) == 2 {
						params = append(params, &RouteParam{
							Name: strings.TrimSpace(parts[0]),
							Type: strings.TrimSpace(parts[1]),
						})
					}
				}
			}
		}
	}

	return params
}

// parseMethodReturns 解析方法返回值
func (p *EnhancedRouteProcessor) parseMethodReturns(ann *types.CommentAnnotation) ([]*RouteReturn, error) {
	// 这里需要实现通过AST解析方法返回值的逻辑

	var returns []*RouteReturn

	// 从注解的AST节点中解析返回值
	if ann.Node != nil {
		if funcDecl, ok := ann.Node.(*ast.FuncDecl); ok && funcDecl.Type != nil {
			if funcDecl.Type.Results != nil {
				for _, result := range funcDecl.Type.Results.List {
					returnType := p.extractTypeName(result.Type)
					if len(result.Names) > 0 {
						for range result.Names {
							returns = append(returns, &RouteReturn{
								Type:    returnType,
								IsError: returnType == "error",
							})
						}
					} else {
						// 匿名返回值
						returns = append(returns, &RouteReturn{
							Type:    returnType,
							IsError: returnType == "error",
						})
					}
				}
			}
		}
	}

	// 如果AST解析失败，尝试从注解属性中获取
	if len(returns) == 0 {
		returns = p.parseReturnsFromAttributes(ann)
	}

	return returns, nil
}

// parseReturnsFromAttributes 从注解属性中解析返回值
func (p *EnhancedRouteProcessor) parseReturnsFromAttributes(ann *types.CommentAnnotation) []*RouteReturn {
	var returns []*RouteReturn

	// 尝试从注解属性中获取返回值信息
	for _, attr := range ann.GetAttributes() {
		if attr.Name == "returns" {
			if returnStr, ok := attr.Value.(string); ok {
				// 解析返回值字符串，格式：name:type,name2:type2
				returnPairs := strings.Split(returnStr, ",")
				for _, pair := range returnPairs {
					parts := strings.Split(strings.TrimSpace(pair), ":")
					if len(parts) == 2 {
						returnType := strings.TrimSpace(parts[1])
						returns = append(returns, &RouteReturn{
							Type:    returnType,
							IsError: returnType == "error",
						})
					} else if len(parts) == 1 {
						// 只有类型，没有名称
						returnType := strings.TrimSpace(parts[0])
						returns = append(returns, &RouteReturn{
							Type:    returnType,
							IsError: returnType == "error",
						})
					}
				}
			}
		}
	}

	return returns
}

// parseRouteAuth 解析路由认证信息
func (p *EnhancedRouteProcessor) parseRouteAuth(ann *types.CommentAnnotation) *RouteAuth {
	// 查找认证注解
	authAnno := p.findAuthAnnotation(ann)
	if authAnno == nil {
		return nil
	}

	auth := &RouteAuth{}
	roles, found := types.GetStringSliceAttribute(authAnno.Attributes, "roles")
	if found {
		auth.RequiredRoles = roles
	}

	permissions, found := types.GetStringSliceAttribute(authAnno.Attributes, "permissions")
	if found {
		auth.RequiredPermissions = permissions
	}

	return auth
}

// parseRouteValidation 解析路由验证信息
func (p *EnhancedRouteProcessor) parseRouteValidation(ann *types.CommentAnnotation) *RouteValidation {
	// 查找验证注解
	validationAnno := p.findValidationAnnotation(ann)
	if validationAnno == nil {
		return nil
	}

	validation := &RouteValidation{}
	validation.Rules = types.GetStringAttribute(validationAnno.Attributes, "rules", "")
	groups, found := types.GetStringSliceAttribute(validationAnno.Attributes, "groups")
	if found {
		validation.Groups = groups
	}

	return validation
}

// parseRouteBinding 解析路由绑定信息
func (p *EnhancedRouteProcessor) parseRouteBinding(ann *types.CommentAnnotation) *RouteBinding {
	// 查找绑定注解
	bindingAnno := p.findBindingAnnotation(ann)
	if bindingAnno == nil {
		return nil
	}

	binding := &RouteBinding{}
	binding.Source = types.GetStringAttribute(bindingAnno.Attributes, "source", "json")
	binding.Model = types.GetStringAttribute(bindingAnno.Attributes, "model", "")

	return binding
}

// findAuthAnnotation 查找认证注解
func (p *EnhancedRouteProcessor) findAuthAnnotation(ann *types.CommentAnnotation) *types.CommentAnnotation {
	// 这里需要实现查找认证注解的逻辑
	// 暂时返回nil，实际实现中需要完善
	return nil
}

// findValidationAnnotation 查找验证注解
func (p *EnhancedRouteProcessor) findValidationAnnotation(ann *types.CommentAnnotation) *types.CommentAnnotation {
	// 这里需要实现查找验证注解的逻辑
	// 暂时返回nil，实际实现中需要完善
	return nil
}

// findBindingAnnotation 查找绑定注解
func (p *EnhancedRouteProcessor) findBindingAnnotation(ann *types.CommentAnnotation) *types.CommentAnnotation {
	// 这里需要实现查找绑定注解的逻辑
	// 暂时返回nil，实际实现中需要完善
	return nil
}

// addRequiredImports 添加必要的导入
func (p *EnhancedRouteProcessor) addRequiredImports(controllerInfo *EnhancedRouteInfo) {
	// 添加基础导入
	controllerInfo.Imports["github.com/gin-gonic/gin"] = ""
	controllerInfo.Imports["net/http"] = ""

	// 根据路由信息添加必要的导入
	for _, route := range controllerInfo.Routes {
		if route.Auth != nil && (len(route.Auth.RequiredRoles) > 0 || len(route.Auth.RequiredPermissions) > 0) {
			controllerInfo.Imports["github.com/grain-framework/grain/pkg/auth"] = ""
		}

		if route.Validation != nil && route.Validation.Rules != "" {
			controllerInfo.Imports["github.com/go-playground/validator/v10"] = "validator"
		}

		if route.Binding != nil && route.Binding.Source != "" {
			controllerInfo.Imports["github.com/gin-gonic/gin/binding"] = ""
		}
	}
}

// getPackageName 获取包名
func (p *EnhancedRouteProcessor) getPackageName(pkgPath string) string {
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

// generateEnhancedRouteCode 生成增强路由代码
func (p *EnhancedRouteProcessor) generateEnhancedRouteCode(pkgPath, controllerName string, routeInfo *EnhancedRouteInfo) error {
	// 创建输出目录
	outputDir := filepath.Join(p.outputPath, pkgPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 创建输出文件
	outputFile := filepath.Join(outputDir, strings.ToLower(controllerName)+"_enhanced_route_gen.go")

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer file.Close()

	// 执行模板
	if err := p.tmpl.Execute(file, routeInfo); err != nil {
		return fmt.Errorf("生成增强路由代码失败: %w", err)
	}

	if p.verbose {
		fmt.Printf("增强路由代码生成成功: %s\n", outputFile)
	}

	return nil
}

// 增强路由代码模板
const enhancedRouteTemplate = `// Code generated by gRain. DO NOT EDIT.
// 增强路由代码，支持自动参数绑定

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

// {{.ControllerName}}EnhancedRouter {{.ControllerName}}增强路由器
type {{.ControllerName}}EnhancedRouter struct {
	controller interface{}
}

// New{{.ControllerName}}EnhancedRouter 创建增强路由器
func New{{.ControllerName}}EnhancedRouter(controller interface{}) *{{.ControllerName}}EnhancedRouter {
	return &{{.ControllerName}}EnhancedRouter{
		controller: controller,
	}
}

// RegisterRoutes 注册路由到给定的路由器
func (r *{{.ControllerName}}EnhancedRouter) RegisterRoutes(router gin.IRouter) {
	{{if .PathPrefix}}
	// 使用路由前缀
	group := router.Group("{{.PathPrefix}}")
	{{else}}
	// 没有路由前缀
	group := router.Group("")
	{{end}}

	{{range .Routes}}
	// {{.MethodName}} 路由
	group.{{.Method}}("{{.Path}}", r.{{.MethodName}}Handler)
	{{end}}
}

{{range .Routes}}
// {{.MethodName}}Handler {{.MethodName}}处理器
func (r *{{$.ControllerName}}EnhancedRouter) {{.MethodName}}Handler(c *gin.Context) {
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

	{{if .Auth}}
	// 权限检查
	{{if .Auth.RequiredRoles}}
	if err := r.checkRoles(c, []string{ {{range .Auth.RequiredRoles}}"{{.}}", {{end}} }); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "权限不足",
			"error":   err.Error(),
		})
		return
	}
	{{end}}
	{{if .Auth.RequiredPermissions}}
	if err := r.checkPermissions(c, []string{ {{range .Auth.RequiredPermissions}}"{{.}}", {{end}} }); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "权限不足",
			"error":   err.Error(),
		})
		return
	}
	{{end}}
	{{end}}

	{{if .Validation}}
	// 参数验证
	{{if .Validation.Rules}}
	if err := r.validateParams(c, {{range $index, $param := $.Params}}{{if $index}}, {{end}}{{$param.Name}}{{end}}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数验证失败",
			"error":   err.Error(),
		})
		return
	}
	{{end}}
	{{end}}

	// 调用原始控制器方法
	{{if .Params}}
	// 使用反射调用方法
	method := reflect.ValueOf(r.controller).MethodByName("{{.MethodName}}")
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
	method := reflect.ValueOf(r.controller).MethodByName("{{.MethodName}}")
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

// 权限检查方法
func (r *{{.ControllerName}}EnhancedRouter) checkRoles(c *gin.Context, roles []string) error {
	// 实现角色检查逻辑
	// 这里应该集成到实际的认证系统中
	return nil
}

func (r *{{.ControllerName}}EnhancedRouter) checkPermissions(c *gin.Context, permissions []string) error {
	// 实现权限检查逻辑
	// 这里应该集成到实际的认证系统中
	return nil
}

// 参数验证方法
func (r *{{.ControllerName}}EnhancedRouter) validateParams(c *gin.Context, params ...interface{}) error {
	// 实现参数验证逻辑
	// 这里应该集成到实际的验证系统中
	return nil
}

// 自动参数绑定中间件
func AutoParamBinding{{.ControllerName}}(controller interface{}) gin.HandlerFunc {
	router := New{{.ControllerName}}EnhancedRouter(controller)
	
	return func(c *gin.Context) {
		// 根据请求路径和方法自动调用对应的处理器
		path := c.Request.URL.Path
		method := c.Request.Method
		
		// 这里可以根据路径和方法自动路由到对应的处理器
		// 暂时使用一个通用的处理方式
		for _, route := range router.Routes {
			if strings.Contains(path, route.Path) && method == route.Method {
				// 调用对应的处理器
				switch route.MethodName {
				{{range .Routes}}
				case "{{.MethodName}}":
					router.{{.MethodName}}Handler(c)
					c.Abort() // 阻止继续执行
					return
				{{end}}
				}
			}
		}
		
		c.Next()
	}
}
`
