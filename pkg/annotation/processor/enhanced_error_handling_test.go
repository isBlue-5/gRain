package processor

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessorError 测试处理器错误
func TestProcessorError(t *testing.T) {
	t.Run("基本错误创建", func(t *testing.T) {
		err := NewProcessorError("TEST_ERROR", "测试错误", ErrorLevelError)

		assert.Equal(t, "TEST_ERROR", err.Code)
		assert.Equal(t, "测试错误", err.Message)
		assert.Equal(t, ErrorLevelError, err.Level)
		assert.NotZero(t, err.Timestamp)
		assert.NotEmpty(t, err.File)
		assert.NotZero(t, err.Line)
		assert.NotEmpty(t, err.Stack)
		assert.NotNil(t, err.Context)
	})

	t.Run("错误包装", func(t *testing.T) {
		originalErr := errors.New("原始错误")
		err := WrapError(originalErr, "WRAPPED_ERROR", "包装错误", ErrorLevelError)

		assert.Equal(t, "WRAPPED_ERROR", err.Code)
		assert.Equal(t, "包装错误", err.Message)
		assert.Equal(t, originalErr, err.Cause)
		assert.Equal(t, originalErr, errors.Unwrap(err))
	})

	t.Run("添加上下文", func(t *testing.T) {
		err := NewProcessorError("TEST_ERROR", "测试错误", ErrorLevelError)
		err.WithContext("file", "test.go").WithContext("line", 123)

		assert.Equal(t, "test.go", err.Context["file"])
		assert.Equal(t, 123, err.Context["line"])
	})

	t.Run("错误字符串表示", func(t *testing.T) {
		err := NewProcessorError("TEST_ERROR", "测试错误", ErrorLevelError)
		errStr := err.Error()

		assert.Contains(t, errStr, "ERROR")
		assert.Contains(t, errStr, "TEST_ERROR")
		assert.Contains(t, errStr, "测试错误")
	})
}

// TestErrorLevel 测试错误级别
func TestErrorLevel(t *testing.T) {
	testCases := []struct {
		level    ErrorLevel
		expected string
	}{
		{ErrorLevelInfo, "INFO"},
		{ErrorLevelWarn, "WARN"},
		{ErrorLevelError, "ERROR"},
		{ErrorLevelFatal, "FATAL"},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expected, tc.level.String())
	}
}

// TestDefaultLogger 测试默认日志记录器
func TestDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger(ErrorLevelWarn)

	// 测试不同级别的日志记录
	// 注意：这些测试主要检查不会panic，实际输出需要手动验证
	logger.Info("信息日志", String("key", "value"))
	logger.Warn("警告日志", Int("count", 10))
	logger.Error("错误日志", Error(errors.New("测试错误")))
	logger.Fatal("致命错误", Duration("duration", time.Second))
}

