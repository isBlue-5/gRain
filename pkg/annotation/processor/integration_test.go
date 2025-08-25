package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grain-framework/grain/pkg/annotation/registry"
	"github.com/grain-framework/grain/pkg/annotation/types"
	"github.com/stretchr/testify/suite"
)

// TestFullCodeGeneration 测试完整的代码生成流程
func TestFullCodeGeneration(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "integration_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试用的Go源文件
	testSourceDir := filepath.Join(tempDir, "test_source")
	if err := os.MkdirAll(testSourceDir, 0755); err != nil {
		t.Fatalf("创建测试源目录失败: %v", err)
	}

	// 创建包含注解的测试文件
	testFileContent := `package main

import "fmt"

// frame:controller
type UserController struct{}

// frame:route
// frame:auth
// frame:ratelimit
func (c *UserController) GetUsers() {
	fmt.Println("Get users")
}

// frame:transaction
func (c *UserController) CreateUser() {
	fmt.Println("Create user")
}

// frame:log
func (c *UserController) UpdateUser() {
	fmt.Println("Update user")
}
`
	testFilePath := filepath.Join(testSourceDir, "user_controller.go")
	if err := os.WriteFile(testFilePath, []byte(testFileContent), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建注解解析器和注册表
	parser := NewAnnotationParser("frame:")
	reg := registry.NewRegistry()

	// 创建代码生成器
	generator := NewGenerator(parser, reg, tempDir, "frame:")

	// 测试生成器创建
	if generator == nil {
		t.Fatal("生成器创建失败")
	}

	// 执行代码生成
	err = generator.Generate([]string{testFilePath})
	if err != nil {
		t.Fatalf("代码生成失败: %v", err)
	}

	// 验证生成的代码文件
	expectedFiles := []string{
		"route",
		"authz",
		"ratelimit",
		"transaction",
		"log",
	}

	// 列出生成目录中的所有文件
	t.Logf("生成目录: %s", tempDir)
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(tempDir, path)
			t.Logf("生成的文件: %s", relPath)
		}
		return nil
	})
	if err != nil {
		t.Errorf("遍历生成目录失败: %v", err)
	}

	for _, expectedDir := range expectedFiles {
		dirPath := filepath.Join(tempDir, expectedDir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("期望的生成目录不存在: %s", expectedDir)
		} else {
			// 检查目录中是否有生成的文件
			files, err := os.ReadDir(dirPath)
			if err != nil {
				t.Errorf("读取目录 %s 失败: %v", expectedDir, err)
			} else if len(files) == 0 {
				t.Errorf("目录 %s 中没有生成的文件", expectedDir)
			} else {
				t.Logf("成功生成目录 %s，包含 %d 个文件", expectedDir, len(files))
			}
		}
	}
}

