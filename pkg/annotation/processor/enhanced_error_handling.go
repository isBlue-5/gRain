// Package processor provides enhanced error handling and logging capabilities
package processor

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ErrorLevel 错误级别
type ErrorLevel int

const (
	ErrorLevelInfo ErrorLevel = iota
	ErrorLevelWarn
	ErrorLevelError
	ErrorLevelFatal
)

// String 返回错误级别的字符串表示
func (el ErrorLevel) String() string {
	switch el {
	case ErrorLevelInfo:
		return "INFO"
	case ErrorLevelWarn:
		return "WARN"
	case ErrorLevelError:
		return "ERROR"
	case ErrorLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ProcessorError 处理器错误
type ProcessorError struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Level     ErrorLevel             `json:"level"`
	Timestamp time.Time              `json:"timestamp"`
	File      string                 `json:"file"`
	Line      int                    `json:"line"`
	Stack     string                 `json:"stack"`
	Context   map[string]interface{} `json:"context"`
	Cause     error                  `json:"cause,omitempty"`
}

// Error 实现error接口
func (pe *ProcessorError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", pe.Level, pe.Code, pe.Message)
}

// Unwrap 返回原始错误
func (pe *ProcessorError) Unwrap() error {
	return pe.Cause
}

// WithContext 添加上下文信息
func (pe *ProcessorError) WithContext(key string, value interface{}) *ProcessorError {
	if pe.Context == nil {
		pe.Context = make(map[string]interface{})
	}
	pe.Context[key] = value
	return pe
}

// NewProcessorError 创建新的处理器错误
func NewProcessorError(code, message string, level ErrorLevel) *ProcessorError {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	}

	return &ProcessorError{
		Code:      code,
		Message:   message,
		Level:     level,
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
		Stack:     getStackTrace(),
		Context:   make(map[string]interface{}),
	}
}

// WrapError 包装现有错误
func WrapError(err error, code, message string, level ErrorLevel) *ProcessorError {
	pe := NewProcessorError(code, message, level)
	pe.Cause = err
	return pe
}

// getStackTrace 获取堆栈跟踪
func getStackTrace() string {
	const size = 64 << 10
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	return string(buf)
}

// Logger 日志记录器接口
type Logger interface {
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	Debug(msg string, fields ...Field)
}

// Field 日志字段
type Field struct {
	Key   string
	Value interface{}
}

// String 创建字符串字段
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int 创建整数字段
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Error 创建错误字段
func Error(err error) Field {
	return Field{Key: "error", Value: err}
}

// Duration 创建时长字段
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

// DefaultLogger 默认日志记录器
type DefaultLogger struct {
	level       ErrorLevel
	debugMode   bool
	verboseMode bool
	mu          sync.RWMutex
}

// NewDefaultLogger 创建默认日志记录器
func NewDefaultLogger(level ErrorLevel) *DefaultLogger {
	return &DefaultLogger{level: level}
}

// Info 记录信息日志
func (dl *DefaultLogger) Info(msg string, fields ...Field) {
	if dl.level <= ErrorLevelInfo {
		dl.log("INFO", msg, fields...)
	}
}

// Warn 记录警告日志
func (dl *DefaultLogger) Warn(msg string, fields ...Field) {
	if dl.level <= ErrorLevelWarn {
		dl.log("WARN", msg, fields...)
	}
}

// Error 记录错误日志
func (dl *DefaultLogger) Error(msg string, fields ...Field) {
	if dl.level <= ErrorLevelError {
		dl.log("ERROR", msg, fields...)
	}
}

// Fatal 记录致命错误日志
func (dl *DefaultLogger) Fatal(msg string, fields ...Field) {
	dl.log("FATAL", msg, fields...)
}

// Debug 记录调试日志
func (dl *DefaultLogger) Debug(msg string, fields ...Field) {
	// Debug 级别比 Info 更低，但在我们的系统中暂不实现
	// 可以根据需要扩展

	// 检查是否启用调试模式
	if !dl.isDebugEnabled() {
		return
	}

	// 记录调试信息
	dl.log("DEBUG", msg, fields...)

	// 如果启用了详细调试，记录更多信息
	if dl.isVerboseDebugEnabled() {
		dl.logDebugDetails(msg, fields...)
	}
}

