package processor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/isBlue-5/grain/pkg/annotation/registry"
	anntypes "github.com/isBlue-5/grain/pkg/annotation/types"
)

// RouteProcessor 路由处理器
type RouteProcessor struct {
	// 注解注册中心
	registry registry.Registry

	// 注解前缀
	prefix string

	// 输出目录
	outputDir string
}

// NewRouteProcessor 创建路由处理器
func NewRouteProcessor(reg registry.Registry, prefix, outputDir string) *RouteProcessor {
	return &RouteProcessor{
		registry:  reg,
		prefix:    prefix,
		outputDir: outputDir,
	}
}

// ProcessRoute 处理路由注解
func (p *RouteProcessor) ProcessRoute(pkgPaths []string) error {
	fmt.Printf("DEBUG: 开始处理路由注解...\n")

	// 查找所有控制器和路由注解
	controllers := p.collectControllerInfo()
	routes := p.collectRouteInfo()

	fmt.Printf("DEBUG: 找到 %d 个控制器, %d 个路由\n", len(controllers), len(routes))

	// 将路由归类到控制器
	controllerRoutes := make(map[string][]RouteInfo)
	for _, route := range routes {
		controllerRoutes[route.ReceiverType] = append(controllerRoutes[route.ReceiverType], route)
	}

	fmt.Printf("DEBUG: 路由归类结果: %+v\n", controllerRoutes)

	// 生成路由注册代码
	for _, controller := range controllers {
		routes := controllerRoutes[controller.StructName]
		fmt.Printf("DEBUG: 控制器 %s 有 %d 个路由\n", controller.StructName, len(routes))
		if len(routes) > 0 {
			// 创建路由模板数据
			data := routeTemplateData{
				PackageName:    "main", // 使用固定的包名，避免绝对路径问题
				ControllerName: controller.StructName,
				PathPrefix:     controller.PathPrefix,
				Routes:         make([]routeInfo, 0),
				Imports:        []string{},
			}

			// 转换路由信息
			for _, route := range routes {
				routeInfo := routeInfo{
					MethodName:      route.FuncName,
					HttpMethod:      route.Method,
					Path:            route.Path,
					HasValidation:   route.HasValidation,
					BindModel:       route.BindModel,
					BindSourceUpper: strings.ToUpper(route.BindSource),
					RequiredRoles:   route.RequiredRoles,
				}
				data.Routes = append(data.Routes, routeInfo)
			}

			// 生成路由代码
			if err := p.generateRouteCode(data); err != nil {
				return fmt.Errorf("生成路由代码失败: %w", err)
			}
		}
	}

	fmt.Printf("DEBUG: 路由注解处理完成\n")
	return nil
}

// ControllerInfo 控制器信息
type ControllerInfo struct {
	// 结构体名称
	StructName string

	// 包路径
	PkgPath string

	// 路由前缀
	PathPrefix string

	// 原始注解
	Annotation anntypes.Annotation
}

// RouteInfo 路由信息
type RouteInfo struct {
	// 接收者类型
	ReceiverType string

	// 函数名称
	FuncName string

	// HTTP方法
	Method string

	// 路径
	Path string

	// 源码位置
	Position anntypes.Position

	// 所需角色
	RequiredRoles []string

	// 所需权限
	RequiredPermissions []string

	// 是否需要验证
	HasValidation bool

	// 验证规则
	ValidationRules string

	// 验证组
	ValidateGroups []string

	// 绑定来源
	BindSource string

	// 绑定模型
	BindModel string

	// 包路径
	PkgPath string

	// 中间件
	Middlewares []string

	// 原始注解
	Annotation anntypes.Annotation
}