// TestErrorHandler 测试错误处理器
func TestErrorHandler(t *testing.T) {
	t.Run("基本错误处理", func(t *testing.T) {
		logger := NewDefaultLogger(ErrorLevelInfo)
		handler := NewErrorHandler(logger)

		// 处理普通错误
		err := errors.New("普通错误")
		handler.HandleError(err)

		assert.True(t, handler.HasErrors())
		assert.False(t, handler.HasFatalErrors())

		errors := handler.GetErrors()
		assert.Len(t, errors, 1)
		assert.Equal(t, "UNKNOWN_ERROR", errors[0].Code)
		assert.Equal(t, "普通错误", errors[0].Message)
	})

	t.Run("处理ProcessorError", func(t *testing.T) {
		logger := NewDefaultLogger(ErrorLevelInfo)
		handler := NewErrorHandler(logger)

		// 处理ProcessorError
		processorErr := NewProcessorError("TEST_ERROR", "测试错误", ErrorLevelError)
		handler.HandleError(processorErr)

		errors := handler.GetErrors()
		assert.Len(t, errors, 1)
		assert.Equal(t, "TEST_ERROR", errors[0].Code)
		assert.Equal(t, "测试错误", errors[0].Message)
	})

	t.Run("致命错误检测", func(t *testing.T) {
		logger := NewDefaultLogger(ErrorLevelInfo)
		handler := NewErrorHandler(logger)

		// 添加普通错误
		handler.HandleError(NewError("ERROR1", "错误1"))
		assert.False(t, handler.HasFatalErrors())

		// 添加致命错误
		handler.HandleError(NewFatalError("FATAL1", "致命错误1"))
		assert.True(t, handler.HasFatalErrors())
	})

	t.Run("清除错误", func(t *testing.T) {
		logger := NewDefaultLogger(ErrorLevelInfo)
		handler := NewErrorHandler(logger)

		handler.HandleError(errors.New("错误1"))
		handler.HandleError(errors.New("错误2"))
		assert.True(t, handler.HasErrors())

		handler.Clear()
		assert.False(t, handler.HasErrors())
		assert.Len(t, handler.GetErrors(), 0)
	})

	t.Run("生成错误报告", func(t *testing.T) {
		logger := NewDefaultLogger(ErrorLevelInfo)
		handler := NewErrorHandler(logger)

		// 没有错误时
		report := handler.GenerateErrorReport()
		assert.Contains(t, report, "No errors recorded")

		// 有错误时
		err := NewProcessorError("TEST_ERROR", "测试错误", ErrorLevelError)
		err.WithContext("test_key", "test_value")
		handler.HandleError(err)

		report = handler.GenerateErrorReport()
		assert.Contains(t, report, "Error Report")
		assert.Contains(t, report, "TEST_ERROR")
		assert.Contains(t, report, "测试错误")
		assert.Contains(t, report, "test_key")
		assert.Contains(t, report, "test_value")
	})
}

// TestField 测试日志字段
func TestField(t *testing.T) {
	stringField := String("name", "test")
	assert.Equal(t, "name", stringField.Key)
	assert.Equal(t, "test", stringField.Value)

	intField := Int("count", 42)
	assert.Equal(t, "count", intField.Key)
	assert.Equal(t, 42, intField.Value)

	errField := Error(errors.New("test error"))
	assert.Equal(t, "error", errField.Key)
	assert.Error(t, errField.Value.(error))

	durationField := Duration("time", time.Second)
	assert.Equal(t, "time", durationField.Key)
	assert.Equal(t, time.Second, durationField.Value)
}

// TestConvenienceFunctions 测试便捷函数
func TestConvenienceFunctions(t *testing.T) {
	// 清除全局错误处理器
	GlobalErrorHandler.Clear()

	t.Run("便捷错误创建", func(t *testing.T) {
		infoErr := NewInfoError("INFO_TEST", "信息错误")
		assert.Equal(t, ErrorLevelInfo, infoErr.Level)

		warnErr := NewWarnError("WARN_TEST", "警告错误")
		assert.Equal(t, ErrorLevelWarn, warnErr.Level)

		err := NewError("ERROR_TEST", "普通错误")
		assert.Equal(t, ErrorLevelError, err.Level)

		fatalErr := NewFatalError("FATAL_TEST", "致命错误")
		assert.Equal(t, ErrorLevelFatal, fatalErr.Level)
	})

	t.Run("全局错误处理", func(t *testing.T) {
		err := errors.New("全局测试错误")
		HandleError(err)

		assert.True(t, GlobalErrorHandler.HasErrors())

		errors := GlobalErrorHandler.GetErrors()
		assert.Len(t, errors, 1)
		assert.Contains(t, errors[0].Message, "全局测试错误")
	})
}