// log 内部日志记录方法
func (dl *DefaultLogger) log(level, msg string, fields ...Field) {
	var fieldStrs []string
	for _, field := range fields {
		fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", field.Key, field.Value))
	}

	logMsg := fmt.Sprintf("[%s] %s", level, msg)
	if len(fieldStrs) > 0 {
		logMsg += " " + strings.Join(fieldStrs, " ")
	}

	log.Println(logMsg)
}

// isDebugEnabled 检查是否启用调试模式
func (dl *DefaultLogger) isDebugEnabled() bool {
	dl.mu.RLock()
	defer dl.mu.RUnlock()
	return dl.debugMode
}

// isVerboseDebugEnabled 检查是否启用详细调试模式
func (dl *DefaultLogger) isVerboseDebugEnabled() bool {
	dl.mu.RLock()
	defer dl.mu.RUnlock()
	return dl.debugMode && dl.verboseMode
}

// logDebugDetails 记录详细的调试信息
func (dl *DefaultLogger) logDebugDetails(msg string, fields ...Field) {
	// 记录调用栈信息
	if stack := getCallStack(); stack != "" {
		dl.log("DEBUG_STACK", "调用栈信息", String("stack", stack))
	}

	// 记录内存使用情况
	if memStats := getMemoryStats(); memStats != "" {
		dl.log("DEBUG_MEMORY", "内存使用情况", String("stats", memStats))
	}

	// 记录字段详细信息
	for _, field := range fields {
		dl.log("DEBUG_FIELD", fmt.Sprintf("字段: %s", field.Key),
			String("value", fmt.Sprintf("%v", field.Value)),
			String("type", fmt.Sprintf("%T", field.Value)))
	}
}

// getCallStack 获取调用栈信息
func getCallStack() string {
	// 获取调用栈（限制深度以避免性能问题）
	var stack []string
	for i := 1; i < 10; i++ {
		if pc, file, line, ok := runtime.Caller(i); ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				stack = append(stack, fmt.Sprintf("%s:%d %s", filepath.Base(file), line, fn.Name()))
			}
		}
	}

	if len(stack) > 0 {
		return strings.Join(stack, " -> ")
	}
	return ""
}

// getMemoryStats 获取内存统计信息
func getMemoryStats() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return fmt.Sprintf("Alloc=%d, TotalAlloc=%d, Sys=%d, NumGC=%d",
		m.Alloc, m.TotalAlloc, m.Sys, m.NumGC)
}

// ErrorHandler 错误处理器
type ErrorHandler struct {
	logger Logger
	errors []ProcessorError
}

// NewErrorHandler 创建新的错误处理器
func NewErrorHandler(logger Logger) *ErrorHandler {
	if logger == nil {
		logger = NewDefaultLogger(ErrorLevelInfo)
	}

	return &ErrorHandler{
		logger: logger,
		errors: make([]ProcessorError, 0),
	}
}

// HandleError 处理错误
func (eh *ErrorHandler) HandleError(err error) {
	if err == nil {
		return
	}

	var pe *ProcessorError
	if processorErr, ok := err.(*ProcessorError); ok {
		pe = processorErr
	} else {
		pe = WrapError(err, "UNKNOWN_ERROR", err.Error(), ErrorLevelError)
	}

	// 记录错误
	eh.errors = append(eh.errors, *pe)

	// 根据错误级别记录日志
	switch pe.Level {
	case ErrorLevelInfo:
		eh.logger.Info(pe.Message,
			String("code", pe.Code),
			String("file", pe.File),
			Int("line", pe.Line))
	case ErrorLevelWarn:
		eh.logger.Warn(pe.Message,
			String("code", pe.Code),
			String("file", pe.File),
			Int("line", pe.Line))
	case ErrorLevelError:
		eh.logger.Error(pe.Message,
			String("code", pe.Code),
			String("file", pe.File),
			Int("line", pe.Line),
			Error(pe.Cause))
	case ErrorLevelFatal:
		eh.logger.Fatal(pe.Message,
			String("code", pe.Code),
			String("file", pe.File),
			Int("line", pe.Line),
			Error(pe.Cause))
	}
}

