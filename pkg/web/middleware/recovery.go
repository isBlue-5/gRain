package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grain-framework/grain/pkg/core/errors"
	"github.com/grain-framework/grain/pkg/web/render"
)

// RecoveryConfig 恢复中间件配置
type RecoveryConfig struct {
	// 是否启用
	Enabled bool

	// 是否记录堆栈跟踪
	LogStack bool

	// 是否记录请求信息
	LogRequest bool

	// 自定义错误处理
	ErrorHandler func(*gin.Context, interface{})
}

// DefaultRecoveryConfig 默认恢复配置
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		Enabled:    true,
		LogStack:   true,
		LogRequest: true,
		ErrorHandler: func(c *gin.Context, err interface{}) {
			// 默认错误处理
			appErr := errors.NewError(errors.CodeInternal, "服务器内部错误")
			render.ErrorWithMessage(c, appErr.Status(), appErr.Status(), appErr.Error(), nil)
		},
	}
}

// RecoveryOption 恢复选项函数
type RecoveryOption func(*RecoveryConfig)

// WithLogStack 设置是否记录堆栈跟踪
func WithLogStack(logStack bool) RecoveryOption {
	return func(c *RecoveryConfig) {
		c.LogStack = logStack
	}
}

// WithLogRequest 设置是否记录请求信息
func WithLogRequest(logRequest bool) RecoveryOption {
	return func(c *RecoveryConfig) {
		c.LogRequest = logRequest
	}
}

// WithErrorHandler 设置自定义错误处理
func WithErrorHandler(handler func(*gin.Context, interface{})) RecoveryOption {
	return func(c *RecoveryConfig) {
		c.ErrorHandler = handler
	}
}

// Recovery 恢复中间件
func Recovery(options ...RecoveryOption) gin.HandlerFunc {
	// 应用选项
	config := DefaultRecoveryConfig()
	for _, option := range options {
		option(config)
	}

	// 如果未启用，返回空中间件
	if !config.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// 返回中间件
	return func(c *gin.Context) {
		// 恢复函数
		defer func() {
			if err := recover(); err != nil {
				// 检查是否已写入响应
				if c.Writer.Status() != http.StatusOK {
					return
				}

				// 记录时间
				start := time.Now()

				// 记录请求信息
				if config.LogRequest {
					httpRequest, _ := httputil.DumpRequest(c.Request, false)
					headers := c.Request.Header

					// 记录请求头
					fmt.Fprintf(os.Stderr, "[Recovery] %s panic recovered:\n", time.Now().Format("2006/01/02 - 15:04:05"))
					fmt.Fprintf(os.Stderr, "[Recovery] %s %s\n", c.Request.Method, c.Request.URL.Path)
					fmt.Fprintf(os.Stderr, "[Recovery] Headers:\n%v\n", headers)
					fmt.Fprintf(os.Stderr, "[Recovery] %q\n", httpRequest)
				}

				// 记录堆栈跟踪
				if config.LogStack {
					stack := stack(3)
					fmt.Fprintf(os.Stderr, "[Recovery] Stack Trace:\n%s\n", stack)
				}

				// 记录错误信息
				fmt.Fprintf(os.Stderr, "[Recovery] Error: %v\n", err)
				fmt.Fprintf(os.Stderr, "[Recovery] Elapsed: %v\n", time.Since(start))

				// 调用错误处理
				if config.ErrorHandler != nil {
					config.ErrorHandler(c, err)
				}

				// 中止处理
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()

		// 处理请求
		c.Next()
	}
}

// stack 返回堆栈跟踪
func stack(skip int) []byte {
	buf := new(bytes.Buffer)

	// 跟踪堆栈
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		// 格式化堆栈
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}

		// 显示源码
		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}

	return buf.Bytes()
}

// source 返回源代码
func source(lines [][]byte, n int) []byte {
	// 确保行号有效
	n--
	if n < 0 || n >= len(lines) {
		return []byte("???")
	}
	return bytes.TrimSpace(lines[n])
}

// function 返回函数名
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return []byte("???")
	}
	name := []byte(fn.Name())

	// 简化包名
	if i := bytes.LastIndex(name, []byte(".")); i >= 0 {
		name = name[i+1:]
	}
	return name
}
