// Package middleware 提供Web中间件
package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// LogConfig 日志中间件配置
type LogConfig struct {
	// 是否启用
	Enabled bool

	// 是否记录请求体
	LogRequest bool

	// 是否记录响应体
	LogResponse bool

	// 请求体大小限制
	MaxRequestSize int

	// 响应体大小限制
	MaxResponseSize int

	// 是否显示颜色
	UseColors bool

	// 忽略的路径
	IgnorePaths []string
}

// DefaultLogConfig 默认日志配置
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Enabled:         true,
		LogRequest:      false,
		LogResponse:     false,
		MaxRequestSize:  2048, // 2KB
		MaxResponseSize: 2048, // 2KB
		UseColors:       true,
		IgnorePaths:     []string{"/health", "/metrics"},
	}
}

// LogOption 日志选项函数
type LogOption func(*LogConfig)

// WithRequestLog 启用请求体记录
func WithRequestLog(enabled bool) LogOption {
	return func(c *LogConfig) {
		c.LogRequest = enabled
	}
}

// WithResponseLog 启用响应体记录
func WithResponseLog(enabled bool) LogOption {
	return func(c *LogConfig) {
		c.LogResponse = enabled
	}
}

// WithMaxSize 设置最大记录大小
func WithMaxSize(reqSize, respSize int) LogOption {
	return func(c *LogConfig) {
		c.MaxRequestSize = reqSize
		c.MaxResponseSize = respSize
	}
}

// WithColors 设置是否使用颜色
func WithColors(useColors bool) LogOption {
	return func(c *LogConfig) {
		c.UseColors = useColors
	}
}

// WithIgnorePaths 设置忽略的路径
func WithIgnorePaths(paths ...string) LogOption {
	return func(c *LogConfig) {
		c.IgnorePaths = paths
	}
}

// ResponseWriter 响应写入器
type ResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 写入响应
func (w *ResponseWriter) Write(b []byte) (int, error) {
	// 写入缓冲区
	if w.body != nil {
		w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// WriteString 写入字符串响应
func (w *ResponseWriter) WriteString(s string) (int, error) {
	// 写入缓冲区
	if w.body != nil {
		w.body.WriteString(s)
	}
	return w.ResponseWriter.WriteString(s)
}

// Logger 日志中间件
func Logger(options ...LogOption) gin.HandlerFunc {
	// 应用选项
	config := DefaultLogConfig()
	for _, option := range options {
		option(config)
	}

	// 如果未启用，返回空中间件
	if !config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// 定义颜色函数
	var (
		green   = colorFunc(config.UseColors, "\033[97;42m%s\033[0m", "%s")
		white   = colorFunc(config.UseColors, "\033[90;47m%s\033[0m", "%s")
		yellow  = colorFunc(config.UseColors, "\033[90;43m%s\033[0m", "%s")
		red     = colorFunc(config.UseColors, "\033[97;41m%s\033[0m", "%s")
		blue    = colorFunc(config.UseColors, "\033[97;44m%s\033[0m", "%s")
		magenta = colorFunc(config.UseColors, "\033[97;45m%s\033[0m", "%s")
		reset   = colorFunc(config.UseColors, "\033[0m%s\033[0m", "%s")
	)

	// 返回中间件
	return func(c *gin.Context) {
		// 检查是否忽略此路径
		path := c.Request.URL.Path
		for _, ignorePath := range config.IgnorePaths {
			if path == ignorePath {
				c.Next()
				return
			}
		}

		// 开始时间
		start := time.Now()

		// 读取请求体
		var requestBody []byte
		if config.LogRequest && c.Request.Body != nil && c.Request.ContentLength > 0 {
			if c.Request.ContentLength <= int64(config.MaxRequestSize) {
				// 读取请求体
				body, err := io.ReadAll(c.Request.Body)
				if err == nil {
					requestBody = body
				}
				// 重置请求体
				c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			}
		}

		// 包装响应写入器以捕获响应体
		var responseBody *bytes.Buffer
		if config.LogResponse {
			responseBody = &bytes.Buffer{}
			c.Writer = &ResponseWriter{
				ResponseWriter: c.Writer,
				body:           responseBody,
			}
		}

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)

		// 获取状态码和方法
		statusCode := c.Writer.Status()
		method := c.Request.Method

		// 确定状态颜色
		var statusColor func(string) string
		switch {
		case statusCode >= 200 && statusCode < 300:
			statusColor = green
		case statusCode >= 300 && statusCode < 400:
			statusColor = white
		case statusCode >= 400 && statusCode < 500:
			statusColor = yellow
		default:
			statusColor = red
		}

		// 确定方法颜色
		var methodColor func(string) string
		switch method {
		case http.MethodGet:
			methodColor = blue
		case http.MethodPost:
			methodColor = green
		case http.MethodPut:
			methodColor = yellow
		case http.MethodDelete:
			methodColor = red
		case http.MethodPatch:
			methodColor = magenta
		default:
			methodColor = reset
		}

		// 构建日志信息
		logMsg := fmt.Sprintf("[GIN] %s | %s | %s | %s | %s",
			methodColor(fmt.Sprintf(" %s ", method)),
			statusColor(fmt.Sprintf(" %3d ", statusCode)),
			reset(latency.String()),
			reset(c.Request.URL.Path),
			c.ClientIP(),
		)

		// 记录请求体（如果有）
		if config.LogRequest && len(requestBody) > 0 {
			logMsg += fmt.Sprintf("\n[REQUEST] %s", string(requestBody))
		}

		// 记录响应体（如果有）
		if config.LogResponse && responseBody != nil && responseBody.Len() > 0 {
			// 限制响应体大小
			respBody := responseBody.Bytes()
			if len(respBody) > config.MaxResponseSize {
				respBody = append(respBody[:config.MaxResponseSize], []byte("... (truncated)")...)
			}
			logMsg += fmt.Sprintf("\n[RESPONSE] %s", string(respBody))
		}

		// 输出日志
		if statusCode >= 500 {
			fmt.Fprintln(os.Stderr, logMsg)
		} else {
			fmt.Fprintln(os.Stdout, logMsg)
		}
	}
}

// colorFunc 返回颜色函数
func colorFunc(useColor bool, colorFormat, plainFormat string) func(string) string {
	return func(s string) string {
		if useColor {
			return fmt.Sprintf(colorFormat, s)
		}
		return fmt.Sprintf(plainFormat, s)
	}
}
