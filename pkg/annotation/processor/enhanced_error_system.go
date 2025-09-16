package processor

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ErrorSeverity 错误严重级别
type ErrorSeverity int

const (
	SeverityInfo ErrorSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ErrorCode 错误代码枚举
type ErrorCode string

const (
	// 解析错误
	ErrInvalidAnnotation     ErrorCode = "E001"
	ErrMissingAttribute      ErrorCode = "E002"
	ErrInvalidAttributeValue ErrorCode = "E003"
	ErrUnsupportedType       ErrorCode = "E004"

	// 生成错误
	ErrTemplateNotFound    ErrorCode = "E101"
	ErrTemplateCompilation ErrorCode = "E102"
	ErrCodeGeneration      ErrorCode = "E103"
	ErrOutputWriting       ErrorCode = "E104"

	// 类型错误
	ErrTypeResolution     ErrorCode = "E201"
	ErrTypeCompatibility  ErrorCode = "E202"
	ErrMissingImport      ErrorCode = "E203"
	ErrCircularDependency ErrorCode = "E204"

	// 配置错误
	ErrInvalidConfig    ErrorCode = "E301"
	ErrMissingConfig    ErrorCode = "E302"
	ErrConfigValidation ErrorCode = "E303"
)

// SourceLocation 源码位置信息
type SourceLocation struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	StartPos int    `json:"start_pos"`
	EndPos   int    `json:"end_pos"`
	Context  string `json:"context,omitempty"`
}

// ErrorSuggestion 修复建议
type ErrorSuggestion struct {
	Type        string `json:"type"` // "fix", "warning", "info"
	Description string `json:"description"`
	Code        string `json:"code,omitempty"`
	Action      string `json:"action,omitempty"`
}

