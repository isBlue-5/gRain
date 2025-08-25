// Package errors 提供框架的错误处理基础设施
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// 标准错误包的常用函数重新导出，以便于使用
var (
	New    = errors.New
	Unwrap = errors.Unwrap
	Is     = errors.Is
	As     = errors.As
)

// ErrorCode 定义错误类型标识符
type ErrorCode string

// HTTP状态码常量
const (
	StatusBadRequest          = http.StatusBadRequest          // 400
	StatusUnauthorized        = http.StatusUnauthorized        // 401
	StatusForbidden           = http.StatusForbidden           // 403
	StatusNotFound            = http.StatusNotFound            // 404
	StatusMethodNotAllowed    = http.StatusMethodNotAllowed    // 405
	StatusConflict            = http.StatusConflict            // 409
	StatusInternalServerError = http.StatusInternalServerError // 500
	StatusServiceUnavailable  = http.StatusServiceUnavailable  // 503
)

// 预定义错误码
const (
	// 通用错误
	CodeUnknown         ErrorCode = "UNKNOWN_ERROR"
	CodeInternal        ErrorCode = "INTERNAL_ERROR"
	CodeInvalidArgument ErrorCode = "INVALID_ARGUMENT"
	CodeNotFound        ErrorCode = "NOT_FOUND"
	CodeAlreadyExists   ErrorCode = "ALREADY_EXISTS"
	CodeTimeout         ErrorCode = "TIMEOUT"

	// 认证与授权错误
	CodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	CodePermissionDenied ErrorCode = "PERMISSION_DENIED"
	CodeSessionExpired   ErrorCode = "SESSION_EXPIRED"
	CodeInvalidToken     ErrorCode = "INVALID_TOKEN"

	// 数据库相关错误
	CodeDatabaseError    ErrorCode = "DATABASE_ERROR"
	CodeTransactionError ErrorCode = "TRANSACTION_ERROR"
	CodeDuplicateKey     ErrorCode = "DUPLICATE_KEY"

	// 验证错误
	CodeValidationError ErrorCode = "VALIDATION_ERROR"
	CodeRequired        ErrorCode = "REQUIRED_FIELD"
	CodeInvalidFormat   ErrorCode = "INVALID_FORMAT"

	// 网络错误
	CodeNetworkError       ErrorCode = "NETWORK_ERROR"
	CodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// AppError 应用错误类型，包含错误码、消息和原因
type AppError struct {
	code    ErrorCode              // 错误代码
	message string                 // 错误消息
	cause   error                  // 原始错误
	status  int                    // HTTP状态码
	details map[string]interface{} // 错误详情
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.code, e.message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.code, e.message)
}

// Unwrap 实现errors.Unwrap接口
func (e *AppError) Unwrap() error {
	return e.cause
}

// Code 返回错误码
func (e *AppError) Code() ErrorCode {
	return e.code
}

// Message 返回错误消息
func (e *AppError) Message() string {
	return e.message
}

// Cause 返回原始错误
func (e *AppError) Cause() error {
	return e.cause
}

// Status 返回HTTP状态码
func (e *AppError) Status() int {
	return e.status
}

// Details 返回错误详情
func (e *AppError) Details() map[string]interface{} {
	return e.details
}

// WithCause 设置原始错误
func (e *AppError) WithCause(cause error) *AppError {
	e.cause = cause
	return e
}

// WithStatus 设置HTTP状态码
func (e *AppError) WithStatus(status int) *AppError {
	e.status = status
	return e
}

// WithDetails 设置错误详情
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.details = details
	return e
}

// AddDetail 添加单个详情项
func (e *AppError) AddDetail(key string, value interface{}) *AppError {
	if e.details == nil {
		e.details = make(map[string]interface{})
	}
	e.details[key] = value
	return e
}

// ToMap 将错误转换为Map，用于API响应
func (e *AppError) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"code":    e.code,
		"message": e.message,
	}

	if len(e.details) > 0 {
		result["details"] = e.details
	}

	return result
}

// NewError 创建新的应用错误
func NewError(code ErrorCode, message string) *AppError {
	return &AppError{
		code:    code,
		message: message,
		status:  getDefaultStatusForCode(code),
	}
}

// Wrap 包装一个错误为应用错误
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		code:    code,
		message: message,
		cause:   err,
		status:  getDefaultStatusForCode(code),
	}
}

// WrapWithStatus 包装一个错误为应用错误，并指定HTTP状态码
func WrapWithStatus(err error, code ErrorCode, message string, status int) *AppError {
	return &AppError{
		code:    code,
		message: message,
		cause:   err,
		status:  status,
	}
}

// IsAppError 检查错误是否为AppError类型
func IsAppError(err error) bool {
	var appErr *AppError
	return err != nil && As(err, &appErr)
}

// GetAppError 尝试从错误链中提取AppError
func GetAppError(err error) *AppError {
	var appErr *AppError
	if err != nil && As(err, &appErr) {
		return appErr
	}
	return nil
}

// 根据错误码获取默认HTTP状态码
func getDefaultStatusForCode(code ErrorCode) int {
	switch code {
	case CodeUnknown, CodeInternal:
		return StatusInternalServerError
	case CodeInvalidArgument, CodeValidationError, CodeRequired, CodeInvalidFormat:
		return StatusBadRequest
	case CodeNotFound:
		return StatusNotFound
	case CodeAlreadyExists, CodeDuplicateKey:
		return StatusConflict
	case CodeUnauthorized, CodeSessionExpired, CodeInvalidToken:
		return StatusUnauthorized
	case CodePermissionDenied:
		return StatusForbidden
	case CodeServiceUnavailable:
		return StatusServiceUnavailable
	default:
		return StatusInternalServerError
	}
}

// 常用预定义错误
var (
	ErrInternal        = NewError(CodeInternal, "内部服务错误")
	ErrInvalidArgument = NewError(CodeInvalidArgument, "无效的参数")
	ErrNotFound        = NewError(CodeNotFound, "资源未找到")
	ErrAlreadyExists   = NewError(CodeAlreadyExists, "资源已存在")
	ErrUnauthorized    = NewError(CodeUnauthorized, "未授权的请求")
	ErrForbidden       = NewError(CodePermissionDenied, "权限不足")
	ErrTimeout         = NewError(CodeTimeout, "操作超时")
	ErrDatabaseError   = NewError(CodeDatabaseError, "数据库操作失败")
	ErrValidationError = NewError(CodeValidationError, "验证失败")
)