// 收集控制器信息
func (p *RouteProcessor) collectControllerInfo() []ControllerInfo {
	var controllers []ControllerInfo

	// 查找所有controller注解
	annotations := p.registry.FindByType(anntypes.ControllerType)
	fmt.Printf("DEBUG: 找到 %d 个ControllerType注解\n", len(annotations))

	// 修复：也查找"controller"类型的注解
	controllerAnnotations := p.registry.FindByType("controller")
	fmt.Printf("DEBUG: 找到 %d 个controller注解\n", len(controllerAnnotations))
	annotations = append(annotations, controllerAnnotations...)

	for _, anno := range annotations {
		fmt.Printf("DEBUG: 处理注解: 类型=%s, 目标类型=%s, 目标名称=%s\n",
			anno.GetType(), anno.GetTargetType(), anno.GetTargetName())

		// 确保目标是类型
		if anno.GetTargetType() != anntypes.TypeTarget {
			fmt.Printf("DEBUG: 跳过非类型目标注解\n")
			continue
		}

		info := ControllerInfo{
			StructName: anno.GetTargetName(),
			PkgPath:    filepath.Dir(anno.GetPosition().Filename),
			PathPrefix: anntypes.GetStringAttribute(anno.GetAttributes(), "path", ""),
			Annotation: anno,
		}

		controllers = append(controllers, info)
		fmt.Printf("DEBUG: 添加控制器: %s\n", info.StructName)
	}

	return controllers
}

// 收集路由信息
func (p *RouteProcessor) collectRouteInfo() []RouteInfo {
	var routes []RouteInfo

	// 查找所有route注解
	annotations := p.registry.FindByType(anntypes.RouteType)
	fmt.Printf("DEBUG: 找到 %d 个RouteType注解\n", len(annotations))

	// 修复：也查找"route"类型的注解
	routeAnnotations := p.registry.FindByType("route")
	fmt.Printf("DEBUG: 找到 %d 个route注解\n", len(routeAnnotations))
	annotations = append(annotations, routeAnnotations...)

	for _, anno := range annotations {
		fmt.Printf("DEBUG: 处理路由注解: 类型=%s, 目标类型=%s, 目标名称=%s\n",
			anno.GetType(), anno.GetTargetType(), anno.GetTargetName())

		// 确保目标是方法
		if anno.GetTargetType() != anntypes.MethodTarget {
			fmt.Printf("DEBUG: 跳过非方法目标注解\n")
			continue
		}

		info, err := p.processRoute(anno)
		if err != nil {
			fmt.Printf("DEBUG: 处理路由失败: %v\n", err)
			// 处理错误或跳过
			continue
		}

		// 添加包路径
		info.PkgPath = p.resolvePackagePath(anno.GetPosition().Filename)
		info.Annotation = anno

		routes = append(routes, info)
		fmt.Printf("DEBUG: 添加路由: %s -> %s\n", info.ReceiverType, info.FuncName)
	}

	return routes
}

// processRoute 处理路由注解
func (p *RouteProcessor) processRoute(anno anntypes.Annotation) (RouteInfo, error) {
	info := RouteInfo{}

	// 获取方法信息
	info.Method = anntypes.GetStringAttribute(anno.GetAttributes(), "method", "GET")
	info.Path = anntypes.GetStringAttribute(anno.GetAttributes(), "path", "")

	// 如果是注释注解，获取接收者类型和函数名
	if commentAnno, ok := anno.(*anntypes.CommentAnnotation); ok {
		info.ReceiverType = commentAnno.ReceiverType
		info.FuncName = anno.GetTargetName()
		info.Position = anno.GetPosition()
	}

	// 处理其他注解（如auth、validation等）
	authAnno := findAuthAnnotation(anno.GetPosition().Filename, anno.GetTargetName())
	if authAnno != nil {
		// 处理认证注解
		roles, found := anntypes.GetStringSliceAttribute(authAnno.GetAttributes(), "roles")
		if found {
			info.RequiredRoles = roles
		}

		permissions, found := anntypes.GetStringSliceAttribute(authAnno.GetAttributes(), "permissions")
		if found {
			info.RequiredPermissions = permissions
		}
	}

	// 处理验证注解
	validationAnno := findValidationAnnotation(anno.GetPosition().Filename, anno.GetTargetName())
	if validationAnno != nil {
		info.HasValidation = true
		info.ValidationRules = anntypes.GetStringAttribute(validationAnno.GetAttributes(), "rules", "")
		info.ValidateGroups, _ = anntypes.GetStringSliceAttribute(validationAnno.GetAttributes(), "groups")
	}

	// 处理绑定注解
	bindAnno := findBindAnnotation(anno.GetPosition().Filename, anno.GetTargetName())
	if bindAnno != nil {
		info.BindSource = anntypes.GetStringAttribute(bindAnno.GetAttributes(), "source", "json")
		info.BindModel = anntypes.GetStringAttribute(bindAnno.GetAttributes(), "model", "")
	}

	return info, nil
}

