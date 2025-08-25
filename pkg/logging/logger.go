package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level 日志级别
type Level int

const (
	// DebugLevel 调试级别
	DebugLevel Level = iota
	// InfoLevel 信息级别
	InfoLevel
	// WarnLevel 警告级别
	WarnLevel
	// ErrorLevel 错误级别
	ErrorLevel
	// FatalLevel 致命级别
	FatalLevel
)

// String 级别字符串表示
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return fmt.Sprintf("Level(%d)", l)
	}
}

// ParseLevel 解析日志级别字符串
func ParseLevel(level string) (Level, error) {
	switch level {
	case "debug", "DEBUG":
		return DebugLevel, nil
	case "info", "INFO":
		return InfoLevel, nil
	case "warn", "WARN", "warning", "WARNING":
		return WarnLevel, nil
	case "error", "ERROR":
		return ErrorLevel, nil
	case "fatal", "FATAL":
		return FatalLevel, nil
	default:
		return InfoLevel, fmt.Errorf("unknown log level: %s", level)
	}
}

// Field 日志字段
type Field struct {
	Key   string
	Value interface{}
}

// Fields 创建日志字段列表
func Fields(keysAndValues ...interface{}) []Field {
	if len(keysAndValues)%2 != 0 {
		keysAndValues = append(keysAndValues, "MISSING_VALUE")
	}

	fields := make([]Field, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			key = fmt.Sprintf("%v", keysAndValues[i])
		}
		fields = append(fields, Field{Key: key, Value: keysAndValues[i+1]})
	}

	return fields
}

// Entry 日志条目
type Entry struct {
	Time    time.Time
	Level   Level
	Message string
	Fields  []Field
	TraceID string
}

// Logger 日志记录器接口
type Logger interface {
	// Debug 调试级别日志
	Debug(msg string, fields ...Field)

	// Info 信息级别日志
	Info(msg string, fields ...Field)

	// Warn 警告级别日志
	Warn(msg string, fields ...Field)

	// Error 错误级别日志
	Error(msg string, fields ...Field)

	// Fatal 致命级别日志
	Fatal(msg string, fields ...Field)

	// With 创建带有固定字段的新记录器
	With(fields ...Field) Logger

	// WithContext 从上下文创建新记录器
	WithContext(ctx context.Context) Logger

	// SetLevel 设置日志级别
	SetLevel(level Level)

	// GetLevel 获取当前日志级别
	GetLevel() Level

	// Sync 刷新缓存的日志
	Sync() error
}

// Formatter 日志格式化器接口
type Formatter interface {
	Format(entry *Entry) ([]byte, error)
}

// Option 日志选项函数
type Option func(*DefaultLogger)

// WithLevel 设置日志级别
func WithLevel(level Level) Option {
	return func(logger *DefaultLogger) {
		logger.level = level
	}
}

// WithFormatter 设置格式化器
func WithFormatter(formatter Formatter) Option {
	return func(logger *DefaultLogger) {
		logger.formatter = formatter
	}
}

// WithOutput 添加输出目标
func WithOutput(output io.Writer) Option {
	return func(logger *DefaultLogger) {
		logger.mu.Lock()
		defer logger.mu.Unlock()
		logger.outputs = append(logger.outputs, output)
	}
}

// WithContextKeys 设置从上下文提取的键
func WithContextKeys(keys []string) Option {
	return func(logger *DefaultLogger) {
		logger.contextKeys = keys
	}
}

// DefaultLogger 默认日志记录器
type DefaultLogger struct {
	level       Level
	outputs     []io.Writer
	formatter   Formatter
	contextKeys []string
	mu          sync.Mutex
	fields      []Field
}

// NewLogger 创建默认日志记录器
func NewLogger(options ...Option) *DefaultLogger {
	logger := &DefaultLogger{
		level:     InfoLevel,
		outputs:   []io.Writer{os.Stdout},
		formatter: &JSONFormatter{},
		fields:    make([]Field, 0),
	}

	for _, opt := range options {
		opt(logger)
	}

	return logger
}