// TestAnnotationParsing 测试注解解析功能
func TestAnnotationParsing(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "annotation_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试用的Go源文件
	testSourceDir := filepath.Join(tempDir, "test_source")
	if err := os.MkdirAll(testSourceDir, 0755); err != nil {
		t.Fatalf("创建测试源目录失败: %v", err)
	}

	// 创建包含各种注解的测试文件
	testFileContent := `package main

import "fmt"

// frame:entity(name="User")
type User struct {
	ID   int    ` + "`" + `db:"id" json:"id"` + "`" + `
	Name string ` + "`" + `db:"name" json:"name"` + "`" + `
}

// frame:service(name="UserService")
type UserService struct{}

// frame:inject
func NewUserService() *UserService {
	return &UserService{}
}

// frame:cache(ttl="5m", key="user_{id}")
func (s *UserService) GetUser(id int) *User {
	return &User{ID: id, Name: "Test User"}
}

// frame:query(sql="SELECT * FROM users WHERE id = ?")
func (s *UserService) FindUser(id int) *User {
	return &User{ID: id, Name: "Test User"}
}
`
	testFilePath := filepath.Join(testSourceDir, "user_service.go")
	if err := os.WriteFile(testFilePath, []byte(testFileContent), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建注解解析器
	parser := NewAnnotationParser("frame:")

	// 解析注解
	annotations, err := parser.ParsePackage(testSourceDir)
	if err != nil {
		t.Fatalf("解析注解失败: %v", err)
	}

	// 验证解析到的注解数量
	expectedAnnotationCount := 5 // entity, service, inject, cache, query
	if len(annotations) != expectedAnnotationCount {
		t.Errorf("期望解析到 %d 个注解，实际解析到 %d 个", expectedAnnotationCount, len(annotations))
	}

	t.Logf("成功解析到 %d 个注解", len(annotations))
}

// TestProcessorIntegration 测试处理器集成
func TestProcessorIntegration(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "processor_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试用的Go源文件
	testSourceDir := filepath.Join(tempDir, "test_source")
	if err := os.MkdirAll(testSourceDir, 0755); err != nil {
		t.Fatalf("创建测试源目录失败: %v", err)
	}

	// 创建包含限流和权限注解的测试文件
	testFileContent := `package main

import "fmt"

// @frame:controller(name="ProductController")
type ProductController struct{}

// @frame:route(method="POST", path="/products")
// @frame:auth(roles={"ADMIN"})
// @frame:rateLimit(limit=50, period="1m", algorithm="token_bucket")
func (c *ProductController) CreateProduct() {
	fmt.Println("Create product")
}

// @frame:route(method="GET", path="/products/{id}")
// @frame:rateLimit(limit=200, period="1m")
func (c *ProductController) GetProduct() {
	fmt.Println("Get product")
}
`
	testFilePath := filepath.Join(testSourceDir, "product_controller.go")
	if err := os.WriteFile(testFilePath, []byte(testFileContent), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建注解解析器和注册表
	parser := NewAnnotationParser("@frame:")
	reg := registry.NewRegistry()

	// 先解析注解
	annotations, err := parser.ParseFile(testFilePath)
	if err != nil {
		t.Fatalf("解析注解失败: %v", err)
	}

	// 将解析的注解注册到注册表
	for _, anno := range annotations {
		reg.Register(anno)
	}

	// 创建各个处理器
	rateLimitProcessor := NewRateLimitProcessor(reg, parser, filepath.Join(tempDir, "ratelimit"))
	authzProcessor := NewAuthzProcessor(reg, parser, filepath.Join(tempDir, "authz"))
	routeProcessor := NewRouteProcessor(reg, "@frame:", filepath.Join(tempDir, "route"))

	// 测试处理器创建
	if rateLimitProcessor == nil {
		t.Fatal("限流处理器创建失败")
	}
	if authzProcessor == nil {
		t.Fatal("权限控制处理器创建失败")
	}
	if routeProcessor == nil {
		t.Fatal("路由处理器创建失败")
	}

	// 执行处理
	err = rateLimitProcessor.ProcessRateLimit([]string{testFilePath})
	if err != nil {
		t.Errorf("限流处理失败: %v", err)
	}

	err = authzProcessor.ProcessAuthz([]string{testFilePath})
	if err != nil {
		t.Errorf("权限控制处理失败: %v", err)
	}

	err = routeProcessor.ProcessRoute([]string{testFilePath})
	if err != nil {
		t.Errorf("路由处理失败: %v", err)
	}

	// 验证生成的目录
	expectedDirs := []string{
		"ratelimit",
		"authz",
		"route",
	}

	for _, expectedDir := range expectedDirs {
		dirPath := filepath.Join(tempDir, expectedDir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("期望的生成目录不存在: %s", expectedDir)
		} else {
			// 检查目录中是否有生成的文件
			files, err := os.ReadDir(dirPath)
			if err != nil {
				t.Errorf("读取目录 %s 失败: %v", expectedDir, err)
			} else if len(files) == 0 {
				t.Errorf("目录 %s 中没有生成的文件", expectedDir)
			} else {
				t.Logf("成功生成目录 %s，包含 %d 个文件", expectedDir, len(files))
			}
		}
	}
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "error_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建注解解析器
	parser := NewAnnotationParser("frame:")

	// 测试解析不存在的文件
	_, err = parser.ParseFile("/nonexistent/file.go")
	if err == nil {
		t.Error("解析不存在的文件应该返回错误")
	}

	// 测试解析不存在的目录
	_, err = parser.ParsePackage("/nonexistent/directory")
	if err == nil {
		t.Error("解析不存在的目录应该返回错误")
	}

	// 测试解析空路径
	_, err = parser.ParsePackage("")
	if err != nil {
		t.Errorf("解析空路径不应该返回错误: %v", err)
	}
}

// TestPerformance 测试性能
func TestPerformance(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "performance_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建大量测试文件
	testSourceDir := filepath.Join(tempDir, "test_source")
	if err := os.MkdirAll(testSourceDir, 0755); err != nil {
		t.Fatalf("创建测试源目录失败: %v", err)
	}

	// 创建多个测试文件
	for i := 0; i < 10; i++ {
		testFileContent := fmt.Sprintf(`package main

import "fmt"

// @frame:controller(name="Controller%d")
type Controller%d struct{}

// @frame:route(method="GET", path="/api%d")
// @frame:auth(roles=["USER"])
func (c *Controller%d) Handle%d() {
	fmt.Println("Handle %d")
}
`, i, i, i, i, i, i)

		testFilePath := filepath.Join(testSourceDir, fmt.Sprintf("controller_%d.go", i))
		if err := os.WriteFile(testFilePath, []byte(testFileContent), 0644); err != nil {
			t.Fatalf("创建测试文件 %d 失败: %v", i, err)
		}
	}

	// 创建注解解析器和注册表
	parser := NewAnnotationParser("@frame:")
	reg := registry.NewRegistry()

	// 创建代码生成器
	generator := NewGenerator(parser, reg, tempDir, "@frame:")

	// 执行代码生成并测量时间
	start := time.Now()

	// 收集所有文件路径
	var filePaths []string
	err = filepath.Walk(testSourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			filePaths = append(filePaths, path)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("收集文件路径失败: %v", err)
	}

	err = generator.Generate(filePaths)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("代码生成失败: %v", err)
	}

	t.Logf("处理 10 个文件耗时: %v", duration)

	// 验证生成的文件数量
	expectedFileCount := 50 // 10个控制器 * 5种处理器
	actualFileCount := 0
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			actualFileCount++
		}
		return nil
	})

	if err != nil {
		t.Errorf("统计生成文件失败: %v", err)
	}

	t.Logf("期望生成 %d 个文件，实际生成 %d 个文件", expectedFileCount, actualFileCount)
}

