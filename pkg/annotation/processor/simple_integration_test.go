package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/isBlue-5/grain/pkg/annotation/registry"
	"github.com/isBlue-5/grain/pkg/annotation/types"
	"github.com/stretchr/testify/assert"
)

// TestSimpleCodeGenerationWorkflow 测试简化的代码生成工作流
func TestSimpleCodeGenerationWorkflow(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建注册中心
	reg := registry.NewRegistry()

	// 创建路由处理器
	processor := NewFixedRouteProcessor(reg, "@frame:", tempDir)

	// 创建测试注解
	controllerAnno := &types.CommentAnnotation{
		Type:       types.ControllerType,
		TargetType: types.TypeTarget,
		TargetName: "UserController",
		Position:   types.Position{Filename: "user_controller.go"},
	}

	reg.Register(controllerAnno)

	// 生成代码
	err := processor.ProcessRoutes()
	assert.NoError(t, err)

	// 验证生成了通用接口文件
	commonFile := filepath.Join(tempDir, "common", "interfaces.go")
	assert.FileExists(t, commonFile)

	// 检查文件内容
	content, err := os.ReadFile(commonFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "type Controller interface")
	assert.Contains(t, string(content), "type Validator interface")
}

// TestCodeQualityValidation 测试代码质量验证
func TestCodeQualityValidation(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建高质量的测试代码
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

	filePath := filepath.Join(tempDir, "high_quality.go")
	err := os.WriteFile(filePath, []byte(highQualityCode), 0644)
	assert.NoError(t, err)

	// 创建验证器
	validator := NewCodeQualityValidator(tempDir)

	// 验证代码质量
	results, err := validator.ValidateGeneratedCode()
	assert.NoError(t, err)

	// 应该有一个文件通过验证
	assert.Len(t, results, 1)
	assert.True(t, results[0].Passed)
	assert.Empty(t, results[0].Errors)
}

// TestErrorDetection 测试错误检测
func TestErrorDetection(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建有问题的代码
	problematicCode := `package test

import "fmt"

func ProblematicFunction() {
	fmt.Println("This function has issues")
	undefinedVariable // 未定义的变量
}
`

	filePath := filepath.Join(tempDir, "problematic.go")
	err := os.WriteFile(filePath, []byte(problematicCode), 0644)
	assert.NoError(t, err)

	// 创建验证器
	validator := NewCodeQualityValidator(tempDir)

	// 验证代码质量
	results, err := validator.ValidateGeneratedCode()
	assert.NoError(t, err)

	// 应该检测到错误
	assert.Len(t, results, 1)
	assert.False(t, results[0].Passed)
	assert.NotEmpty(t, results[0].Errors)
}

// TestPerformanceValidation 测试性能验证
func TestPerformanceValidation(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建多个测试文件
	for i := 0; i < 10; i++ {
		fileName := filepath.Join(tempDir, fmt.Sprintf("test_%d.go", i))
		content := fmt.Sprintf(`package test%d

import "fmt"

func Hello%d() {
	fmt.Println("Hello from test %d")
}
`, i, i, i)

		err := os.WriteFile(fileName, []byte(content), 0644)
		assert.NoError(t, err)
	}

	// 创建验证器
	validator := NewCodeQualityValidator(tempDir)

	// 验证代码质量
	results, err := validator.ValidateGeneratedCode()
	assert.NoError(t, err)

	// 应该验证了所有文件
	assert.Len(t, results, 10)

	// 所有文件都应该通过验证
	for _, result := range results {
		assert.True(t, result.Passed)
	}
}
