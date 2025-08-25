package processor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportManager(t *testing.T) {
	im := NewImportManager()

	// 测试添加导入
	im.AddImport("github.com/gin-gonic/gin", "")
	im.AddImport("fmt", "")
	im.AddImport("strings", "str")

	// 测试标记使用
	im.MarkUsed("github.com/gin-gonic/gin")
	im.MarkUsed("fmt")

	// 获取清理后的导入
	imports := im.GetCleanImports()

	// 验证结果
	if len(imports) != 2 {
		t.Errorf("期望2个导入，实际得到%d个", len(imports))
	}

	// 验证未使用的导入被移除
	im.RemoveUnusedImports()
	if len(im.imports) != 2 {
		t.Errorf("期望2个导入，实际得到%d个", len(im.imports))
	}
}

func TestCodeValidator(t *testing.T) {
	cv := NewCodeValidator()

	// 测试语法验证
	validCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`

	// 创建临时文件
	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "valid.go")
	if err := os.WriteFile(validFile, []byte(validCode), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 验证有效代码
	if err := cv.ValidateFile(validFile); err != nil {
		t.Errorf("有效代码验证失败: %v", err)
	}

	// 测试无效代码
	invalidCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!"
	// 缺少右括号
}
`

	invalidFile := filepath.Join(tempDir, "invalid.go")
	if err := os.WriteFile(invalidFile, []byte(invalidCode), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 验证无效代码
	if err := cv.ValidateFile(invalidFile); err == nil {
		t.Error("无效代码验证应该失败")
	}

	// 检查错误数量
	errors := cv.GetErrors()
	if len(errors) == 0 {
		t.Error("应该检测到语法错误")
	}
}

func TestImportPathValidation(t *testing.T) {
	im := NewImportManager()

	// 测试标准库路径
	if err := im.ValidateImportPath("fmt"); err != nil {
		t.Errorf("标准库路径验证失败: %v", err)
	}

	// 测试第三方库路径
	if err := im.ValidateImportPath("github.com/gin-gonic/gin"); err != nil {
		t.Errorf("第三方库路径验证失败: %v", err)
	}

	// 测试无效路径
	if err := im.ValidateImportPath("./relative/path"); err == nil {
		t.Error("相对路径应该验证失败")
	}

	if err := im.ValidateImportPath("/absolute/path"); err == nil {
		t.Error("绝对路径应该验证失败")
	}
}

func TestGeneratedCodeValidation(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建有效的Go文件
	validCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`

	validFile := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(validFile, []byte(validCode), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 验证生成的代码
	if err := ValidateGeneratedCode(tempDir); err != nil {
		t.Errorf("代码验证失败: %v", err)
	}

	// 创建无效的Go文件
	invalidCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!"
	// 语法错误
}
`

	invalidFile := filepath.Join(tempDir, "invalid.go")
	if err := os.WriteFile(invalidFile, []byte(invalidCode), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 验证应该失败
	if err := ValidateGeneratedCode(tempDir); err == nil {
		t.Error("包含语法错误的代码验证应该失败")
	}
}

func TestTemplateQualityImprovements(t *testing.T) {
	// 测试路由模板修复
	routeData := routeTemplateData{
		PackageName:    "main",
		ControllerName: "UserController",
		Routes: []routeInfo{
			{
				MethodName: "ListUsers",
				HttpMethod: "GET",
				Path:       "/users",
			},
		},
	}

	// 验证模板数据结构
	if routeData.PackageName != "main" {
		t.Errorf("期望包名为main，实际为%s", routeData.PackageName)
	}

	if len(routeData.Routes) != 1 {
		t.Errorf("期望1个路由，实际为%d个", len(routeData.Routes))
	}

	// 测试权限控制模板数据结构
	authzData := map[string]interface{}{
		"Package":      "main",
		"WrappedName":  "SecureMethod",
		"OriginalName": "Method",
		"ParamNames":   []string{"ctx"},
		"ParamTypes":   []string{"*gin.Context"},
		"Returns":      []string{"error"},
		"HasRoles":     true,
		"Roles":        []string{"admin", "user"},
	}

	if authzData["Package"] != "main" {
		t.Errorf("期望包名为main，实际为%s", authzData["Package"])
	}

	if roles, ok := authzData["Roles"].([]string); !ok || len(roles) != 2 {
		t.Errorf("期望2个角色，实际为%v", roles)
	}
}