// IntegrationTestSuite 集成测试套件
type IntegrationTestSuite struct {
	suite.Suite
	tempDir   string
	registry  registry.Registry
	processor *FixedRouteProcessor
	validator *CodeQualityValidator
}

// SetupSuite 测试套件设置
func (suite *IntegrationTestSuite) SetupSuite() {
	suite.tempDir = suite.T().TempDir()
	suite.registry = registry.NewRegistry()
	suite.processor = NewFixedRouteProcessor(suite.registry, "@frame:", suite.tempDir)
	suite.validator = NewCodeQualityValidator(suite.tempDir)
}

// TestCodeGenerationWorkflow 测试完整的代码生成工作流
func (suite *IntegrationTestSuite) TestCodeGenerationWorkflow() {
	// 1. 创建测试注解
	suite.createTestAnnotations()

	// 2. 生成代码
	err := suite.processor.ProcessRoutes()
	suite.NoError(err)

	// 3. 验证生成的代码质量
	results, err := suite.validator.ValidateGeneratedCode()
	suite.NoError(err)

	// 4. 检查验证结果
	suite.assertValidationResults(results)

	// 5. 验证文件结构
	suite.assertFileStructure()
}

// TestCodeQualityStandards 测试代码质量标准
func (suite *IntegrationTestSuite) TestCodeQualityStandards() {
	// 创建高质量的测试代码
	suite.createHighQualityTestCode()

	// 验证代码质量
	results, err := suite.validator.ValidateGeneratedCode()
	suite.NoError(err)

	// 所有文件都应该通过验证
	for _, result := range results {
		suite.True(result.Passed, "File %s should pass validation", result.File)
		suite.Empty(result.Errors, "File %s should have no errors", result.File)
	}
}

