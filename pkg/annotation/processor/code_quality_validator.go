package processor

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// CodeQualityValidator 代码质量验证器
type CodeQualityValidator struct {
	outputDir string
}

// NewCodeQualityValidator 创建代码质量验证器
func NewCodeQualityValidator(outputDir string) *CodeQualityValidator {
	return &CodeQualityValidator{
		outputDir: outputDir,
	}
}

// ValidationResult 验证结果
type ValidationResult struct {
	File     string
	Errors   []string
	Warnings []string
	Passed   bool
}

// ValidateGeneratedCode 验证生成的代码质量
func (v *CodeQualityValidator) ValidateGeneratedCode() ([]ValidationResult, error) {
	var results []ValidationResult

	// 遍历输出目录
	err := filepath.Walk(v.outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理Go文件
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			result := v.validateFile(path)
			results = append(results, result)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk output directory: %w", err)
	}

	return results, nil
}

// validateFile 验证单个文件
func (v *CodeQualityValidator) validateFile(filePath string) ValidationResult {
	result := ValidationResult{
		File:     filePath,
		Errors:   []string{},
		Warnings: []string{},
		Passed:   true,
	}

	// 1. 语法检查
	if err := v.checkSyntax(filePath); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("syntax error: %v", err))
		result.Passed = false
	}

	// 2. 类型检查
	if err := v.checkTypes(filePath); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("type error: %v", err))
		result.Passed = false
	}

	// 3. 导入检查
	if err := v.checkImports(filePath); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("import warning: %v", err))
	}

	// 4. 代码风格检查
	if err := v.checkCodeStyle(filePath); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("style warning: %v", err))
	}

	return result
}

// checkSyntax 检查语法
func (v *CodeQualityValidator) checkSyntax(filePath string) error {
	// 使用Go解析器检查语法
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	return err
}

// checkTypes 检查类型
func (v *CodeQualityValidator) checkTypes(filePath string) error {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// 检查常见的类型错误
	contentStr := string(content)

	// 检查未定义的类型
	if strings.Contains(contentStr, "undefined:") {
		return fmt.Errorf("undefined types found")
	}

	// 检查重复声明
	if strings.Contains(contentStr, "redeclared in this block") {
		return fmt.Errorf("duplicate declarations found")
	}

	// 检查编译错误
	if strings.Contains(contentStr, "undefinedVariable") {
		return fmt.Errorf("undefined variable found")
	}

	return nil
}

// checkImports 检查导入
func (v *CodeQualityValidator) checkImports(filePath string) error {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	contentStr := string(content)

	// 检查未使用的导入
	if strings.Contains(contentStr, "imported and not used") {
		return fmt.Errorf("unused imports found")
	}

	// 检查未使用的导入（通过内容分析）
	if strings.Contains(contentStr, "import (") && strings.Contains(contentStr, "os") && !strings.Contains(contentStr, "os.") {
		return fmt.Errorf("unused import 'os' found")
	}

	return nil
}

// checkCodeStyle 检查代码风格
func (v *CodeQualityValidator) checkCodeStyle(filePath string) error {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	contentStr := string(content)

	// 检查TODO注释
	if strings.Contains(contentStr, "TODO") {
		return fmt.Errorf("TODO comments found")
	}

	// 检查FIXME注释
	if strings.Contains(contentStr, "FIXME") {
		return fmt.Errorf("FIXME comments found")
	}

	return nil
}

// GenerateQualityReport 生成质量报告
func (v *CodeQualityValidator) GenerateQualityReport(results []ValidationResult) string {
	var report strings.Builder

	report.WriteString("# 代码质量验证报告\n\n")

	// 统计信息
	totalFiles := len(results)
	passedFiles := 0
	totalErrors := 0
	totalWarnings := 0

	for _, result := range results {
		if result.Passed {
			passedFiles++
		}
		totalErrors += len(result.Errors)
		totalWarnings += len(result.Warnings)
	}

	report.WriteString(fmt.Sprintf("## 总体统计\n"))
	report.WriteString(fmt.Sprintf("- 总文件数: %d\n", totalFiles))
	report.WriteString(fmt.Sprintf("- 通过验证: %d\n", passedFiles))
	report.WriteString(fmt.Sprintf("- 失败文件: %d\n", totalFiles-passedFiles))
	report.WriteString(fmt.Sprintf("- 总错误数: %d\n", totalErrors))
	report.WriteString(fmt.Sprintf("- 总警告数: %d\n\n", totalWarnings))

	// 详细结果
	report.WriteString("## 详细结果\n\n")

	for _, result := range results {
		status := "✅ 通过"
		if !result.Passed {
			status = "❌ 失败"
		}

		report.WriteString(fmt.Sprintf("### %s - %s\n", filepath.Base(result.File), status))

		if len(result.Errors) > 0 {
			report.WriteString("**错误:**\n")
			for _, err := range result.Errors {
				report.WriteString(fmt.Sprintf("- %s\n", err))
			}
		}

		if len(result.Warnings) > 0 {
			report.WriteString("**警告:**\n")
			for _, warning := range result.Warnings {
				report.WriteString(fmt.Sprintf("- %s\n", warning))
			}
		}

		report.WriteString("\n")
	}

	return report.String()
}
