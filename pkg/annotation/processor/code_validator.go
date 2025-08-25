package processor

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CodeValidator 代码质量验证器
type CodeValidator struct {
	// 验证规则
	rules []ValidationRule

	// 错误收集
	errors []ValidationError

	// 警告收集
	warnings []ValidationWarning
}

// ValidationRule 验证规则接口
type ValidationRule interface {
	Validate(code string, filePath string) []ValidationError
}

// ValidationError 验证错误
type ValidationError struct {
	File     string
	Line     int
	Column   int
	Message  string
	Severity ErrorSeverity
}

// ValidationWarning 验证警告
type ValidationWarning struct {
	File    string
	Line    int
	Column  int
	Message string
}

// ErrorSeverity 错误严重程度
type ErrorSeverity int

const (
	SeverityError ErrorSeverity = iota
	SeverityWarning
	SeverityInfo
)

// NewCodeValidator 创建新的代码验证器
func NewCodeValidator() *CodeValidator {
	cv := &CodeValidator{
		rules:    make([]ValidationRule, 0),
		errors:   make([]ValidationError, 0),
		warnings: make([]ValidationWarning, 0),
	}

	// 添加默认验证规则
	cv.addDefaultRules()

	return cv
}

// addDefaultRules 添加默认验证规则
func (cv *CodeValidator) addDefaultRules() {
	cv.rules = append(cv.rules,
		&SyntaxRule{},
		&ImportRule{},
		&TypeRule{},
		&NamingRule{},
		&StructureRule{},
	)
}

// ValidateFile 验证单个文件
func (cv *CodeValidator) ValidateFile(filePath string) error {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// 执行所有验证规则
	for _, rule := range cv.rules {
		errors := rule.Validate(string(content), filePath)
		cv.errors = append(cv.errors, errors...)
	}

	// 检查是否有严重错误
	for _, err := range cv.errors {
		if err.Severity == SeverityError {
			return fmt.Errorf("文件 %s 验证失败: %s", filePath, err.Message)
		}
	}

	return nil
}

// ValidateDirectory 验证整个目录
func (cv *CodeValidator) ValidateDirectory(dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理Go文件
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			if err := cv.ValidateFile(path); err != nil {
				return err
			}
		}

		return nil
	})
}

// GetErrors 获取所有错误
func (cv *CodeValidator) GetErrors() []ValidationError {
	return cv.errors
}

// GetWarnings 获取所有警告
func (cv *CodeValidator) GetWarnings() []ValidationWarning {
	return cv.warnings
}

// HasErrors 检查是否有错误
func (cv *CodeValidator) HasErrors() bool {
	for _, err := range cv.errors {
		if err.Severity == SeverityError {
			return true
		}
	}
	return false
}

// PrintReport 打印验证报告
func (cv *CodeValidator) PrintReport() {
	if len(cv.errors) == 0 && len(cv.warnings) == 0 {
		fmt.Println("✅ 代码验证通过，未发现任何问题")
		return
	}

	fmt.Println("\n--- 代码验证报告 ---")

	// 打印错误
	if len(cv.errors) > 0 {
		fmt.Printf("\n❌ 发现 %d 个错误:\n", len(cv.errors))
		for _, err := range cv.errors {
			severity := "ERROR"
			if err.Severity == SeverityWarning {
				severity = "WARNING"
			} else if err.Severity == SeverityInfo {
				severity = "INFO"
			}
			fmt.Printf("  %s:%d:%d [%s] %s\n", err.File, err.Line, err.Column, severity, err.Message)
		}
	}

	// 打印警告
	if len(cv.warnings) > 0 {
		fmt.Printf("\n⚠️  发现 %d 个警告:\n", len(cv.warnings))
		for _, warning := range cv.warnings {
			fmt.Printf("  %s:%d:%d [WARNING] %s\n", warning.File, warning.Line, warning.Column, warning.Message)
		}
	}

	fmt.Println("------------------------")
}

// SyntaxRule 语法验证规则
type SyntaxRule struct{}

