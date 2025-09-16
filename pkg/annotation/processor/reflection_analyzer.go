// Package processor 提供反射使用情况分析工具
package processor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ReflectionUsage 反射使用情况记录
type ReflectionUsage struct {
	// 文件路径
	FilePath string
	// 行号
	LineNumber int
	// 函数名
	FunctionName string
	// 反射调用类型
	CallType string
	// 具体代码
	Code string
	// 优化建议
	Suggestion string
}

// ReflectionAnalyzer 反射使用分析器
type ReflectionAnalyzer struct {
	// 文件集
	fileSet *token.FileSet
	// 反射使用记录
	usages []ReflectionUsage
	// 反射调用模式
	reflectPatterns []*regexp.Regexp
}

// NewReflectionAnalyzer 创建反射分析器
func NewReflectionAnalyzer() *ReflectionAnalyzer {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`reflect\.ValueOf`),
		regexp.MustCompile(`reflect\.TypeOf`),
		regexp.MustCompile(`reflect\.New`),
		regexp.MustCompile(`reflect\.MakeSlice`),
		regexp.MustCompile(`reflect\.MakeMap`),
		regexp.MustCompile(`\.MethodByName`),
		regexp.MustCompile(`\.FieldByName`),
		regexp.MustCompile(`\.Call`),
		regexp.MustCompile(`\.Set`),
		regexp.MustCompile(`\.Interface\(\)`),
		regexp.MustCompile(`\.Kind\(\)`),
		regexp.MustCompile(`\.Type\(\)`),
	}

	return &ReflectionAnalyzer{
		fileSet:         token.NewFileSet(),
		usages:          make([]ReflectionUsage, 0),
		reflectPatterns: patterns,
	}
}

// AnalyzeDirectory 分析目录中的反射使用情况
func (ra *ReflectionAnalyzer) AnalyzeDirectory(dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只分析Go文件
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		return ra.analyzeFile(path)
	})
}

// analyzeFile 分析单个文件的反射使用情况
func (ra *ReflectionAnalyzer) analyzeFile(filePath string) error {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败 %s: %w", filePath, err)
	}

	// 解析AST
	file, err := parser.ParseFile(ra.fileSet, filePath, content, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("解析文件失败 %s: %w", filePath, err)
	}

	// 分析AST中的反射使用
	ra.analyzeAST(file, filePath, string(content))

	return nil
}

// analyzeAST 分析AST中的反射使用
func (ra *ReflectionAnalyzer) analyzeAST(file *ast.File, filePath string, content string) {
	lines := strings.Split(content, "\n")

	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.CallExpr:
			ra.analyzeCallExpr(n, filePath, lines)
		case *ast.SelectorExpr:
			ra.analyzeSelectorExpr(n, filePath, lines)
		}
		return true
	})
}

// analyzeCallExpr 分析函数调用表达式
func (ra *ReflectionAnalyzer) analyzeCallExpr(call *ast.CallExpr, filePath string, lines []string) {
	pos := ra.fileSet.Position(call.Pos())
	if pos.Line <= 0 || pos.Line > len(lines) {
		return
	}

	line := lines[pos.Line-1]

	// 检查是否是反射调用
	for _, pattern := range ra.reflectPatterns {
		if pattern.MatchString(line) {
			usage := ReflectionUsage{
				FilePath:     filePath,
				LineNumber:   pos.Line,
				FunctionName: ra.getCurrentFunction(call, filePath),
				CallType:     ra.identifyCallType(line),
				Code:         strings.TrimSpace(line),
				Suggestion:   ra.generateSuggestion(line),
			}
			ra.usages = append(ra.usages, usage)
			break
		}
	}
}

// analyzeSelectorExpr 分析选择器表达式
func (ra *ReflectionAnalyzer) analyzeSelectorExpr(sel *ast.SelectorExpr, filePath string, lines []string) {
	pos := ra.fileSet.Position(sel.Pos())
	if pos.Line <= 0 || pos.Line > len(lines) {
		return
	}

	line := lines[pos.Line-1]

	// 检查是否是反射相关的方法调用
	reflectMethods := []string{"MethodByName", "FieldByName", "Call", "Set", "Interface", "Kind", "Type"}
	for _, method := range reflectMethods {
		if sel.Sel.Name == method && strings.Contains(line, "reflect") {
			usage := ReflectionUsage{
				FilePath:     filePath,
				LineNumber:   pos.Line,
				FunctionName: ra.getCurrentFunction(sel, filePath),
				CallType:     method,
				Code:         strings.TrimSpace(line),
				Suggestion:   ra.generateSuggestion(line),
			}
			ra.usages = append(ra.usages, usage)
			break
		}
	}
}

