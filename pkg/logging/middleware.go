package logging

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"sync"

	"compress/gzip"

	"github.com/gin-gonic/gin"
)

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level       string   `json:"level" yaml:"level" env:"LOG_LEVEL" default:"info"`
	Format      string   `json:"format" yaml:"format" env:"LOG_FORMAT" default:"json"`
	TimeFormat  string   `json:"timeFormat" yaml:"timeFormat" env:"LOG_TIME_FORMAT" default:"2006-01-02T15:04:05.000Z07:00"`
	Outputs     []string `json:"outputs" yaml:"outputs" env:"LOG_OUTPUTS" default:"stdout"`
	ContextKeys []string `json:"contextKeys" yaml:"contextKeys" env:"LOG_CONTEXT_KEYS"`

	// 文件输出配置
	File struct {
		Path       string `json:"path" yaml:"path" env:"LOG_FILE_PATH" default:"logs/app.log"`
		MaxSize    int    `json:"maxSize" yaml:"maxSize" env:"LOG_FILE_MAX_SIZE" default:"100"`
		MaxBackups int    `json:"maxBackups" yaml:"maxBackups" env:"LOG_FILE_MAX_BACKUPS" default:"3"`
		MaxAge     int    `json:"maxAge" yaml:"maxAge" env:"LOG_FILE_MAX_AGE" default:"30"`
		Compress   bool   `json:"compress" yaml:"compress" env:"LOG_FILE_COMPRESS" default:"true"`
	} `json:"file" yaml:"file"`

	// 性能日志配置
	Performance struct {
		Enabled            bool          `json:"enabled" yaml:"enabled" env:"LOG_PERF_ENABLED" default:"true"`
		DefaultThreshold   time.Duration `json:"defaultThreshold" yaml:"defaultThreshold" env:"LOG_PERF_THRESHOLD" default:"200ms"`
		SlowQueryThreshold time.Duration `json:"slowQueryThreshold" yaml:"slowQueryThreshold" env:"LOG_SLOW_QUERY_THRESHOLD" default:"500ms"`
	} `json:"performance" yaml:"performance"`

	// 审计日志配置
	Audit struct {
		Enabled     bool   `json:"enabled" yaml:"enabled" env:"LOG_AUDIT_ENABLED" default:"true"`
		Destination string `json:"destination" yaml:"destination" env:"LOG_AUDIT_DESTINATION" default:"file"`
		Path        string `json:"path" yaml:"path" env:"LOG_AUDIT_PATH" default:"logs/audit.log"`
	} `json:"audit" yaml:"audit"`
}

// FileConfig 文件输出配置
type FileConfig struct {
	Path       string `json:"path" yaml:"path" env:"LOG_FILE_PATH" default:"logs/app.log"`
	MaxSize    int    `json:"maxSize" yaml:"maxSize" env:"LOG_FILE_MAX_SIZE" default:"100"`
	MaxBackups int    `json:"maxBackups" yaml:"maxBackups" env:"LOG_FILE_MAX_BACKUPS" default:"3"`
	MaxAge     int    `json:"maxAge" yaml:"maxAge" env:"LOG_FILE_MAX_AGE" default:"30"`
	Compress   bool   `json:"compress" yaml:"compress" env:"LOG_FILE_COMPRESS" default:"true"`
}

// 上下文键类型
type contextKey string

const (
	// 请求ID在上下文中的键
	requestIDKey contextKey = "request_id"
	// 跟踪ID在上下文中的键
	traceIDKey contextKey = "trace_id"
	// 用户ID在上下文中的键
	userIDKey contextKey = "user_id"
)

// LoggerMiddleware Gin日志中间件
func LoggerMiddleware(logger Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		start := time.Now()

		// 设置请求ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
			c.Header("X-Request-ID", requestID)
		}

		// 将日志记录器存入上下文
		contextLogger := logger.With(
			Field{Key: "request_id", Value: requestID},
			Field{Key: "client_ip", Value: c.ClientIP()},
		)
		c.Set("logger", contextLogger)

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)

		// 获取状态码和错误信息
		statusCode := c.Writer.Status()
		errorList := c.Errors.Errors()

		// 记录访问日志
		fields := []Field{
			{Key: "status", Value: statusCode},
			{Key: "latency", Value: latency.String()},
			{Key: "path", Value: c.Request.URL.Path},
			{Key: "method", Value: c.Request.Method},
			{Key: "size", Value: c.Writer.Size()},
		}

		if len(errorList) > 0 {
			fields = append(fields, Field{Key: "errors", Value: errorList})
		}

		// 根据状态码决定日志级别
		switch {
		case statusCode >= 500:
			contextLogger.Error("Server Error", fields...)
		case statusCode >= 400:
			contextLogger.Warn("Client Error", fields...)
		default:
			contextLogger.Info("Request Completed", fields...)
		}
	}
}