func (r *SyntaxRule) Validate(code string, filePath string) []ValidationError {
	var errors []ValidationError

	// 使用Go解析器检查语法
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, filePath, []byte(code), parser.ParseComments)
	if err != nil {
		// 尝试解析错误信息获取行号和列号
		line, column := extractLineColumn(err.Error())
		errors = append(errors, ValidationError{
			File:     filePath,
			Line:     line,
			Column:   column,
			Message:  fmt.Sprintf("语法错误: %s", err.Error()),
			Severity: SeverityError,
		})
	}

	return errors
}

// ImportRule 导入验证规则
type ImportRule struct{}

func (r *ImportRule) Validate(code string, filePath string) []ValidationError {
	var errors []ValidationError

	// 检查未使用的导入
	if strings.Contains(code, "imported and not used") {
		errors = append(errors, ValidationError{
			File:     filePath,
			Line:     0,
			Column:   0,
			Message:  "存在未使用的导入",
			Severity: SeverityWarning,
		})
	}

	// 检查循环导入
	if strings.Contains(code, "import cycle") {
		errors = append(errors, ValidationError{
			File:     filePath,
			Line:     0,
			Column:   0,
			Message:  "检测到循环导入",
			Severity: SeverityError,
		})
	}

	return errors
}

// TypeRule 类型验证规则
type TypeRule struct{}

func (r *TypeRule) Validate(code string, filePath string) []ValidationError {
	var errors []ValidationError

	// 检查未定义的类型
	if strings.Contains(code, "undefined:") {
		errors = append(errors, ValidationError{
			File:     filePath,
			Line:     0,
			Column:   0,
			Message:  "存在未定义的类型",
			Severity: SeverityError,
		})
	}

	// 检查类型不匹配
	if strings.Contains(code, "cannot use") && strings.Contains(code, "as") {
		errors = append(errors, ValidationError{
			File:     filePath,
			Line:     0,
			Column:   0,
			Message:  "存在类型不匹配",
			Severity: SeverityError,
		})
	}

	return errors
}

// NamingRule 命名验证规则
type NamingRule struct{}

func (r *NamingRule) Validate(code string, filePath string) []ValidationError {
	var errors []ValidationError

	// 检查是否遵循Go命名约定
	lines := strings.Split(code, "\n")

	for lineNum, line := range lines {
		lineNum++ // 转换为1-based行号

		// 检查包名命名约定
		if strings.HasPrefix(strings.TrimSpace(line), "package ") {
			packageName := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "package "))
			if !r.isValidPackageName(packageName) {
				errors = append(errors, ValidationError{
					File:     filePath,
					Line:     lineNum,
					Column:   0,
					Message:  fmt.Sprintf("包名 '%s' 不符合Go命名约定，应该使用小写字母", packageName),
					Severity: SeverityWarning,
				})
			}
		}

		// 检查类型声明命名约定
		if strings.Contains(line, "type ") && strings.Contains(line, " struct") {
			typeName := r.extractTypeName(line)
			if typeName != "" && !r.isValidTypeName(typeName) {
				errors = append(errors, ValidationError{
					File:     filePath,
					Line:     lineNum,
					Column:   0,
					Message:  fmt.Sprintf("类型名 '%s' 不符合Go命名约定，应该使用PascalCase", typeName),
					Severity: SeverityWarning,
				})
			}
		}

		// 检查函数声明命名约定
		if strings.Contains(line, "func ") && !strings.Contains(line, "(") {
			funcName := r.extractFunctionName(line)
			if funcName != "" && !r.isValidFunctionName(funcName) {
				errors = append(errors, ValidationError{
					File:     filePath,
					Line:     lineNum,
					Column:   0,
					Message:  fmt.Sprintf("函数名 '%s' 不符合Go命名约定，应该使用PascalCase", funcName),
					Severity: SeverityWarning,
				})
			}
		}

		// 检查变量声明命名约定
		if strings.Contains(line, "var ") || strings.Contains(line, ":=") {
			varName := r.extractVariableName(line)
			if varName != "" && !r.isValidVariableName(varName) {
				errors = append(errors, ValidationError{
					File:     filePath,
					Line:     lineNum,
					Column:   0,
					Message:  fmt.Sprintf("变量名 '%s' 不符合Go命名约定，应该使用camelCase", varName),
					Severity: SeverityWarning,
				})
			}
		}
	}

	return errors
}

// StructureRule 结构验证规则
type StructureRule struct{}

