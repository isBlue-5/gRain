package core

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/isBlue-5/gRain/pkg/annotation/processor"
	"github.com/isBlue-5/gRain/pkg/annotation/registry"
)

// AutoGenerator 自动代码生成器
type AutoGenerator struct {
	workDir   string
	outputDir string
	registry  *registry.Registry
	generator *processor.SwaggerGenerator
	swaggerUI *processor.SwaggerServer
}

// NewAutoGenerator 创建新的自动生成器
func NewAutoGenerator(workDir string) *AutoGenerator {
	outputDir := filepath.Join(workDir, "generated")

	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create output directory: %v", err))
	}

	// 创建注解注册表
	reg := registry.NewRegistry()

	// 创建Swagger生成器
	swaggerGen := processor.NewSwaggerGenerator(reg, workDir)

	// 创建Swagger服务器
	swaggerServer := processor.NewSwaggerServer(swaggerGen)

	return &AutoGenerator{
		workDir:   workDir,
		outputDir: outputDir,
		registry:  reg,
		generator: swaggerGen,
		swaggerUI: swaggerServer,
	}
}

// AutoGenerate 自动生成所有代码
func (ag *AutoGenerator) AutoGenerate() error {
	fmt.Println("🚀 gRain框架自动代码生成开始...")
	startTime := time.Now()

	// 1. 扫描项目目录，解析注解
	if err := ag.scanAndParseAnnotations(); err != nil {
		return fmt.Errorf("failed to scan and parse annotations: %w", err)
	}

	// 2. 生成依赖注入代码
	if err := ag.generateDependencyInjection(); err != nil {
		return fmt.Errorf("failed to generate dependency injection: %w", err)
	}

	// 3. 生成路由注册代码
	if err := ag.generateRouteRegistration(); err != nil {
		return fmt.Errorf("failed to generate route registration: %w", err)
	}

	// 4. 生成Swagger文档
	if err := ag.generateSwaggerDocs(); err != nil {
		return fmt.Errorf("failed to generate swagger docs: %w", err)
	}

	// 5. 生成主注册函数
	if err := ag.generateMainRegistration(); err != nil {
		return fmt.Errorf("failed to generate main registration: %w", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("✅ gRain框架自动代码生成完成！耗时: %v\n", duration)

	return nil
}

// scanAndParseAnnotations 扫描并解析注解
func (ag *AutoGenerator) scanAndParseAnnotations() error {
	fmt.Println("📖 扫描项目目录，解析注解...")

	// 扫描控制器目录
	controllerDirs := []string{
		filepath.Join(ag.workDir, "controllers"),
		filepath.Join(ag.workDir, "handlers"),
		filepath.Join(ag.workDir, "api"),
	}

	for _, dir := range controllerDirs {
		if err := ag.scanDirectory(dir); err != nil {
			fmt.Printf("⚠️  扫描目录 %s 时出现警告: %v\n", dir, err)
		}
	}

	// 扫描模型目录
	modelDirs := []string{
		filepath.Join(ag.workDir, "models"),
		filepath.Join(ag.workDir, "entities"),
		filepath.Join(ag.workDir, "dto"),
	}

	for _, dir := range modelDirs {
		if err := ag.scanModels(dir); err != nil {
			fmt.Printf("⚠️  扫描模型目录 %s 时出现警告: %v\n", dir, err)
		}
	}

	fmt.Printf("📊 解析完成，共发现 %d 个注解\n", ag.registry.GetAnnotationCount())
	return nil
}

// scanDirectory 扫描目录
func (ag *AutoGenerator) scanDirectory(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil // 目录不存在，跳过
	}

	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// 解析Go文件
		return ag.parseGoFile(path)
	})
}

// scanModels 扫描模型目录
func (ag *AutoGenerator) scanModels(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil
	}

	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// 解析模型文件
		return ag.parseModelFile(path)
	})
}

// parseGoFile 解析Go文件
func (ag *AutoGenerator) parseGoFile(filePath string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	// 使用注解处理器解析文件
	processor := processor.NewAnnotationProcessor(ag.registry)
	return processor.ProcessFile(node, filePath)
}

// parseModelFile 解析模型文件
func (ag *AutoGenerator) parseModelFile(filePath string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	// 解析模型定义
	return ag.parseModelDefinitions(node, filePath)
}

// parseModelDefinitions 解析模型定义
func (ag *AutoGenerator) parseModelDefinitions(node *ast.File, filePath string) error {
	// 这里可以添加模型解析逻辑
	// 目前主要关注控制器注解
	return nil
}

// generateDependencyInjection 生成依赖注入代码
func (ag *AutoGenerator) generateDependencyInjection() error {
	fmt.Println("🔧 生成依赖注入代码...")

	// 获取所有需要依赖注入的类型
	injectTypes := ag.registry.GetInjectTypes()
	if len(injectTypes) == 0 {
		fmt.Println("ℹ️  没有发现需要依赖注入的类型")
		return nil
	}

	// 生成依赖注入代码
	code := ag.generateInjectCode(injectTypes)

	// 写入文件
	outputFile := filepath.Join(ag.outputDir, "dependency_injection.go")
	if err := os.WriteFile(outputFile, []byte(code), 0644); err != nil {
		return fmt.Errorf("failed to write dependency injection file: %w", err)
	}

	fmt.Printf("✅ 依赖注入代码生成完成: %s\n", outputFile)
	return nil
}