// DiagnosticError 诊断错误结构
type DiagnosticError struct {
	Code        ErrorCode         `json:"code"`
	Severity    ErrorSeverity     `json:"severity"`
	Message     string            `json:"message"`
	Description string            `json:"description,omitempty"`
	Location    *SourceLocation   `json:"location,omitempty"`
	Suggestions []ErrorSuggestion `json:"suggestions,omitempty"`
	Context     interface{}       `json:"context,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	StackTrace  []string          `json:"stack_trace,omitempty"`
}

// Error 实现error接口
func (e *DiagnosticError) Error() string {
	if e.Location != nil {
		return fmt.Sprintf("[%s] %s:%d:%d: %s", e.Code, e.Location.File, e.Location.Line, e.Location.Column, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ToJSON 转换为JSON格式
func (e *DiagnosticError) ToJSON() (string, error) {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ErrorCollector 错误收集器
type ErrorCollector struct {
	errors   []*DiagnosticError
	warnings []*DiagnosticError
	fileSet  *token.FileSet
}

// NewErrorCollector 创建错误收集器
func NewErrorCollector(fileSet *token.FileSet) *ErrorCollector {
	return &ErrorCollector{
		errors:   make([]*DiagnosticError, 0),
		warnings: make([]*DiagnosticError, 0),
		fileSet:  fileSet,
	}
}

// AddError 添加错误
func (ec *ErrorCollector) AddError(code ErrorCode, message string, pos token.Pos, context interface{}) {
	err := &DiagnosticError{
		Code:      code,
		Severity:  SeverityError,
		Message:   message,
		Location:  ec.getSourceLocation(pos),
		Context:   context,
		Timestamp: time.Now(),
	}

	// 添加智能建议
	err.Suggestions = ec.generateSuggestions(code, message, context)

	ec.errors = append(ec.errors, err)
}

// AddWarning 添加警告
func (ec *ErrorCollector) AddWarning(code ErrorCode, message string, pos token.Pos, context interface{}) {
	warning := &DiagnosticError{
		Code:      code,
		Severity:  SeverityWarning,
		Message:   message,
		Location:  ec.getSourceLocation(pos),
		Context:   context,
		Timestamp: time.Now(),
	}

	warning.Suggestions = ec.generateSuggestions(code, message, context)

	ec.warnings = append(ec.warnings, warning)
}

// getSourceLocation 获取源码位置信息
func (ec *ErrorCollector) getSourceLocation(pos token.Pos) *SourceLocation {
	if ec.fileSet == nil || !pos.IsValid() {
		return nil
	}

	position := ec.fileSet.Position(pos)
	return &SourceLocation{
		File:     filepath.Base(position.Filename),
		Line:     position.Line,
		Column:   position.Column,
		StartPos: int(pos),
		EndPos:   int(pos),
	}
}

// generateSuggestions 生成智能修复建议
func (ec *ErrorCollector) generateSuggestions(code ErrorCode, message string, context interface{}) []ErrorSuggestion {
	var suggestions []ErrorSuggestion

	switch code {
	case ErrInvalidAnnotation:
		suggestions = append(suggestions, ErrorSuggestion{
			Type:        "fix",
			Description: "检查注解语法是否正确",
			Code:        "// frame:route({\"method\":\"GET\", \"path\":\"/api/users\"})",
			Action:      "使用标准JSON格式的注解语法",
		})

	case ErrMissingAttribute:
		suggestions = append(suggestions, ErrorSuggestion{
			Type:        "fix",
			Description: "添加缺失的必需属性",
			Code:        "// frame:route({\"method\":\"GET\", \"path\":\"/api/users\"})",
			Action:      "确保所有必需的属性都已指定",
		})

	case ErrTypeResolution:
		suggestions = append(suggestions, ErrorSuggestion{
			Type:        "fix",
			Description: "检查类型导入和定义",
			Code:        "import \"your/package/path\"",
			Action:      "确保相关类型已正确导入",
		})

	case ErrTemplateNotFound:
		suggestions = append(suggestions, ErrorSuggestion{
			Type:        "fix",
			Description: "检查模板文件路径",
			Action:      "确保模板文件存在且路径正确",
		})

	case ErrInvalidConfig:
		suggestions = append(suggestions, ErrorSuggestion{
			Type:        "fix",
			Description: "检查配置文件格式",
			Action:      "验证配置文件的JSON/YAML语法",
		})
	}

	// 添加通用建议
	suggestions = append(suggestions, ErrorSuggestion{
		Type:        "info",
		Description: "查看详细文档获取更多帮助",
		Action:      "访问 https://github.com/your-repo/gRain/docs",
	})

	return suggestions
}

// HasErrors 是否有错误
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// HasWarnings 是否有警告
func (ec *ErrorCollector) HasWarnings() bool {
	return len(ec.warnings) > 0
}

// GetErrors 获取所有错误
func (ec *ErrorCollector) GetErrors() []*DiagnosticError {
	return ec.errors
}

// GetWarnings 获取所有警告
func (ec *ErrorCollector) GetWarnings() []*DiagnosticError {
	return ec.warnings
}

// GetAllDiagnostics 获取所有诊断信息
func (ec *ErrorCollector) GetAllDiagnostics() []*DiagnosticError {
	all := make([]*DiagnosticError, 0, len(ec.errors)+len(ec.warnings))
	all = append(all, ec.errors...)
	all = append(all, ec.warnings...)

	// 按位置排序
	sort.Slice(all, func(i, j int) bool {
		if all[i].Location == nil || all[j].Location == nil {
			return false
		}
		if all[i].Location.File != all[j].Location.File {
			return all[i].Location.File < all[j].Location.File
		}
		return all[i].Location.Line < all[j].Location.Line
	})

	return all
}

// Clear 清空所有错误和警告
func (ec *ErrorCollector) Clear() {
	ec.errors = ec.errors[:0]
	ec.warnings = ec.warnings[:0]
}

// DiagnosticReporter 诊断报告器
type DiagnosticReporter struct {
	collector *ErrorCollector
}

// NewDiagnosticReporter 创建诊断报告器
func NewDiagnosticReporter(fileSet *token.FileSet) *DiagnosticReporter {
	return &DiagnosticReporter{
		collector: NewErrorCollector(fileSet),
	}
}

// GetCollector 获取错误收集器
func (dr *DiagnosticReporter) GetCollector() *ErrorCollector {
	return dr.collector
}

// GenerateReport 生成诊断报告
func (dr *DiagnosticReporter) GenerateReport() *DiagnosticReport {
	diagnostics := dr.collector.GetAllDiagnostics()

	report := &DiagnosticReport{
		Summary: DiagnosticSummary{
			TotalErrors:   len(dr.collector.GetErrors()),
			TotalWarnings: len(dr.collector.GetWarnings()),
			TotalIssues:   len(diagnostics),
			Timestamp:     time.Now(),
		},
		Diagnostics: diagnostics,
	}

	// 生成统计信息
	report.Statistics = dr.generateStatistics(diagnostics)

	return report
}

// DiagnosticSummary 诊断摘要
type DiagnosticSummary struct {
	TotalErrors   int       `json:"total_errors"`
	TotalWarnings int       `json:"total_warnings"`
	TotalIssues   int       `json:"total_issues"`
	Timestamp     time.Time `json:"timestamp"`
}

// DiagnosticStatistics 诊断统计
type DiagnosticStatistics struct {
	ErrorsByCode     map[ErrorCode]int     `json:"errors_by_code"`
	ErrorsByFile     map[string]int        `json:"errors_by_file"`
	ErrorsBySeverity map[ErrorSeverity]int `json:"errors_by_severity"`
	MostCommonErrors []ErrorFrequency      `json:"most_common_errors"`
}

// ErrorFrequency 错误频率
type ErrorFrequency struct {
	Code    ErrorCode `json:"code"`
	Count   int       `json:"count"`
	Message string    `json:"message"`
}

// DiagnosticReport 诊断报告
type DiagnosticReport struct {
	Summary     DiagnosticSummary    `json:"summary"`
	Statistics  DiagnosticStatistics `json:"statistics"`
	Diagnostics []*DiagnosticError   `json:"diagnostics"`
}

// generateStatistics 生成统计信息
func (dr *DiagnosticReporter) generateStatistics(diagnostics []*DiagnosticError) DiagnosticStatistics {
	stats := DiagnosticStatistics{
		ErrorsByCode:     make(map[ErrorCode]int),
		ErrorsByFile:     make(map[string]int),
		ErrorsBySeverity: make(map[ErrorSeverity]int),
	}

	// 统计各种维度的错误
	for _, diag := range diagnostics {
		stats.ErrorsByCode[diag.Code]++
		stats.ErrorsBySeverity[diag.Severity]++

		if diag.Location != nil {
			stats.ErrorsByFile[diag.Location.File]++
		}
	}

	// 生成最常见错误列表
	type codeCount struct {
		code  ErrorCode
		count int
	}

	var codeCounts []codeCount
	for code, count := range stats.ErrorsByCode {
		codeCounts = append(codeCounts, codeCount{code, count})
	}

	sort.Slice(codeCounts, func(i, j int) bool {
		return codeCounts[i].count > codeCounts[j].count
	})

	// 取前5个最常见的错误
	maxCount := 5
	if len(codeCounts) < maxCount {
		maxCount = len(codeCounts)
	}

	for i := 0; i < maxCount; i++ {
		stats.MostCommonErrors = append(stats.MostCommonErrors, ErrorFrequency{
			Code:    codeCounts[i].code,
			Count:   codeCounts[i].count,
			Message: dr.getErrorMessage(codeCounts[i].code),
		})
	}

	return stats
}

// getErrorMessage 获取错误代码对应的消息
func (dr *DiagnosticReporter) getErrorMessage(code ErrorCode) string {
	messages := map[ErrorCode]string{
		ErrInvalidAnnotation:     "Invalid annotation syntax",
		ErrMissingAttribute:      "Missing required attribute",
		ErrInvalidAttributeValue: "Invalid attribute value",
		ErrUnsupportedType:       "Unsupported type",
		ErrTemplateNotFound:      "Template file not found",
		ErrTemplateCompilation:   "Template compilation failed",
		ErrCodeGeneration:        "Code generation failed",
		ErrOutputWriting:         "Failed to write output",
		ErrTypeResolution:        "Type resolution failed",
		ErrTypeCompatibility:     "Type compatibility error",
		ErrMissingImport:         "Missing import",
		ErrCircularDependency:    "Circular dependency detected",
		ErrInvalidConfig:         "Invalid configuration",
		ErrMissingConfig:         "Missing configuration",
		ErrConfigValidation:      "Configuration validation failed",
	}

	if msg, exists := messages[code]; exists {
		return msg
	}
	return "Unknown error"
}

// ToJSON 转换报告为JSON
func (dr *DiagnosticReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(dr, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// PrintSummary 打印摘要
func (dr *DiagnosticReport) PrintSummary() string {
	var sb strings.Builder

	sb.WriteString("=== Diagnostic Summary ===\n")
	sb.WriteString(fmt.Sprintf("Total Issues: %d\n", dr.Summary.TotalIssues))
	sb.WriteString(fmt.Sprintf("Errors: %d\n", dr.Summary.TotalErrors))
	sb.WriteString(fmt.Sprintf("Warnings: %d\n", dr.Summary.TotalWarnings))
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n", dr.Summary.Timestamp.Format(time.RFC3339)))

	if len(dr.Statistics.MostCommonErrors) > 0 {
		sb.WriteString("\nMost Common Issues:\n")
		for i, freq := range dr.Statistics.MostCommonErrors {
			sb.WriteString(fmt.Sprintf("%d. [%s] %s (%d occurrences)\n",
				i+1, freq.Code, freq.Message, freq.Count))
		}
	}

	return sb.String()
}

// EnhancedErrorHandler 增强的错误处理器
type EnhancedErrorHandler struct {
	reporter    *DiagnosticReporter
	fileSet     *token.FileSet
	contextInfo map[string]interface{}
}

// NewEnhancedErrorHandler 创建增强错误处理器
func NewEnhancedErrorHandler(fileSet *token.FileSet) *EnhancedErrorHandler {
	return &EnhancedErrorHandler{
		reporter:    NewDiagnosticReporter(fileSet),
		fileSet:     fileSet,
		contextInfo: make(map[string]interface{}),
	}
}

// SetContext 设置上下文信息
func (eh *EnhancedErrorHandler) SetContext(key string, value interface{}) {
	eh.contextInfo[key] = value
}

// HandleAnnotationError 处理注解错误
func (eh *EnhancedErrorHandler) HandleAnnotationError(node ast.Node, code ErrorCode, message string) {
	var pos token.Pos
	if node != nil {
		pos = node.Pos()
	}

	eh.reporter.GetCollector().AddError(code, message, pos, eh.contextInfo)
}

// HandleTypeError 处理类型错误
func (eh *EnhancedErrorHandler) HandleTypeError(pos token.Pos, code ErrorCode, message string, typeInfo interface{}) {
	context := map[string]interface{}{
		"type_info": typeInfo,
	}
	for k, v := range eh.contextInfo {
		context[k] = v
	}

	eh.reporter.GetCollector().AddError(code, message, pos, context)
}

// GetReport 获取诊断报告
func (eh *EnhancedErrorHandler) GetReport() *DiagnosticReport {
	return eh.reporter.GenerateReport()
}

// HasErrors 是否有错误
func (eh *EnhancedErrorHandler) HasErrors() bool {
	return eh.reporter.GetCollector().HasErrors()
}

// Clear 清空错误
func (eh *EnhancedErrorHandler) Clear() {
	eh.reporter.GetCollector().Clear()
}