// getCurrentFunction 获取当前所在的函数名
func (ra *ReflectionAnalyzer) getCurrentFunction(node ast.Node, filePath string) string {
	// 简化实现，返回文件名
	return filepath.Base(filePath)
}

// identifyCallType 识别反射调用类型
func (ra *ReflectionAnalyzer) identifyCallType(line string) string {
	if strings.Contains(line, "reflect.ValueOf") {
		return "ValueOf"
	}
	if strings.Contains(line, "reflect.TypeOf") {
		return "TypeOf"
	}
	if strings.Contains(line, "MethodByName") {
		return "MethodByName"
	}
	if strings.Contains(line, "FieldByName") {
		return "FieldByName"
	}
	if strings.Contains(line, ".Call") {
		return "Call"
	}
	return "Unknown"
}

// generateSuggestion 生成优化建议
func (ra *ReflectionAnalyzer) generateSuggestion(line string) string {
	if strings.Contains(line, "reflect.ValueOf") {
		return "考虑使用类型断言或接口替换ValueOf"
	}
	if strings.Contains(line, "reflect.TypeOf") {
		return "考虑使用编译时类型信息替换TypeOf"
	}
	if strings.Contains(line, "MethodByName") {
		return "考虑使用直接方法调用替换MethodByName"
	}
	if strings.Contains(line, "FieldByName") {
		return "考虑使用结构体字段直接访问替换FieldByName"
	}
	if strings.Contains(line, ".Call") {
		return "考虑生成类型安全的调用代码替换反射Call"
	}
	return "考虑使用编译时代码生成替换反射"
}

// GetUsages 获取所有反射使用记录
func (ra *ReflectionAnalyzer) GetUsages() []ReflectionUsage {
	return ra.usages
}

// GenerateReport 生成反射使用报告
func (ra *ReflectionAnalyzer) GenerateReport() string {
	if len(ra.usages) == 0 {
		return "未发现反射使用"
	}

	var report strings.Builder
	report.WriteString("# 反射使用情况分析报告\n\n")
	report.WriteString(fmt.Sprintf("**总计发现**: %d 处反射使用\n\n", len(ra.usages)))

	// 按文件分组
	fileUsages := make(map[string][]ReflectionUsage)
	for _, usage := range ra.usages {
		fileUsages[usage.FilePath] = append(fileUsages[usage.FilePath], usage)
	}

	for filePath, usages := range fileUsages {
		report.WriteString(fmt.Sprintf("## 文件: %s\n\n", filePath))
		report.WriteString(fmt.Sprintf("**反射调用次数**: %d\n\n", len(usages)))

		for i, usage := range usages {
			report.WriteString(fmt.Sprintf("### %d. %s (第%d行)\n\n", i+1, usage.CallType, usage.LineNumber))
			report.WriteString(fmt.Sprintf("**代码**: `%s`\n\n", usage.Code))
			report.WriteString(fmt.Sprintf("**优化建议**: %s\n\n", usage.Suggestion))
		}
	}

	// 统计信息
	report.WriteString("## 统计信息\n\n")
	callTypes := make(map[string]int)
	for _, usage := range ra.usages {
		callTypes[usage.CallType]++
	}

	report.WriteString("| 反射调用类型 | 使用次数 |\n")
	report.WriteString("|--------------|----------|\n")
	for callType, count := range callTypes {
		report.WriteString(fmt.Sprintf("| %s | %d |\n", callType, count))
	}

	return report.String()
}

// AnalyzeProcessorDirectory 分析processor目录的反射使用情况
func AnalyzeProcessorReflectionUsage() (*ReflectionAnalyzer, error) {
	analyzer := NewReflectionAnalyzer()

	// 分析processor目录
	err := analyzer.AnalyzeDirectory(".")
	if err != nil {
		return nil, fmt.Errorf("分析processor目录失败: %w", err)
	}

	return analyzer, nil
}
