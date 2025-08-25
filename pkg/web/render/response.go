// Package render 提供响应渲染和内容协商功能
package render

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	// 状态码
	Code int `json:"code" xml:"code"`

	// 响应消息
	Message string `json:"message,omitempty" xml:"message,omitempty"`

	// 响应数据
	Data interface{} `json:"data,omitempty" xml:"data,omitempty"`

	// 错误信息
	Error string `json:"error,omitempty" xml:"error,omitempty"`

	// 分页信息
	Pagination *Pagination `json:"pagination,omitempty" xml:"pagination,omitempty"`

	// 扩展字段
	Extensions map[string]interface{} `json:"extensions,omitempty" xml:"extensions,omitempty"`
}

// Pagination 分页信息
type Pagination struct {
	// 当前页码
	Page int `json:"page" xml:"page"`

	// 每页大小
	Size int `json:"size" xml:"size"`

	// 总记录数
	Total int64 `json:"total" xml:"total"`

	// 总页数
	Pages int `json:"pages" xml:"pages"`
}

// NewResponse 创建新响应
func NewResponse(code int, message string, data interface{}) *Response {
	return &Response{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewError 创建错误响应
func NewError(code int, message string, err error) *Response {
	resp := &Response{
		Code:    code,
		Message: message,
	}

	if err != nil {
		resp.Error = err.Error()
	}

	return resp
}

// WithPagination 添加分页信息
func (r *Response) WithPagination(page, size int, total int64) *Response {
	pages := 0
	if size > 0 {
		pages = int((total + int64(size) - 1) / int64(size))
	}

	r.Pagination = &Pagination{
		Page:  page,
		Size:  size,
		Total: total,
		Pages: pages,
	}

	return r
}

// WithExtension 添加扩展字段
func (r *Response) WithExtension(key string, value interface{}) *Response {
	if r.Extensions == nil {
		r.Extensions = make(map[string]interface{})
	}

	r.Extensions[key] = value

	return r
}

// Responder 响应处理器
type Responder struct {
	// 默认状态码
	DefaultSuccessCode int

	// 默认成功消息
	DefaultSuccessMessage string

	// 默认错误状态码
	DefaultErrorCode int

	// 默认错误消息
	DefaultErrorMessage string
}

// NewResponder 创建响应处理器
func NewResponder() *Responder {
	return &Responder{
		DefaultSuccessCode:    0,
		DefaultSuccessMessage: "success",
		DefaultErrorCode:      500,
		DefaultErrorMessage:   "internal server error",
	}
}

// Success 成功响应
func (r *Responder) Success(c *gin.Context, data interface{}) {
	resp := NewResponse(r.DefaultSuccessCode, r.DefaultSuccessMessage, data)
	r.Respond(c, http.StatusOK, resp)
}

// SuccessWithMessage 带消息的成功响应
func (r *Responder) SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	resp := NewResponse(r.DefaultSuccessCode, message, data)
	r.Respond(c, http.StatusOK, resp)
}

// Error 错误响应
func (r *Responder) Error(c *gin.Context, httpStatus int, err error) {
	resp := NewError(r.DefaultErrorCode, r.DefaultErrorMessage, err)
	r.Respond(c, httpStatus, resp)
}

// ErrorWithMessage 带消息的错误响应
func (r *Responder) ErrorWithMessage(c *gin.Context, httpStatus int, code int, message string, err error) {
	resp := NewError(code, message, err)
	r.Respond(c, httpStatus, resp)
}

// Respond 自定义响应
func (r *Responder) Respond(c *gin.Context, httpStatus int, resp *Response) {
	// 检查Accept头，决定响应格式
	acceptHeader := c.GetHeader("Accept")

	// 默认使用JSON
	if acceptHeader == "" || strings.Contains(acceptHeader, "application/json") {
		c.JSON(httpStatus, resp)
		return
	}

	// XML响应
	if strings.Contains(acceptHeader, "application/xml") || strings.Contains(acceptHeader, "text/xml") {
		c.XML(httpStatus, resp)
		return
	}

	// YAML响应
	if strings.Contains(acceptHeader, "application/yaml") || strings.Contains(acceptHeader, "text/yaml") {
		c.YAML(httpStatus, resp)
		return
	}

	// 默认使用JSON
	c.JSON(httpStatus, resp)
}

// RespondWithFormat 根据指定格式响应
func (r *Responder) RespondWithFormat(c *gin.Context, httpStatus int, resp *Response, format string) {
	switch strings.ToLower(format) {
	case "json":
		c.JSON(httpStatus, resp)
	case "xml":
		c.XML(httpStatus, resp)
	case "yaml":
		c.YAML(httpStatus, resp)
	case "string":
		data, err := json.Marshal(resp)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error marshaling response: %v", err)
			return
		}
		c.String(httpStatus, string(data))
	default:
		c.JSON(httpStatus, resp)
	}
}

// JSONPretty 返回格式化的JSON响应
func (r *Responder) JSONPretty(c *gin.Context, httpStatus int, resp *Response) {
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		c.String(http.StatusInternalServerError, "Error marshaling response: %v", err)
		return
	}

	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Status(httpStatus)
	c.Writer.Write(data)
}

// XMLPretty 返回格式化的XML响应
func (r *Responder) XMLPretty(c *gin.Context, httpStatus int, resp *Response) {
	data, err := xml.MarshalIndent(resp, "", "  ")
	if err != nil {
		c.String(http.StatusInternalServerError, "Error marshaling response: %v", err)
		return
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Status(httpStatus)
	c.Writer.Write([]byte(xml.Header))
	c.Writer.Write(data)
}

// DefaultResponder 默认响应处理器
var DefaultResponder = NewResponder()

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	DefaultResponder.Success(c, data)
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	DefaultResponder.SuccessWithMessage(c, message, data)
}

// Error 错误响应
func Error(c *gin.Context, httpStatus int, err error) {
	DefaultResponder.Error(c, httpStatus, err)
}

// ErrorWithMessage 带消息的错误响应
func ErrorWithMessage(c *gin.Context, httpStatus int, code int, message string, err error) {
	DefaultResponder.ErrorWithMessage(c, httpStatus, code, message, err)
}

// Respond 自定义响应
func Respond(c *gin.Context, httpStatus int, resp *Response) {
	DefaultResponder.Respond(c, httpStatus, resp)
}
