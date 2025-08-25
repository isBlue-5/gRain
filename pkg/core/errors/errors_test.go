package errors

import (
	"database/sql"
	"fmt"
	"testing"
)

// TestNewError 测试创建新的应用错误
func TestNewError(t *testing.T) {
	err := NewError(CodeNotFound, "资源未找到")

	if err.code != CodeNotFound {
		t.Errorf("Expected error code %s, got %s", CodeNotFound, err.code)
	}

	if err.message != "资源未找到" {
		t.Errorf("Expected error message '资源未找到', got '%s'", err.message)
	}

	if err.Status() != StatusNotFound {
		t.Errorf("Expected HTTP status %d, got %d", StatusNotFound, err.Status())
	}

	if err.cause != nil {
		t.Errorf("Expected nil cause, got %v", err.cause)
	}

	// 测试 Error() 方法
	expectedMsg := "NOT_FOUND: 资源未找到"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error string '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestWrapError 测试包装错误
func TestWrapError(t *testing.T) {
	// 原始错误
	originalErr := fmt.Errorf("原始错误")

	// 包装错误
	err := Wrap(originalErr, CodeDatabaseError, "数据库操作失败")

	if err.code != CodeDatabaseError {
		t.Errorf("Expected error code %s, got %s", CodeDatabaseError, err.code)
	}

	if err.message != "数据库操作失败" {
		t.Errorf("Expected error message '数据库操作失败', got '%s'", err.message)
	}

	if err.cause != originalErr {
		t.Errorf("Expected cause to be originalErr, got %v", err.cause)
	}

	if err.Status() != StatusInternalServerError {
		t.Errorf("Expected HTTP status %d, got %d", StatusInternalServerError, err.Status())
	}

	// 测试 Error() 方法
	expectedMsg := "DATABASE_ERROR: 数据库操作失败: 原始错误"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error string '%s', got '%s'", expectedMsg, err.Error())
	}

	// 测试 Unwrap 方法
	if Unwrap(err) != originalErr {
		t.Errorf("Expected Unwrap(err) to be originalErr, got %v", Unwrap(err))
	}
}

// TestWrapWithStatus 测试带状态码的包装错误
func TestWrapWithStatus(t *testing.T) {
	originalErr := fmt.Errorf("请求参数错误")

	err := WrapWithStatus(originalErr, CodeInvalidArgument, "无效的用户ID", StatusBadRequest)

	if err.Status() != StatusBadRequest {
		t.Errorf("Expected HTTP status %d, got %d", StatusBadRequest, err.Status())
	}
}

// TestErrorChain 测试错误链
func TestErrorChain(t *testing.T) {
	// 创建错误链
	level1 := fmt.Errorf("level 1 error")
	level2 := Wrap(level1, CodeDatabaseError, "level 2 error")
	level3 := Wrap(level2, CodeInternal, "level 3 error")

	// 测试 Is 和 As
	if !Is(level3, level2) {
		t.Error("Expected Is(level3, level2) to be true")
	}

	if !Is(level3, level1) {
		t.Error("Expected Is(level3, level1) to be true")
	}

	var appErr *AppError
	if !As(level3, &appErr) {
		t.Error("Expected As(level3, &appErr) to be true")
	}

	if appErr.code != CodeInternal {
		t.Errorf("Expected appErr.code to be %s, got %s", CodeInternal, appErr.code)
	}
}

// TestErrorDetails 测试错误详情
func TestErrorDetails(t *testing.T) {
	// 创建错误并添加详情
	err := NewError(CodeValidationError, "验证失败")
	err.AddDetail("field", "username")
	err.AddDetail("reason", "too short")

	// 检查详情
	details := err.Details()
	if details == nil {
		t.Fatal("Expected non-nil details")
	}

	if field, ok := details["field"]; !ok || field != "username" {
		t.Errorf("Expected details[\"field\"] to be \"username\", got %v", field)
	}

	if reason, ok := details["reason"]; !ok || reason != "too short" {
		t.Errorf("Expected details[\"reason\"] to be \"too short\", got %v", reason)
	}

	// 测试 WithDetails 方法
	newDetails := map[string]interface{}{
		"code":   400,
		"params": []string{"id", "name"},
	}
	err.WithDetails(newDetails)

	if code, ok := err.Details()["code"]; !ok || code != 400 {
		t.Errorf("Expected details[\"code\"] to be 400, got %v", code)
	}
}