// 辅助函数，查找认证注解
func findAuthAnnotation(filename, targetName string) anntypes.Annotation {
	// 实现查找认证注解的逻辑
	// 在实际实现中，这应该通过注解注册中心查询

	// 尝试从文件名推断认证需求
	if strings.Contains(filename, "admin") || strings.Contains(filename, "Admin") {
		// 管理员相关文件，返回管理员认证注解
		return &anntypes.CommentAnnotation{
			Type:       anntypes.AuthType,
			TargetName: targetName,
			Attributes: []anntypes.AnnotationAttribute{
				{Name: "required", Value: true},
				{Name: "roles", Value: []string{"admin"}},
			},
		}
	}

	if strings.Contains(filename, "user") || strings.Contains(filename, "User") {
		// 用户相关文件，返回用户认证注解
		return &anntypes.CommentAnnotation{
			Type:       anntypes.AuthType,
			TargetName: targetName,
			Attributes: []anntypes.AnnotationAttribute{
				{Name: "required", Value: true},
				{Name: "roles", Value: []string{"user", "admin"}},
			},
		}
	}

	if strings.Contains(filename, "public") || strings.Contains(filename, "Public") {
		// 公开文件，返回公开访问注解
		return &anntypes.CommentAnnotation{
			Type:       anntypes.AuthType,
			TargetName: targetName,
			Attributes: []anntypes.AnnotationAttribute{
				{Name: "required", Value: false},
				{Name: "roles", Value: []string{}},
			},
		}
	}

	// 默认返回需要认证的注解
	return &anntypes.CommentAnnotation{
		Type:       anntypes.AuthType,
		TargetName: targetName,
		Attributes: []anntypes.AnnotationAttribute{
			{Name: "required", Value: true},
			{Name: "roles", Value: []string{"user"}},
		},
	}
}

// 辅助函数，查找验证注解
func findValidationAnnotation(filename, targetName string) anntypes.Annotation {
	// 实现查找验证注解的逻辑...
	return nil
}

// 辅助函数，查找绑定注解
func findBindAnnotation(filename, targetName string) anntypes.Annotation {
	// 实现查找绑定注解的逻辑...
	return nil
}

// generateRouteFile 生成路由文件
func (p *RouteProcessor) generateRouteFile(controller *ControllerInfo, routes []RouteInfo, outputPath string) error {
	// 解析包路径
	pkgPath := p.resolvePackagePath(controller.PkgPath)

	// 解析输出路径
	outputDir := p.resolveOutputPath(pkgPath, "route")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 准备模板数据
	data := routeTemplateData{
		PackageName:    pkgPath,
		ControllerName: controller.StructName,
		PathPrefix:     controller.PathPrefix,
		Routes:         make([]routeInfo, 0),
		Imports:        []string{},
	}

	// 处理路由 - 根据实际的注解数据来构建
	for _, route := range routes {
		routeInfo := routeInfo{
			MethodName:    route.FuncName,
			HttpMethod:    route.Method,
			Path:          route.Path,
			Controller:    route.ReceiverType,
			HasValidation: route.HasValidation,
			BindModel:     route.BindModel,
			BindSource:    route.BindSource,
			RequiredRoles: route.RequiredRoles,
			Middlewares:   route.Middlewares,
		}
		data.Routes = append(data.Routes, routeInfo)
	}

	// 生成代码并写入文件
	if err := p.generateRouteCode(data); err != nil {
		return fmt.Errorf("failed to generate route code: %w", err)
	}

	return nil
}

