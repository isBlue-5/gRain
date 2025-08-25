package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodeQualityValidator_ValidateGeneratedCode(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()

	// 创建测试Go文件
	testFiles := map[string]string{
		"valid.go": `package test

import "fmt"

func Hello() {
	fmt.Println("Hello, World!")
}`,
		"invalid.go": `package test

import "fmt"

func Hello() {
	fmt.Println("Hello, World!")
	undefinedVariable // 未定义的变量
}`,
		"unused_import.go": `package test

import (
	"fmt"
	"os" // 未使用的导入
)

func Hello() {
	fmt.Println("Hello, World!")
}`,
	}

	// 写入测试文件
	for fileName, content := range testFiles {
		filePath := filepath.Join(tempDir, fileName)
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.NoError(t, err)
	}

	// 创建验证器
	validator := NewCodeQualityValidator(tempDir)

	// 验证代码
	results, err := validator.ValidateGeneratedCode()
	assert.NoError(t, err)
	assert.Len(t, results, 3)

	// 验证结果
	validFile := findResultByFile(results, "valid.go")
	assert.NotNil(t, validFile)
	assert.True(t, validFile.Passed)
	assert.Empty(t, validFile.Errors)

	invalidFile := findResultByFile(results, "invalid.go")
	assert.NotNil(t, invalidFile)
	assert.False(t, invalidFile.Passed)
	assert.NotEmpty(t, invalidFile.Errors)

	unusedImportFile := findResultByFile(results, "unused_import.go")
	assert.NotNil(t, unusedImportFile)
	assert.True(t, unusedImportFile.Passed) // 导入检查只是警告
	assert.NotEmpty(t, unusedImportFile.Warnings)
}

func TestCodeQualityValidator_GenerateQualityReport(t *testing.T) {
	// 创建验证器
	validator := NewCodeQualityValidator("")

	// 创建测试结果
	results := []ValidationResult{
		{
			File:     "file1.go",
			Errors:   []string{"syntax error"},
			Warnings: []string{"unused import"},
			Passed:   false,
		},
		{
			File:     "file2.go",
			Errors:   []string{},
			Warnings: []string{},
			Passed:   true,
		},
	}

	// 生成报告
	report := validator.GenerateQualityReport(results)

	// 验证报告内容
	assert.Contains(t, report, "总文件数: 2")
	assert.Contains(t, report, "通过验证: 1")
	assert.Contains(t, report, "失败文件: 1")
	assert.Contains(t, report, "总错误数: 1")
	assert.Contains(t, report, "总警告数: 1")
	assert.Contains(t, report, "file1.go")
	assert.Contains(t, report, "file2.go")
}

// 辅助函数：根据文件名查找结果
func findResultByFile(results []ValidationResult, fileName string) *ValidationResult {
	for _, result := range results {
		if filepath.Base(result.File) == fileName {
			return &result
		}
	}
	return nil
}