// generateInjectCode 生成依赖注入代码
func (ag *AutoGenerator) generateInjectCode(injectTypes []string) string {
	var code strings.Builder

	code.WriteString("// Code generated by gRain framework. DO NOT EDIT.\n")
	code.WriteString("package generated\n\n")
	code.WriteString("import (\n")
	code.WriteString("\t\"reflect\"\n")
	code.WriteString("\t\"sync\"\n")
	code.WriteString(")\n\n")

	code.WriteString("// Container 依赖注入容器\n")
	code.WriteString("type Container struct {\n")
	code.WriteString("\tservices map[reflect.Type]interface{}\n")
	code.WriteString("\tmutex   sync.RWMutex\n")
	code.WriteString("}\n\n")

	code.WriteString("// NewContainer 创建新的容器\n")
	code.WriteString("func NewContainer() *Container {\n")
	code.WriteString("\treturn &Container{\n")
	code.WriteString("\t\tservices: make(map[reflect.Type]interface{}),\n")
	code.WriteString("\t}\n")
	code.WriteString("}\n\n")

	code.WriteString("// Register 注册服务\n")
	code.WriteString("func (c *Container) Register(service interface{}) {\n")
	code.WriteString("\tc.mutex.Lock()\n")
	code.WriteString("\tdefer c.mutex.Unlock()\n")
	code.WriteString("\tc.services[reflect.TypeOf(service)] = service\n")
	code.WriteString("}\n\n")

	code.WriteString("// Get 获取服务\n")
	code.WriteString("func (c *Container) Get(serviceType reflect.Type) interface{} {\n")
	code.WriteString("\tc.mutex.RLock()\n")
	code.WriteString("\tdefer c.mutex.RUnlock()\n")
	code.WriteString("\treturn c.services[serviceType]\n")
	code.WriteString("}\n\n")

	// 生成注入函数
	code.WriteString("// InjectDependencies 注入依赖\n")
	code.WriteString("func InjectDependencies(container *Container) {\n")
	for _, injectType := range injectTypes {
		code.WriteString(fmt.Sprintf("\t// 注入 %s\n", injectType))
		code.WriteString(fmt.Sprintf("\t// TODO: 实现 %s 的依赖注入\n", injectType))
	}
	code.WriteString("}\n")

	return code.String()
}

// generateRouteRegistration 生成路由注册代码
func (ag *AutoGenerator) generateRouteRegistration() error {
	fmt.Println("🛣️  生成路由注册代码...")

	// 获取所有路由注解
	routes := ag.registry.GetRoutes()
	if len(routes) == 0 {
		fmt.Println("ℹ️  没有发现路由注解")
		return nil
	}

	// 生成路由注册代码
	code := ag.generateRouteCode(routes)

	// 写入文件
	outputFile := filepath.Join(ag.outputDir, "route_registration.go")
	if err := os.WriteFile(outputFile, []byte(code), 0644); err != nil {
		return fmt.Errorf("failed to write route registration file: %w", err)
	}

	fmt.Printf("✅ 路由注册代码生成完成: %s\n", outputFile)
	return nil
}

// generateRouteCode 生成路由代码
func (ag *AutoGenerator) generateRouteCode(routes []interface{}) string {
	var code strings.Builder

	code.WriteString("// Code generated by gRain framework. DO NOT EDIT.\n")
	code.WriteString("package generated\n\n")
	code.WriteString("import (\n")
	code.WriteString("\t\"github.com/gin-gonic/gin\"\n")
	code.WriteString(")\n\n")

	code.WriteString("// RegisterAllgRainRoutes 注册所有gRain路由\n")
	code.WriteString("func RegisterAllgRainRoutes(router *gin.Engine, config interface{}, dbManager interface{}) {\n")
	code.WriteString("\t// 注册用户相关路由\n")
	code.WriteString("\tregisterUserRoutes(router, config, dbManager)\n")
	code.WriteString("\t\n")
	code.WriteString("\t// 注册产品相关路由\n")
	code.WriteString("\tregisterProductRoutes(router, config, dbManager)\n")
	code.WriteString("\t\n")
	code.WriteString("\t// 注册认证相关路由\n")
	code.WriteString("\tregisterAuthRoutes(router, config, dbManager)\n")
	code.WriteString("\t\n")
	code.WriteString("\t// 注册统计相关路由\n")
	code.WriteString("\tregisterStatsRoutes(router, config, dbManager)\n")
	code.WriteString("}\n\n")

	// 生成各个模块的路由注册函数
	code.WriteString(ag.generateModuleRoutes("User", "/api/users"))
	code.WriteString(ag.generateModuleRoutes("Product", "/api/products"))
	code.WriteString(ag.generateModuleRoutes("Auth", "/api/auth"))
	code.WriteString(ag.generateModuleRoutes("Stats", "/api/stats"))

	return code.String()
}

