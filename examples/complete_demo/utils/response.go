package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SuccessResponse 成功响应
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "操作成功",
		Data:    data,
	})
}

// ErrorResponse 错误响应
func ErrorResponse(c *gin.Context, statusCode int, message string, err error) {
	var errorMsg string
	if err != nil {
		errorMsg = err.Error()
	}

	c.JSON(statusCode, Response{
		Success: false,
		Message: message,
		Error:   errorMsg,
	})
}

// BadRequestResponse 请求参数错误响应
func BadRequestResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, message, nil)
}

// UnauthorizedResponse 未授权响应
func UnauthorizedResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusUnauthorized, message, nil)
}

// ForbiddenResponse 禁止访问响应
func ForbiddenResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusForbidden, message, nil)
}

// NotFoundResponse 资源不存在响应
func NotFoundResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusNotFound, message, nil)
}

// InternalServerErrorResponse 服务器内部错误响应
func InternalServerErrorResponse(c *gin.Context, message string, err error) {
	ErrorResponse(c, http.StatusInternalServerError, message, err)
}

// ValidationErrorResponse 验证错误响应
func ValidationErrorResponse(c *gin.Context, errors interface{}) {
	c.JSON(http.StatusBadRequest, Response{
		Success: false,
		Message: "请求参数验证失败",
		Error:   "validation_error",
		Data:    errors,
	})
}

// PaginatedResponse 分页响应
func PaginatedResponse(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	totalPage := (total + int64(pageSize) - 1) / int64(pageSize)

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "获取数据成功",
		Data: gin.H{
			"data": data,
			"pagination": gin.H{
				"total":      total,
				"page":       page,
				"page_size":  pageSize,
				"total_page": totalPage,
			},
		},
	})
}

// CreatedResponse 创建成功响应
func CreatedResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Message: "创建成功",
		Data:    data,
	})
}

// UpdatedResponse 更新成功响应
func UpdatedResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "更新成功",
		Data:    data,
	})
}

// DeletedResponse 删除成功响应
func DeletedResponse(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "删除成功",
	})
}

// NoContentResponse 无内容响应
func NoContentResponse(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