// generateRouteCode 生成路由代码，避免重复声明
func (p *RouteProcessor) generateRouteCode(data routeTemplateData) error {
	fmt.Printf("DEBUG: 开始生成路由代码，控制器: %s, 输出目录: %s\n", data.ControllerName, p.outputDir)

	// 使用text/template来生成代码，避免fmt.Sprintf参数错误
	tmpl, err := template.New("route").Parse(`package {{.PackageName}}

import (
	"github.com/gin-gonic/gin"
	{{range .Imports}}
	"{{.}}"
	{{end}}
)

// RegisterRoutes 注册路由到给定的路由器
func (c *{{.ControllerName}}) RegisterRoutes(router gin.IRouter) {
	{{if .PathPrefix}}
	// 使用路由前缀
	group := router.Group("{{.PathPrefix}}")
	{{else}}
	// 没有路由前缀
	group := router.Group("")
	{{end}}

	{{range .Routes}}
	// {{.MethodName}} 路由
	group.{{.HttpMethod}}("{{.Path}}", func(ctx *gin.Context) {
		{{if .HasValidation}}
		// 参数验证
		var req {{.BindModel}}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		
		// 自定义验证
		if validator, ok := interface{}(&req).(Validator); ok {
			if err := validator.Validate(); err != nil {
				ctx.JSON(400, gin.H{"error": err.Error()})
				return
			}
		}
		{{end}}
		
		{{if .RequiredRoles}}
		// 权限检查
		if err := checkRoles(ctx, []string{ {{range .RequiredRoles}}"{{.}}", {{end}} }); err != nil {
			ctx.JSON(403, gin.H{"error": err.Error()})
			return
		}
		{{end}}
		
		c.{{.MethodName}}(ctx)
	})
	{{end}}
}
`)
	if err != nil {
		return fmt.Errorf("failed to parse route template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute route template: %w", err)
	}

	// 创建输出目录
	outputDir := p.outputDir // 直接使用输出目录，不创建route子目录
	fmt.Printf("DEBUG: 创建输出目录: %s\n", outputDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 生成输出文件名
	outputFile := filepath.Join(outputDir, generateFriendlyFileName(data.ControllerName)+"_route_gen.go")
	fmt.Printf("DEBUG: 生成输出文件: %s\n", outputFile)

	// 写入文件
	if err := os.WriteFile(outputFile, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write route file: %w", err)
	}

	fmt.Printf("DEBUG: 路由文件生成成功: %s\n", outputFile)
	return nil
}

// generateCommonInterfaces 生成通用接口文件，避免重复声明
func (p *RouteProcessor) generateCommonInterfaces(outputPath string) error {
	commonDir := filepath.Join(outputPath, "common")
	if err := os.MkdirAll(commonDir, 0755); err != nil {
		return fmt.Errorf("failed to create common directory: %w", err)
	}

	commonFile := filepath.Join(commonDir, "interfaces.go")
	commonCode := `package common

import "github.com/gin-gonic/gin"

// Controller 接口定义
type Controller interface {
	RegisterRoutes(router gin.IRouter)
}

// Validator 自定义验证接口
type Validator interface {
	Validate() error
}

// 检查角色权限
func checkRoles(c *gin.Context, roles []string) error {
	// 实现角色检查逻辑
	return nil
}
`

	if err := os.WriteFile(commonFile, []byte(commonCode), 0644); err != nil {
		return fmt.Errorf("failed to write common interfaces file: %w", err)
	}

	return nil
}

// generateFriendlyFileName 生成友好的文件名
// 从UserController生成user，从ProductService生成product
func generateFriendlyFileName(structName string) string {
	// 移除常见的后缀：Controller, Service, Repository, Entity
	suffixes := []string{"Controller", "Service", "Repository", "Entity", "Manager", "Handler"}

	for _, suffix := range suffixes {
		if strings.HasSuffix(structName, suffix) {
			baseName := strings.TrimSuffix(structName, suffix)
			if baseName != "" {
				return strings.ToLower(baseName)
			}
		}
	}

	// 如果没有匹配的后缀，直接转换为小写
	return strings.ToLower(structName)
}

// 生成路由注册代码的数据结构
type routeTemplateData struct {
	PackageName    string
	Imports        []string
	ControllerName string
	PathPrefix     string
	Routes         []routeInfo
}

// 路由信息
type routeInfo struct {
	MethodName      string
	HttpMethod      string
	Path            string
	Controller      string
	HasValidation   bool
	BindModel       string
	BindSource      string
	BindSourceUpper string
	RequiredRoles   []string
	Middlewares     []string
}

// 路由注册代码模板
const routeTemplate = `// Code generated by gRain. DO NOT EDIT.
package {{.PackageName}}

import (
	"github.com/gin-gonic/gin"
	{{range .Imports}}
	"{{.}}"
	{{end}}
)

// RegisterRoutes 注册路由到给定的路由器
func (c *{{.ControllerName}}) RegisterRoutes(router gin.IRouter) {
	{{if .PathPrefix}}
	// 使用路由前缀
	group := router.Group("{{.PathPrefix}}")
	{{else}}
	// 没有路由前缀
	group := router.Group("")
	{{end}}

	{{range .Routes}}
	// {{.MethodName}} 路由
	group.{{.HttpMethod}}("{{.Path}}", func(ctx *gin.Context) {
		{{if .HasValidation}}
		// 参数验证
		var req {{.BindModel}}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(400, gin.H{"error": err.Error()})
			return
		}
		
		// 自定义验证
		if validator, ok := interface{}(&req).(Validator); ok {
			if err := validator.Validate(); err != nil {
				ctx.JSON(400, gin.H{"error": err.Error()})
				return
			}
		}
		{{end}}
		
		{{if .RequiredRoles}}
		// 权限检查
		if err := checkRoles(ctx, []string{ {{range .RequiredRoles}}"{{.}}", {{end}} }); err != nil {
			ctx.JSON(403, gin.H{"error": err.Error()})
			return
		}
		{{end}}
		
		c.{{.MethodName}}(ctx)
	})
	{{end}}
}

// 实现Controller接口
var _ Controller = (*{{.ControllerName}})(nil)

// Controller 接口定义
type Controller interface {
	RegisterRoutes(router gin.IRouter)
}

// Validator 自定义验证接口
type Validator interface {
	Validate() error
}

// 检查角色权限
func checkRoles(c *gin.Context, roles []string) error {
	// 实现角色检查逻辑
	return nil
}
`

// resolveOutputPath 统一的输出路径解析工具，避免深层嵌套和绝对路径问题
func (p *RouteProcessor) resolveOutputPath(pkgPath, subDir string) string {
	// 如果是测试环境或包含绝对路径，使用简化的输出目录
	if strings.Contains(pkgPath, os.TempDir()) || filepath.IsAbs(pkgPath) {
		return filepath.Join(p.outputDir, subDir)
	}

	// 生产环境：使用相对路径
	cleanPkgPath := strings.TrimPrefix(pkgPath, "./")
	cleanPkgPath = strings.TrimPrefix(cleanPkgPath, "../")

	// 避免太深的嵌套，只使用最后一级目录
	if strings.Contains(cleanPkgPath, "/") {
		cleanPkgPath = filepath.Base(cleanPkgPath)
	}

	return filepath.Join(p.outputDir, cleanPkgPath, subDir)
}

// resolvePackagePath 解析包路径，返回相对路径而不是绝对路径
func (p *RouteProcessor) resolvePackagePath(filename string) string {
	absPath := filepath.Dir(filename)

	// 尝试获取相对路径
	workDir, err := os.Getwd()
	if err == nil {
		if relPath, err := filepath.Rel(workDir, absPath); err == nil && !strings.HasPrefix(relPath, "..") {
			return relPath
		}
	}

	// 如果无法获取相对路径，使用目录名
	return filepath.Base(absPath)
}