// TestIsAppError 测试错误类型检查
func TestIsAppError(t *testing.T) {
	// AppError 类型
	appErr := NewError(CodeNotFound, "资源未找到")
	if !IsAppError(appErr) {
		t.Error("Expected IsAppError(appErr) to be true")
	}

	// 非 AppError 类型
	stdErr := fmt.Errorf("standard error")
	if IsAppError(stdErr) {
		t.Error("Expected IsAppError(stdErr) to be false")
	}

	// nil 错误
	if IsAppError(nil) {
		t.Error("Expected IsAppError(nil) to be false")
	}

	// 包装的 AppError
	wrappedErr := fmt.Errorf("wrapped: %w", appErr)
	if !IsAppError(wrappedErr) {
		t.Error("Expected IsAppError(wrappedErr) to be true")
	}
}

// TestGetAppError 测试获取应用错误
func TestGetAppError(t *testing.T) {
	// AppError 类型
	appErr := NewError(CodeNotFound, "资源未找到")
	extracted := GetAppError(appErr)
	if extracted != appErr {
		t.Errorf("Expected GetAppError(appErr) to be appErr, got %v", extracted)
	}

	// 非 AppError 类型
	stdErr := fmt.Errorf("standard error")
	if GetAppError(stdErr) != nil {
		t.Error("Expected GetAppError(stdErr) to be nil")
	}

	// nil 错误
	if GetAppError(nil) != nil {
		t.Error("Expected GetAppError(nil) to be nil")
	}

	// 包装的 AppError
	wrappedErr := fmt.Errorf("wrapped: %w", appErr)
	extracted = GetAppError(wrappedErr)
	if extracted != appErr {
		t.Errorf("Expected GetAppError(wrappedErr) to be appErr, got %v", extracted)
	}
}

// TestToMap 测试转换为Map
func TestToMap(t *testing.T) {
	// 创建错误
	err := NewError(CodeValidationError, "验证失败")
	err.AddDetail("field", "username")

	// 转换为Map
	m := err.ToMap()

	// 验证Map内容
	if m["code"] != CodeValidationError {
		t.Errorf("Expected m[\"code\"] to be %s, got %v", CodeValidationError, m["code"])
	}

	if m["message"] != "验证失败" {
		t.Errorf("Expected m[\"message\"] to be \"验证失败\", got %v", m["message"])
	}

	details, ok := m["details"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected m[\"details\"] to be a map")
	}

	if details["field"] != "username" {
		t.Errorf("Expected details[\"field\"] to be \"username\", got %v", details["field"])
	}
}

// TestCommonErrors 测试预定义的常用错误
func TestCommonErrors(t *testing.T) {
	// 测试 ErrNotFound
	if ErrNotFound.code != CodeNotFound {
		t.Errorf("Expected ErrNotFound.code to be %s, got %s", CodeNotFound, ErrNotFound.code)
	}

	// 测试 ErrInternal
	if ErrInternal.code != CodeInternal {
		t.Errorf("Expected ErrInternal.code to be %s, got %s", CodeInternal, ErrInternal.code)
	}

	// 测试 ErrUnauthorized
	if ErrUnauthorized.Status() != StatusUnauthorized {
		t.Errorf("Expected ErrUnauthorized.Status() to be %d, got %d", StatusUnauthorized, ErrUnauthorized.Status())
	}
}

// TestErrorStatusMapping 测试错误码到HTTP状态码的映射
func TestErrorStatusMapping(t *testing.T) {
	testCases := []struct {
		code   ErrorCode
		status int
	}{
		{CodeNotFound, StatusNotFound},
		{CodeInvalidArgument, StatusBadRequest},
		{CodeUnauthorized, StatusUnauthorized},
		{CodePermissionDenied, StatusForbidden},
		{CodeDuplicateKey, StatusConflict},
		{CodeServiceUnavailable, StatusServiceUnavailable},
	}

	for _, tc := range testCases {
		err := NewError(tc.code, "测试错误")
		if err.Status() != tc.status {
			t.Errorf("For code %s, expected status %d, got %d", tc.code, tc.status, err.Status())
		}
	}
}

// TestErrorWithStandardLibrary 测试与标准库错误的集成
func TestErrorWithStandardLibrary(t *testing.T) {
	// sql.ErrNoRows 集成
	dbErr := Wrap(sql.ErrNoRows, CodeNotFound, "找不到用户记录")

	// 应该能检测到底层是 sql.ErrNoRows
	if !Is(dbErr, sql.ErrNoRows) {
		t.Error("Expected Is(dbErr, sql.ErrNoRows) to be true")
	}

	// 检查状态码映射是否正确
	if dbErr.Status() != StatusNotFound {
		t.Errorf("Expected dbErr.Status() to be %d, got %d", StatusNotFound, dbErr.Status())
	}
}