func (r *StructureRule) Validate(code string, filePath string) []ValidationError {
	var errors []ValidationError

	// 检查代码结构问题
	lines := strings.Split(code, "\n")

	for lineNum, line := range lines {
		lineNum++ // 转换为1-based行号

		// 检查导入语句
		if strings.HasPrefix(strings.TrimSpace(line), "import ") {
			if !r.isValidImportStatement(line) {
				errors = append(errors, ValidationError{
					File:     filePath,
					Line:     lineNum,
					Column:   0,
					Message:  "导入语句格式不正确",
					Severity: SeverityWarning,
				})
			}
		}

		// 检查函数声明
		if strings.Contains(line, "func ") {
			if !r.isValidFunctionDeclaration(line) {
				errors = append(errors, ValidationError{
					File:     filePath,
					Line:     lineNum,
					Column:   0,
					Message:  "函数声明格式不正确",
					Severity: SeverityWarning,
				})
			}
		}

		// 检查结构体声明
		if strings.Contains(line, "type ") && strings.Contains(line, " struct") {
			if !r.isValidStructDeclaration(line) {
				errors = append(errors, ValidationError{
					File:     filePath,
					Line:     lineNum,
					Column:   0,
					Message:  "结构体声明格式不正确",
					Severity: SeverityWarning,
				})
			}
		}

		// 检查接口声明
		if strings.Contains(line, "type ") && strings.Contains(line, " interface") {
			if !r.isValidInterfaceDeclaration(line) {
				errors = append(errors, ValidationError{
					File:     filePath,
					Line:     lineNum,
					Column:   0,
					Message:  "接口声明格式不正确",
					Severity: SeverityWarning,
				})
			}
		}

		// 检查错误处理
		if strings.Contains(line, "err != nil") {
			if !r.hasProperErrorHandling(lines, lineNum) {
				errors = append(errors, ValidationError{
					File:     filePath,
					Line:     lineNum,
					Column:   0,
					Message:  "错误检查后缺少适当的错误处理",
					Severity: SeverityWarning,
				})
			}
		}
	}

	return errors
}

// StructureRule 的辅助方法
// isValidImportStatement 检查导入语句格式
func (r *StructureRule) isValidImportStatement(line string) bool {
	line = strings.TrimSpace(line)

	// 检查基本格式
	if !strings.HasPrefix(line, "import ") {
		return false
	}

	// 检查是否包含引号
	if !strings.Contains(line, `"`) {
		return false
	}

	return true
}

// isValidFunctionDeclaration 检查函数声明格式
func (r *StructureRule) isValidFunctionDeclaration(line string) bool {
	line = strings.TrimSpace(line)

	// 检查基本格式
	if !strings.HasPrefix(line, "func ") {
		return false
	}

	// 检查是否包含函数名
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return false
	}

	// 检查函数名是否有效
	funcName := parts[1]
	if funcName == "" || strings.Contains(funcName, "(") {
		return false
	}

	return true
}

// isValidStructDeclaration 检查结构体声明格式
func (r *StructureRule) isValidStructDeclaration(line string) bool {
	line = strings.TrimSpace(line)

	// 检查基本格式
	if !strings.HasPrefix(line, "type ") || !strings.Contains(line, " struct") {
		return false
	}

	// 检查是否包含结构体名
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return false
	}

	// 检查结构体名是否有效
	structName := parts[1]
	if structName == "" {
		return false
	}

	return true
}

// isValidInterfaceDeclaration 检查接口声明格式
func (r *StructureRule) isValidInterfaceDeclaration(line string) bool {
	line = strings.TrimSpace(line)

	// 检查基本格式
	if !strings.HasPrefix(line, "type ") || !strings.Contains(line, " interface") {
		return false
	}

	// 检查是否包含接口名
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return false
	}

	// 检查接口名是否有效
	interfaceName := parts[1]
	if interfaceName == "" {
		return false
	}

	return true
}