// GetErrors 获取所有错误
func (eh *ErrorHandler) GetErrors() []ProcessorError {
	return eh.errors
}

// HasErrors 检查是否有错误
func (eh *ErrorHandler) HasErrors() bool {
	return len(eh.errors) > 0
}

// HasFatalErrors 检查是否有致命错误
func (eh *ErrorHandler) HasFatalErrors() bool {
	for _, err := range eh.errors {
		if err.Level == ErrorLevelFatal {
			return true
		}
	}
	return false
}

// Clear 清除所有错误
func (eh *ErrorHandler) Clear() {
	eh.errors = eh.errors[:0]
}

// GenerateErrorReport 生成错误报告
func (eh *ErrorHandler) GenerateErrorReport() string {
	if len(eh.errors) == 0 {
		return "No errors recorded."
	}

	report := fmt.Sprintf("Error Report (%d errors)\n", len(eh.errors))
	report += "=====================================\n\n"

	for i, err := range eh.errors {
		report += fmt.Sprintf("Error #%d:\n", i+1)
		report += fmt.Sprintf("  Code: %s\n", err.Code)
		report += fmt.Sprintf("  Level: %s\n", err.Level)
		report += fmt.Sprintf("  Message: %s\n", err.Message)
		report += fmt.Sprintf("  Time: %s\n", err.Timestamp.Format("2006-01-02 15:04:05"))
		report += fmt.Sprintf("  Location: %s:%d\n", err.File, err.Line)

		if len(err.Context) > 0 {
			report += "  Context:\n"
			for k, v := range err.Context {
				report += fmt.Sprintf("    %s: %v\n", k, v)
			}
		}

		if err.Cause != nil {
			report += fmt.Sprintf("  Cause: %s\n", err.Cause.Error())
		}

		report += "\n"
	}

	return report
}

// GlobalErrorHandler 全局错误处理器
var GlobalErrorHandler = NewErrorHandler(NewDefaultLogger(ErrorLevelInfo))

// 便捷函数

// LogInfo 记录信息日志
func LogInfo(msg string, fields ...Field) {
	GlobalErrorHandler.logger.Info(msg, fields...)
}

// LogWarn 记录警告日志
func LogWarn(msg string, fields ...Field) {
	GlobalErrorHandler.logger.Warn(msg, fields...)
}

// LogError 记录错误日志
func LogError(msg string, fields ...Field) {
	GlobalErrorHandler.logger.Error(msg, fields...)
}

// LogFatal 记录致命错误日志
func LogFatal(msg string, fields ...Field) {
	GlobalErrorHandler.logger.Fatal(msg, fields...)
}

// HandleError 处理错误
func HandleError(err error) {
	GlobalErrorHandler.HandleError(err)
}

// NewInfoError 创建信息级错误
func NewInfoError(code, message string) *ProcessorError {
	return NewProcessorError(code, message, ErrorLevelInfo)
}

// NewWarnError 创建警告级错误
func NewWarnError(code, message string) *ProcessorError {
	return NewProcessorError(code, message, ErrorLevelWarn)
}

// NewError 创建错误级错误
func NewError(code, message string) *ProcessorError {
	return NewProcessorError(code, message, ErrorLevelError)
}

// NewFatalError 创建致命错误
func NewFatalError(code, message string) *ProcessorError {
	return NewProcessorError(code, message, ErrorLevelFatal)
}

// SafeExecute 安全执行函数，自动处理错误
func SafeExecute(operation string, fn func() error) error {
	LogInfo("开始执行操作", String("operation", operation))

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	if err != nil {
		LogError("操作执行失败",
			String("operation", operation),
			Duration("duration", duration),
			Error(err))
		HandleError(WrapError(err, "OPERATION_FAILED",
			fmt.Sprintf("操作 %s 执行失败", operation), ErrorLevelError))
		return err
	}

	LogInfo("操作执行成功",
		String("operation", operation),
		Duration("duration", duration))

	return nil
}

// RecoverableExecute 可恢复执行函数，捕获panic
func RecoverableExecute(operation string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = NewFatalError("PANIC_RECOVERED",
				fmt.Sprintf("操作 %s 发生panic: %v", operation, r))
			HandleError(err)
		}
	}()

	return SafeExecute(operation, fn)
}