// TestSafeExecute 测试安全执行
func TestSafeExecute(t *testing.T) {
	// 清除全局错误处理器
	GlobalErrorHandler.Clear()

	t.Run("成功执行", func(t *testing.T) {
		err := SafeExecute("test_operation", func() error {
			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("执行失败", func(t *testing.T) {
		testError := errors.New("测试错误")
		err := SafeExecute("test_operation", func() error {
			return testError
		})

		assert.Error(t, err)
		assert.True(t, GlobalErrorHandler.HasErrors())
	})
}

// TestRecoverableExecute 测试可恢复执行
func TestRecoverableExecute(t *testing.T) {
	// 清除全局错误处理器
	GlobalErrorHandler.Clear()

	t.Run("正常执行", func(t *testing.T) {
		err := RecoverableExecute("normal_operation", func() error {
			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("普通错误", func(t *testing.T) {
		testError := errors.New("普通错误")
		err := RecoverableExecute("error_operation", func() error {
			return testError
		})

		assert.Error(t, err)
		assert.Equal(t, testError, err)
	})

	t.Run("panic恢复", func(t *testing.T) {
		err := RecoverableExecute("panic_operation", func() error {
			panic("测试panic")
		})

		assert.Error(t, err)
		require.IsType(t, &ProcessorError{}, err)

		processorErr := err.(*ProcessorError)
		assert.Equal(t, "PANIC_RECOVERED", processorErr.Code)
		assert.Contains(t, processorErr.Message, "测试panic")
		assert.Equal(t, ErrorLevelFatal, processorErr.Level)

		assert.True(t, GlobalErrorHandler.HasFatalErrors())
	})
}

// TestLogFunctions 测试日志函数
func TestLogFunctions(t *testing.T) {
	// 这些测试主要确保函数不会panic
	// 实际的日志输出需要手动验证

	LogInfo("测试信息", String("key", "value"))
	LogWarn("测试警告", Int("count", 10))
	LogError("测试错误", Error(errors.New("测试")))
	LogFatal("测试致命错误", Duration("time", time.Second))
}

// TestErrorHandlerWithNilLogger 测试带nil logger的错误处理器
func TestErrorHandlerWithNilLogger(t *testing.T) {
	// 传入nil logger应该使用默认logger
	handler := NewErrorHandler(nil)
	require.NotNil(t, handler)

	// 应该能正常处理错误而不panic
	handler.HandleError(errors.New("测试错误"))
	assert.True(t, handler.HasErrors())
}

// TestErrorContextOperations 测试错误上下文操作
func TestErrorContextOperations(t *testing.T) {
	err := NewProcessorError("TEST", "测试", ErrorLevelError)

	// 链式调用
	err.WithContext("key1", "value1").
		WithContext("key2", 42).
		WithContext("key3", true)

	assert.Equal(t, "value1", err.Context["key1"])
	assert.Equal(t, 42, err.Context["key2"])
	assert.Equal(t, true, err.Context["key3"])
}

// TestStackTrace 测试堆栈跟踪
func TestStackTrace(t *testing.T) {
	err := NewProcessorError("STACK_TEST", "堆栈测试", ErrorLevelError)

	assert.NotEmpty(t, err.Stack)
	assert.Contains(t, err.Stack, "TestStackTrace")
}

// TestErrorLevelComparison 测试错误级别比较
func TestErrorLevelComparison(t *testing.T) {
	assert.True(t, ErrorLevelInfo < ErrorLevelWarn)
	assert.True(t, ErrorLevelWarn < ErrorLevelError)
	assert.True(t, ErrorLevelError < ErrorLevelFatal)
}

// BenchmarkErrorCreation 错误创建性能基准测试
func BenchmarkErrorCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewProcessorError("TEST_ERROR", "测试错误", ErrorLevelError)
	}
}

// BenchmarkErrorHandling 错误处理性能基准测试
func BenchmarkErrorHandling(b *testing.B) {
	logger := NewDefaultLogger(ErrorLevelError)
	handler := NewErrorHandler(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := NewProcessorError("TEST_ERROR", "测试错误", ErrorLevelError)
		handler.HandleError(err)
	}
}