// hasProperErrorHandling 检查是否有适当的错误处理
func (r *StructureRule) hasProperErrorHandling(lines []string, currentLine int) bool {
	// 检查当前行和后续几行是否有错误处理
	for i := currentLine; i < len(lines) && i < currentLine+5; i++ {
		line := strings.TrimSpace(lines[i])

		// 检查是否有return语句
		if strings.HasPrefix(line, "return ") {
			return true
		}

		// 检查是否有日志记录
		if strings.Contains(line, "log.") || strings.Contains(line, "fmt.") {
			return true
		}

		// 检查是否有panic
		if strings.Contains(line, "panic(") {
			return true
		}

		// 检查是否有错误包装
		if strings.Contains(line, "fmt.Errorf") || strings.Contains(line, "errors.Wrap") {
			return true
		}
	}

	return false
}

// NamingRule 的辅助方法
// isValidPackageName 检查包名是否符合约定
func (r *NamingRule) isValidPackageName(name string) bool {
	// 包名应该使用小写字母，可以包含下划线
	if name == "" {
		return false
	}

	// 检查是否只包含小写字母、数字和下划线
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_') {
			return false
		}
	}

	// 不能以下划线开头
	return !strings.HasPrefix(name, "_")
}

// isValidTypeName 检查类型名是否符合约定
func (r *NamingRule) isValidTypeName(name string) bool {
	// 类型名应该使用PascalCase
	if name == "" {
		return false
	}

	// 首字母应该大写
	if len(name) > 0 && (name[0] < 'A' || name[0] > 'Z') {
		return false
	}

	return true
}

// isValidFunctionName 检查函数名是否符合约定
func (r *NamingRule) isValidFunctionName(name string) bool {
	// 函数名应该使用PascalCase
	return r.isValidTypeName(name)
}

// isValidVariableName 检查变量名是否符合约定
func (r *NamingRule) isValidVariableName(name string) bool {
	// 变量名应该使用camelCase
	if name == "" {
		return false
	}

	// 首字母应该小写
	if len(name) > 0 && (name[0] < 'a' || name[0] > 'z') {
		return false
	}

	return true
}

// extractTypeName 从类型声明行中提取类型名
func (r *NamingRule) extractTypeName(line string) string {
	// 匹配 "type TypeName struct" 格式
	parts := strings.Fields(strings.TrimSpace(line))
	if len(parts) >= 3 && parts[0] == "type" && parts[2] == "struct" {
		return parts[1]
	}
	return ""
}

// extractFunctionName 从函数声明行中提取函数名
func (r *NamingRule) extractFunctionName(line string) string {
	// 匹配 "func FunctionName" 格式
	parts := strings.Fields(strings.TrimSpace(line))
	if len(parts) >= 2 && parts[0] == "func" {
		return parts[1]
	}
	return ""
}

// extractVariableName 从变量声明行中提取变量名
func (r *NamingRule) extractVariableName(line string) string {
	// 匹配 "var varName" 或 "varName :=" 格式
	line = strings.TrimSpace(line)

	if strings.HasPrefix(line, "var ") {
		parts := strings.Fields(line)
		if len(parts) > 0 {
			return parts[1]
		}
	} else if strings.Contains(line, " :=") {
		parts := strings.Split(line, " :=")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	return ""
}

// extractLineColumn 从错误信息中提取行号和列号
func extractLineColumn(errorMsg string) (line, column int) {
	// 尝试匹配常见的错误格式
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(\d+):(\d+):`),             // 标准格式: line:column:
		regexp.MustCompile(`line (\d+), column (\d+)`), // 描述性格式
		regexp.MustCompile(`at line (\d+)`),            // 简单行号格式
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(errorMsg)
		if len(matches) >= 3 {
			if l, err := parseNumber(matches[1]); err == nil {
				line = l
			}
			if c, err := parseNumber(matches[2]); err == nil {
				column = c
			}
			return
		} else if len(matches) >= 2 {
			if l, err := parseNumber(matches[1]); err == nil {
				line = l
			}
			return
		}
	}

	return 0, 0
}

// parseNumber 解析数字字符串
func parseNumber(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// ValidateGeneratedCode 验证生成的代码
func ValidateGeneratedCode(outputDir string) error {
	validator := NewCodeValidator()

	// 验证整个输出目录
	if err := validator.ValidateDirectory(outputDir); err != nil {
		return err
	}

	// 打印验证报告
	validator.PrintReport()

	// 检查是否有严重错误
	if validator.HasErrors() {
		return fmt.Errorf("代码验证失败，请检查上述错误")
	}

	return nil
}
