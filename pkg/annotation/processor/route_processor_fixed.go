package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grain-framework/grain/pkg/annotation/registry"
	anntypes "github.com/grain-framework/grain/pkg/annotation/types"
)

// FixedRouteProcessor 修复后的路由处理器，解决重复声明问题
type FixedRouteProcessor struct {
	registry  registry.Registry
	prefix    string
	outputDir string
}

// NewFixedRouteProcessor 创建修复后的路由处理器
func NewFixedRouteProcessor(reg registry.Registry, prefix, outputDir string) *FixedRouteProcessor {
	return &FixedRouteProcessor{
		registry:  reg,
		prefix:    prefix,
		outputDir: outputDir,
	}
}

// ProcessRoutes 处理路由注解，避免重复声明
func (p *FixedRouteProcessor) ProcessRoutes() error {
	// 获取所有控制器注解
	controllers := p.registry.FindByType(anntypes.ControllerType)

	// 为每个控制器生成路由文件
	for _, controller := range controllers {
		if err := p.generateControllerRoutes(controller); err != nil {
			return fmt.Errorf("failed to generate routes for controller %s: %w", controller.GetTargetName(), err)
		}
	}

	// 生成通用接口文件（只生成一次）
	if err := p.generateCommonInterfaces(); err != nil {
		return fmt.Errorf("failed to generate common interfaces: %w", err)
	}

	return nil
}

// generateControllerRoutes 为单个控制器生成路由文件
func (p *FixedRouteProcessor) generateControllerRoutes(controller anntypes.Annotation) error {
	// 解析包路径
	pkgPath := p.resolvePackagePath(controller.GetPosition().Filename)

	// 创建输出目录
	outputDir := filepath.Join(p.outputDir, "route", pkgPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 生成文件名
	fileName := fmt.Sprintf("%s_route_gen.go", strings.ToLower(controller.GetTargetName()))
	filePath := filepath.Join(outputDir, fileName)

	// 生成路由代码
	code := p.generateRouteCode(controller, pkgPath)

	// 写入文件
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		return fmt.Errorf("failed to write route file: %w", err)
	}

	return nil
}

// generateRouteCode 生成路由代码，不包含通用接口定义
func (p *FixedRouteProcessor) generateRouteCode(controller anntypes.Annotation, pkgPath string) string {
	// 只生成控制器特定的代码，不包含通用接口
	code := fmt.Sprintf(`package %s

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"reflect"
	"strings"
)

// RegisterRoutes 注册路由到给定的路由器
func (c *%s) RegisterRoutes(router gin.IRouter) {
	// 实现路由注册逻辑
	// 这里可以根据注解信息生成具体的路由
	
	// 获取控制器的方法信息
	controllerType := reflect.TypeOf(c)
	for i := 0; i < controllerType.NumMethod(); i++ {
		method := controllerType.Method(i)
		methodName := method.Name
		
		// 根据方法名推断路由信息
		if strings.HasPrefix(methodName, "List") {
			entityName := strings.TrimPrefix(methodName, "List")
			router.GET("/"+strings.ToLower(entityName), c.handleList(methodName))
		} else if strings.HasPrefix(methodName, "Get") && strings.HasSuffix(methodName, "ByID") {
			entityName := strings.TrimPrefix(methodName, "Get")
			entityName = strings.TrimSuffix(entityName, "ByID")
			router.GET("/"+strings.ToLower(entityName)+"/:id", c.handleGetByID(methodName))
		} else if strings.HasPrefix(methodName, "Create") {
			entityName := strings.TrimPrefix(methodName, "Create")
			router.POST("/"+strings.ToLower(entityName), c.handleCreate(methodName))
		} else if strings.HasPrefix(methodName, "Update") {
			entityName := strings.TrimPrefix(methodName, "Update")
			router.PUT("/"+strings.ToLower(entityName)+"/:id", c.handleUpdate(methodName))
		} else if strings.HasPrefix(methodName, "Delete") {
			entityName := strings.TrimPrefix(methodName, "Delete")
			router.DELETE("/"+strings.ToLower(entityName)+"/:id", c.handleDelete(methodName))
		} else if strings.HasPrefix(methodName, "Patch") {
			entityName := strings.TrimPrefix(methodName, "Patch")
			router.PATCH("/"+strings.ToLower(entityName)+"/:id", c.handlePatch(methodName))
		}
	}
}

// 辅助方法，用于处理不同类型的请求
func (c *%s) handleList(methodName string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 调用实际的列表方法
		method := reflect.ValueOf(c).MethodByName(methodName)
		if method.IsValid() {
			results := method.Call([]reflect.Value{reflect.ValueOf(ctx)})
			// 处理返回值
			if len(results) > 0 {
				ctx.JSON(http.StatusOK, results[0].Interface())
			}
		}
	}
}

func (c *%s) handleGetByID(methodName string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.Param("id")
		ctx.Set("id", id)
		
		method := reflect.ValueOf(c).MethodByName(methodName)
		if method.IsValid() {
			results := method.Call([]reflect.Value{reflect.ValueOf(ctx)})
			if len(results) > 0 {
				ctx.JSON(http.StatusOK, results[0].Interface())
			}
		}
	}
}

func (c *%s) handleCreate(methodName string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		method := reflect.ValueOf(c).MethodByName(methodName)
		if method.IsValid() {
			results := method.Call([]reflect.Value{reflect.ValueOf(ctx)})
			if len(results) > 0 {
				ctx.JSON(http.StatusCreated, results[0].Interface())
			}
		}
	}
}

func (c *%s) handleUpdate(methodName string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.Param("id")
		ctx.Set("id", id)
		
		method := reflect.ValueOf(c).MethodByName(methodName)
		if method.IsValid() {
			results := method.Call([]reflect.Value{reflect.ValueOf(ctx)})
			if len(results) > 0 {
				ctx.JSON(http.StatusOK, results[0].Interface())
			}
		}
	}
}

func (c *%s) handleDelete(methodName string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.Param(ctx.Param("id"))
		ctx.Set("id", id)
		
		method := reflect.ValueOf(c).MethodByName(methodName)
		if method.IsValid() {
			results := method.Call([]reflect.Value{reflect.ValueOf(ctx)})
			if len(results) > 0 {
				ctx.JSON(http.StatusOK, results[0].Interface())
			}
		}
	}
}

func (c *%s) handlePatch(methodName string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.Param("id")
		ctx.Set("id", id)
		
		method := reflect.ValueOf(c).MethodByName(methodName)
		if method.IsValid() {
			results := method.Call([]reflect.Value{reflect.ValueOf(ctx)})
			if len(results) > 0 {
				ctx.JSON(http.StatusOK, results[0].Interface())
			}
		}
	}
}
`, pkgPath, controller.GetTargetName())

	return code
}

// generateCommonInterfaces 生成通用接口文件，避免重复声明
func (p *FixedRouteProcessor) generateCommonInterfaces() error {
	commonDir := filepath.Join(p.outputDir, "common")
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

// resolvePackagePath 解析包路径，返回相对路径
func (p *FixedRouteProcessor) resolvePackagePath(filename string) string {
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