// generateModuleRoutes 生成模块路由
func (ag *AutoGenerator) generateModuleRoutes(moduleName, basePath string) string {
	var code strings.Builder

	code.WriteString(fmt.Sprintf("// register%sRoutes 注册%s相关路由\n", moduleName, moduleName))
	code.WriteString(fmt.Sprintf("func register%sRoutes(router *gin.Engine, config interface{}, dbManager interface{}) {\n", moduleName))
	code.WriteString(fmt.Sprintf("\t%sGroup := router.Group(\"%s\")\n", strings.ToLower(moduleName), basePath))
	code.WriteString(fmt.Sprintf("\t{\n"))
	code.WriteString(fmt.Sprintf("\t\t// %s 路由\n", moduleName))
	code.WriteString(fmt.Sprintf("\t\t// TODO: 根据注解自动生成具体路由\n"))
	code.WriteString(fmt.Sprintf("\t}\n"))
	code.WriteString("}\n\n")

	return code.String()
}

// generateSwaggerDocs 生成Swagger文档
func (ag *AutoGenerator) generateSwaggerDocs() error {
	fmt.Println("📚 生成Swagger文档...")

	// 生成Swagger规范
	spec, err := ag.generator.GenerateSwaggerSpec()
	if err != nil {
		return fmt.Errorf("failed to generate swagger spec: %w", err)
	}

	// 保存到文件
	outputFile := filepath.Join(ag.outputDir, "swagger.json")
	if err := ag.swaggerUI.SaveSwaggerSpec(outputFile); err != nil {
		return fmt.Errorf("failed to save swagger spec: %w", err)
	}

	fmt.Printf("✅ Swagger文档生成完成: %s\n", outputFile)
	return nil
}

// generateMainRegistration 生成主注册函数
func (ag *AutoGenerator) generateMainRegistration() error {
	fmt.Println("🎯 生成主注册函数...")

	code := ag.generateMainCode()

	// 写入文件
	outputFile := filepath.Join(ag.outputDir, "main_registration.go")
	if err := os.WriteFile(outputFile, []byte(code), 0644); err != nil {
		return fmt.Errorf("failed to write main registration file: %w", err)
	}

	fmt.Printf("✅ 主注册函数生成完成: %s\n", outputFile)
	return nil
}

// generateMainCode 生成主注册代码
func (ag *AutoGenerator) generateMainCode() string {
	var code strings.Builder

	code.WriteString("// Code generated by gRain framework. DO NOT EDIT.\n")
	code.WriteString("package generated\n\n")
	code.WriteString("import (\n")
	code.WriteString("\t\"github.com/gin-gonic/gin\"\n")
	code.WriteString(")\n\n")

	code.WriteString("// InitializegRainFramework 初始化gRain框架\n")
	code.WriteString("func InitializegRainFramework(router *gin.Engine, config interface{}, dbManager interface{}) {\n")
	code.WriteString("\t// 1. 注册所有路由\n")
	code.WriteString("\tRegisterAllgRainRoutes(router, config, dbManager)\n")
	code.WriteString("\t\n")
	code.WriteString("\t// 2. 注册Swagger文档\n")
	code.WriteString("\tRegisterSwaggerRoutes(router)\n")
	code.WriteString("\t\n")
	code.WriteString("\t// 3. 注册中间件\n")
	code.WriteString("\tRegisterMiddleware(router)\n")
	code.WriteString("}\n\n")

	code.WriteString("// RegisterSwaggerRoutes 注册Swagger路由\n")
	code.WriteString("func RegisterSwaggerRoutes(router *gin.Engine) {\n")
	code.WriteString("\t// Swagger UI\n")
	code.WriteString("\trouter.GET(\"/swagger\", func(c *gin.Context) {\n")
	code.WriteString("\t\tc.Redirect(301, \"/swagger/index.html\")\n")
	code.WriteString("\t})\n")
	code.WriteString("\t\n")
	code.WriteString("\t// Swagger JSON\n")
	code.WriteString("\trouter.GET(\"/swagger/swagger.json\", func(c *gin.Context) {\n")
	code.WriteString("\t\tc.File(\"./generated/swagger.json\")\n")
	code.WriteString("\t})\n")
	code.WriteString("}\n\n")

	code.WriteString("// RegisterMiddleware 注册中间件\n")
	code.WriteString("func RegisterMiddleware(router *gin.Engine) {\n")
	code.WriteString("\t// 添加gRain框架中间件\n")
	code.WriteString("\t// TODO: 根据注解自动注册中间件\n")
	code.WriteString("}\n")

	return code.String()
}

// GetSwaggerServer 获取Swagger服务器
func (ag *AutoGenerator) GetSwaggerServer() *processor.SwaggerServer {
	return ag.swaggerUI
}

// GetOutputDir 获取输出目录
func (ag *AutoGenerator) GetOutputDir() string {
	return ag.outputDir
}
