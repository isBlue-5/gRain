package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/grain-framework/grain/pkg/web/render"
)

// ContentNegotiationOptions 内容协商选项
type ContentNegotiationOptions struct {
	// 默认内容类型
	DefaultContentType string

	// 支持的内容类型
	SupportedTypes []string

	// 是否强制使用默认内容类型
	ForceDefaultType bool
}

// ContentNegotiationOption 内容协商选项函数
type ContentNegotiationOption func(*ContentNegotiationOptions)

// WithDefaultContentType 设置默认内容类型
func WithDefaultContentType(contentType string) ContentNegotiationOption {
	return func(o *ContentNegotiationOptions) {
		o.DefaultContentType = contentType
	}
}

// WithSupportedTypes 设置支持的内容类型
func WithSupportedTypes(types ...string) ContentNegotiationOption {
	return func(o *ContentNegotiationOptions) {
		o.SupportedTypes = types
	}
}

// WithForceDefaultType 设置是否强制使用默认内容类型
func WithForceDefaultType(force bool) ContentNegotiationOption {
	return func(o *ContentNegotiationOptions) {
		o.ForceDefaultType = force
	}
}

// defaultContentNegotiationOptions 默认内容协商选项
func defaultContentNegotiationOptions() *ContentNegotiationOptions {
	return &ContentNegotiationOptions{
		DefaultContentType: "application/json",
		SupportedTypes:     []string{"application/json", "application/xml", "application/yaml"},
		ForceDefaultType:   false,
	}
}

// ContentNegotiation 内容协商中间件
// 根据请求的Accept头，设置适当的响应格式
func ContentNegotiation(opts ...ContentNegotiationOption) gin.HandlerFunc {
	// 应用选项
	options := defaultContentNegotiationOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *gin.Context) {
		// 如果强制使用默认类型，直接设置
		if options.ForceDefaultType {
			c.Set("responseFormat", options.DefaultContentType)
			c.Next()
			return
		}

		// 获取Accept头
		acceptHeader := c.GetHeader("Accept")
		if acceptHeader == "" {
			c.Set("responseFormat", options.DefaultContentType)
			c.Next()
			return
		}

		// 解析Accept头
		acceptTypes := strings.Split(acceptHeader, ",")
		selectedType := ""

		// 寻找第一个匹配的类型
		for _, acceptType := range acceptTypes {
			// 去除权重部分和空格
			mediaType := strings.TrimSpace(strings.Split(acceptType, ";")[0])

			// 检查是否在支持的类型中
			for _, supportedType := range options.SupportedTypes {
				if mediaType == supportedType || mediaType == "*/*" {
					selectedType = supportedType
					break
				}
			}

			if selectedType != "" {
				break
			}
		}

		// 如果没有匹配的类型，使用默认类型
		if selectedType == "" {
			selectedType = options.DefaultContentType
		}

		// 设置到上下文
		c.Set("responseFormat", selectedType)
		c.Next()
	}
}

// NegotiatedResponse 根据内容协商结果返回响应
func NegotiatedResponse(c *gin.Context, code int, obj interface{}) {
	// 获取响应格式
	format, exists := c.Get("responseFormat")
	if !exists {
		c.JSON(code, obj)
		return
	}

	// 根据格式选择响应方式
	switch format {
	case "application/json":
		c.JSON(code, obj)
	case "application/xml":
		c.XML(code, obj)
	case "application/yaml":
		c.YAML(code, obj)
	default:
		c.JSON(code, obj)
	}
}

// StandardResponseHandler 标准响应处理中间件
// 将所有响应包装成标准格式
func StandardResponseHandler() gin.HandlerFunc {
	responder := render.NewResponder()

	return func(c *gin.Context) {
		// 保存原始writer和状态码
		originalWriter := c.Writer
		c.Writer = &responseWriter{
			ResponseWriter: originalWriter,
			body:           make([]byte, 0),
			statusCode:     http.StatusOK,
		}

		// 处理请求
		c.Next()

		// 检查是否已经有响应
		w := c.Writer.(*responseWriter)
		if w.statusCode != http.StatusOK || len(w.body) == 0 {
			// 已经处理过响应或没有响应数据，不再包装
			return
		}

		// 重置响应
		c.Writer = originalWriter

		// 包装响应
		format, _ := c.Get("responseFormat")
		formatStr := "json"
		if format == "application/xml" {
			formatStr = "xml"
		} else if format == "application/yaml" {
			formatStr = "yaml"
		}

		// 创建标准响应
		resp := render.NewResponse(0, "success", string(w.body))
		responder.RespondWithFormat(c, http.StatusOK, resp, formatStr)
	}
}

// responseWriter 响应写入器
type responseWriter struct {
	gin.ResponseWriter
	body       []byte
	statusCode int
}

// Write 写入响应
func (w *responseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

// WriteString 写入字符串
func (w *responseWriter) WriteString(s string) (int, error) {
	w.body = append(w.body, []byte(s)...)
	return w.ResponseWriter.WriteString(s)
}

// WriteHeader 写入头部
func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}