// TestErrorHandling 测试错误处理
func (suite *IntegrationTestSuite) TestErrorHandling() {
	// 创建有问题的代码
	suite.createProblematicCode()

	// 验证错误检测
	results, err := suite.validator.ValidateGeneratedCode()
	suite.NoError(err)

	// 应该检测到错误
	hasErrors := false
	for _, result := range results {
		if len(result.Errors) > 0 {
			hasErrors = true
			break
		}
	}
	suite.True(hasErrors, "Should detect code quality issues")
}

// createTestAnnotations 创建测试注解
func (suite *IntegrationTestSuite) createTestAnnotations() {
	// 创建控制器注解
	controllerAnno := &types.CommentAnnotation{
		Type:       types.ControllerType,
		TargetType: types.TypeTarget,
		TargetName: "UserController",
		Position:   types.Position{Filename: "user_controller.go"},
	}

	suite.registry.Register(controllerAnno)

	// 创建路由注解
	routeAnno := &types.CommentAnnotation{
		Type:       types.RouteType,
		TargetType: types.MethodTarget,
		TargetName: "CreateUser",
		Position:   types.Position{Filename: "user_controller.go"},
	}

	suite.registry.Register(routeAnno)
}

// createHighQualityTestCode 创建高质量的测试代码
func (suite *IntegrationTestSuite) createHighQualityTestCode() {
	// 创建高质量的Go文件
	highQualityCode := `package test

import "fmt"

// User represents a user entity
type User struct {
	ID   int    ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

// NewUser creates a new user
func NewUser(id int, name string) *User {
	return &User{
		ID:   id,
		Name: name,
	}
}

// String returns string representation
func (u *User) String() string {
	return fmt.Sprintf("User{ID: %d, Name: %s}", u.ID, u.Name)
}
`

	filePath := filepath.Join(suite.tempDir, "high_quality.go")
	err := os.WriteFile(filePath, []byte(highQualityCode), 0644)
	suite.NoError(err)
}

// createProblematicCode 创建有问题的代码
func (suite *IntegrationTestSuite) createProblematicCode() {
	// 创建有语法错误的Go文件
	problematicCode := `package test

import "fmt"

func ProblematicFunction() {
	fmt.Println("This function has issues")
	undefinedVariable // 未定义的变量
}
`

	filePath := filepath.Join(suite.tempDir, "problematic.go")
	err := os.WriteFile(filePath, []byte(problematicCode), 0644)
	suite.NoError(err)
}

// assertValidationResults 断言验证结果
func (suite *IntegrationTestSuite) assertValidationResults(results []ValidationResult) {
	suite.NotEmpty(results, "Should have validation results")

	// 检查是否有通过验证的文件
	hasPassed := false
	for _, result := range results {
		if result.Passed {
			hasPassed = true
			break
		}
	}
	suite.True(hasPassed, "Should have at least one file passing validation")
}

// assertFileStructure 断言文件结构
func (suite *IntegrationTestSuite) assertFileStructure() {
	// 检查是否生成了通用接口文件
	commonFile := filepath.Join(suite.tempDir, "common", "interfaces.go")
	suite.FileExists(commonFile, "Common interfaces file should exist")

	// 检查文件内容
	content, err := os.ReadFile(commonFile)
	suite.NoError(err)
	suite.Contains(string(content), "type Controller interface")
	suite.Contains(string(content), "type Validator interface")
}

// TestMain 测试主函数
func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