// newEntry 创建日志条目
func (l *DefaultLogger) newEntry(level Level, msg string, fields []Field) *Entry {
	// 合并固定字段和传入字段
	allFields := make([]Field, 0, len(l.fields)+len(fields))
	allFields = append(allFields, l.fields...)
	allFields = append(allFields, fields...)

	return &Entry{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
		Fields:  allFields,
	}
}

// write 写入日志
func (l *DefaultLogger) write(entry *Entry) {
	data, err := l.formatter.Format(entry)
	if err != nil {
		// 格式化失败，使用简单格式
		data = []byte(fmt.Sprintf("[%s] %s: %s\n", entry.Level.String(), entry.Time.Format(time.RFC3339), entry.Message))
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for _, output := range l.outputs {
		output.Write(data)
	}
}

// Debug 实现Logger接口
func (l *DefaultLogger) Debug(msg string, fields ...Field) {
	if l.level > DebugLevel {
		return
	}
	entry := l.newEntry(DebugLevel, msg, fields)
	l.write(entry)
}

// Info 实现Logger接口
func (l *DefaultLogger) Info(msg string, fields ...Field) {
	if l.level > InfoLevel {
		return
	}
	entry := l.newEntry(InfoLevel, msg, fields)
	l.write(entry)
}

// Warn 实现Logger接口
func (l *DefaultLogger) Warn(msg string, fields ...Field) {
	if l.level > WarnLevel {
		return
	}
	entry := l.newEntry(WarnLevel, msg, fields)
	l.write(entry)
}

// Error 实现Logger接口
func (l *DefaultLogger) Error(msg string, fields ...Field) {
	if l.level > ErrorLevel {
		return
	}
	entry := l.newEntry(ErrorLevel, msg, fields)
	l.write(entry)
}

// Fatal 实现Logger接口
func (l *DefaultLogger) Fatal(msg string, fields ...Field) {
	if l.level > FatalLevel {
		return
	}
	entry := l.newEntry(FatalLevel, msg, fields)
	l.write(entry)
	os.Exit(1)
}

// With 实现Logger接口
func (l *DefaultLogger) With(fields ...Field) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &DefaultLogger{
		level:       l.level,
		outputs:     l.outputs,
		formatter:   l.formatter,
		contextKeys: l.contextKeys,
		fields:      make([]Field, 0, len(l.fields)+len(fields)),
	}

	// 复制固定字段
	newLogger.fields = append(newLogger.fields, l.fields...)
	// 添加新字段
	newLogger.fields = append(newLogger.fields, fields...)

	return newLogger
}

// WithContext 实现Logger接口
func (l *DefaultLogger) WithContext(ctx context.Context) Logger {
	if len(l.contextKeys) == 0 {
		return l
	}

	fields := make([]Field, 0, len(l.contextKeys))
	for _, key := range l.contextKeys {
		if value := ctx.Value(key); value != nil {
			fields = append(fields, Field{Key: key, Value: value})
		}
	}

	if len(fields) == 0 {
		return l
	}

	return l.With(fields...)
}

// SetLevel 实现Logger接口
func (l *DefaultLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel 实现Logger接口
func (l *DefaultLogger) GetLevel() Level {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// Sync 实现Logger接口
func (l *DefaultLogger) Sync() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 遍历所有输出，执行同步操作
	for _, output := range l.outputs {
		if syncer, ok := output.(interface{ Sync() error }); ok {
			// 如果输出支持同步，调用其Sync方法
			if err := syncer.Sync(); err != nil {
				return fmt.Errorf("同步输出失败: %w", err)
			}
		}

		// 对于缓冲的Writer，尝试刷新
		if flusher, ok := output.(interface{ Flush() error }); ok {
			if err := flusher.Flush(); err != nil {
				return fmt.Errorf("刷新输出失败: %w", err)
			}
		}

		// 对于文件输出，尝试同步到磁盘
		if file, ok := output.(*os.File); ok {
			if err := file.Sync(); err != nil {
				return fmt.Errorf("同步文件到磁盘失败: %w", err)
			}
		}
	}

	return nil
}