// RequestBodyLoggerMiddleware 请求体日志中间件
func RequestBodyLoggerMiddleware(logger Logger, maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 对于文件上传等二进制内容，不记录请求体
		contentType := c.ContentType()
		if strings.Contains(contentType, "multipart/form-data") {
			c.Next()
			return
		}

		// 读取请求体
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(io.LimitReader(c.Request.Body, maxSize))

			// 恢复请求体，以便后续处理
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 获取上下文日志记录器
		contextLogger, exists := c.Get("logger")
		if !exists {
			contextLogger = logger
		}

		// 记录请求体
		if len(bodyBytes) > 0 {
			l := contextLogger.(Logger)
			if int64(len(bodyBytes)) > maxSize {
				l.Debug("Request Body (truncated)",
					Field{Key: "body", Value: string(bodyBytes[:maxSize]) + "..."},
				)
			} else {
				l.Debug("Request Body",
					Field{Key: "body", Value: string(bodyBytes)},
				)
			}
		}

		c.Next()
	}
}

// LoggerFromGin 从Gin上下文创建记录器
func LoggerFromGin(c *gin.Context, baseLogger Logger) Logger {
	// 提取请求ID
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = generateRequestID()
		c.Header("X-Request-ID", requestID)
	}

	// 提取跟踪ID
	traceID := c.GetHeader("X-Trace-ID")

	// 创建带上下文的记录器
	logger := baseLogger.With(
		Field{Key: "request_id", Value: requestID},
		Field{Key: "path", Value: c.Request.URL.Path},
		Field{Key: "method", Value: c.Request.Method},
	)

	if traceID != "" {
		logger = logger.With(Field{Key: "trace_id", Value: traceID})
	}

	// 如果已认证，添加用户信息
	if identity, exists := c.Get("identity"); exists {
		if id, ok := identity.(string); ok {
			logger = logger.With(Field{Key: "user_id", Value: id})
		}
	}

	return logger
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// DefaultLoggingConfig 创建默认日志配置
func DefaultLoggingConfig() LoggingConfig {
	config := LoggingConfig{
		Level:       "info",
		Format:      "json",
		TimeFormat:  "2006-01-02T15:04:05.000Z07:00",
		Outputs:     []string{"stdout"},
		ContextKeys: []string{"request_id", "user_id", "trace_id"},
	}

	// 文件输出默认配置
	config.File.Path = "logs/app.log"
	config.File.MaxSize = 100
	config.File.MaxBackups = 3
	config.File.MaxAge = 30
	config.File.Compress = true

	// 性能日志默认配置
	config.Performance.Enabled = true
	config.Performance.DefaultThreshold = 200 * time.Millisecond
	config.Performance.SlowQueryThreshold = 500 * time.Millisecond

	// 审计日志默认配置
	config.Audit.Enabled = true
	config.Audit.Destination = "file"
	config.Audit.Path = "logs/audit.log"

	return config
}

// NewLoggerFromConfig 从配置创建日志记录器
func NewLoggerFromConfig(config LoggingConfig) (Logger, error) {
	// 解析日志级别
	level, err := ParseLevel(config.Level)
	if err != nil {
		return nil, err
	}

	// 创建格式化器
	var formatter Formatter
	switch strings.ToLower(config.Format) {
	case "json":
		formatter = &JSONFormatter{
			TimeFormat: config.TimeFormat,
		}
	case "text":
		formatter = &TextFormatter{
			TimeFormat: config.TimeFormat,
		}
	case "simple":
		formatter = &SimpleFormatter{
			TimeFormat: config.TimeFormat,
		}
	default:
		return nil, fmt.Errorf("unsupported log format: %s", config.Format)
	}

	// 创建日志选项
	options := []Option{
		WithLevel(level),
		WithFormatter(formatter),
		WithContextKeys(config.ContextKeys),
	}

	// 配置输出目标
	for _, output := range config.Outputs {
		switch output {
		case "stdout":
			options = append(options, WithOutput(os.Stdout))
		case "stderr":
			options = append(options, WithOutput(os.Stderr))
		case "file":
			// 添加文件输出支持
			if fileOutput, err := createFileOutput(config.File); err != nil {
				// 如果文件输出创建失败，记录错误并使用stdout作为后备
				fmt.Fprintf(os.Stderr, "警告: 创建文件输出失败: %v，使用stdout作为后备\n", err)
				options = append(options, WithOutput(os.Stdout))
			} else {
				options = append(options, WithOutput(fileOutput))
			}
		}
	}

	// 创建日志记录器
	return NewLogger(options...), nil
}

// createFileOutput 创建文件输出
func createFileOutput(config FileConfig) (io.Writer, error) {
	// 确保日志目录存在
	if err := os.MkdirAll(filepath.Dir(config.Path), 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 创建文件输出
	file, err := os.OpenFile(config.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}

	// 如果启用了日志轮转，使用轮转Writer
	if config.MaxSize > 0 {
		// 创建轮转Writer
		rotator := &RotatingWriter{
			file:        file,
			maxSize:     int64(config.MaxSize) * 1024 * 1024, // 转换为字节
			maxBackups:  config.MaxBackups,
			maxAge:      config.MaxAge,
			compress:    config.Compress,
			currentSize: 0,
		}

		// 获取当前文件大小
		if stat, err := file.Stat(); err == nil {
			rotator.currentSize = stat.Size()
		}

		return rotator, nil
	}

	return file, nil
}

// RotatingWriter 日志轮转Writer
type RotatingWriter struct {
	file        *os.File
	maxSize     int64
	maxBackups  int
	maxAge      int
	compress    bool
	currentSize int64
	mu          sync.Mutex
}

// Write 实现io.Writer接口
func (rw *RotatingWriter) Write(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// 检查是否需要轮转
	if rw.currentSize+int64(len(p)) > rw.maxSize {
		if err := rw.rotate(); err != nil {
			return 0, err
		}
	}

	// 写入数据
	n, err = rw.file.Write(p)
	if err == nil {
		rw.currentSize += int64(n)
	}

	return n, err
}

// rotate 执行日志轮转
func (rw *RotatingWriter) rotate() error {
	// 关闭当前文件
	if err := rw.file.Close(); err != nil {
		return fmt.Errorf("关闭日志文件失败: %w", err)
	}

	// 重命名当前文件
	backupPath := rw.file.Name() + "." + time.Now().Format("2006-01-02-15-04-05")
	if err := os.Rename(rw.file.Name(), backupPath); err != nil {
		return fmt.Errorf("重命名日志文件失败: %w", err)
	}

	// 如果启用了压缩，压缩备份文件
	if rw.compress {
		go func() {
			if err := compressFile(backupPath); err != nil {
				fmt.Fprintf(os.Stderr, "压缩日志文件失败: %v\n", err)
			}
		}()
	}

	// 清理旧备份文件
	go rw.cleanupOldBackups()

	// 创建新文件
	file, err := os.OpenFile(rw.file.Name(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("创建新日志文件失败: %w", err)
	}

	rw.file = file
	rw.currentSize = 0

	return nil
}

// cleanupOldBackups 清理旧的备份文件
func (rw *RotatingWriter) cleanupOldBackups() {
	dir := filepath.Dir(rw.file.Name())
	pattern := filepath.Base(rw.file.Name()) + ".*"

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return
	}

	// 按修改时间排序
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var files []fileInfo
	for _, match := range matches {
		if stat, err := os.Stat(match); err == nil {
			files = append(files, fileInfo{match, stat.ModTime()})
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	// 删除超过最大备份数量的文件
	if len(files) > rw.maxBackups {
		for _, file := range files[rw.maxBackups:] {
			os.Remove(file.path)
		}
	}

	// 删除超过最大年龄的文件
	if rw.maxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -rw.maxAge)
		for _, file := range files {
			if file.modTime.Before(cutoff) {
				os.Remove(file.path)
			}
		}
	}
}

// compressFile 压缩文件
func compressFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	compressedPath := filePath + ".gz"
	compressedFile, err := os.Create(compressedPath)
	if err != nil {
		return err
	}
	defer compressedFile.Close()

	gzipWriter := gzip.NewWriter(compressedFile)
	defer gzipWriter.Close()

	_, err = io.Copy(gzipWriter, file)
	if err != nil {
		return err
	}

	// 压缩成功后删除原文件
	return os.Remove(filePath)
}
