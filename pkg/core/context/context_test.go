package context

import (
	"context"
	"testing"
	"time"
)

// TestNewContext 测试创建新的上下文
func TestNewContext(t *testing.T) {
	// 从nil创建上下文
	ctx1 := NewContext(nil)
	if ctx1 == nil {
		t.Fatal("NewContext(nil) returned nil")
	}

	// 从父上下文创建
	parentCtx := context.Background()
	ctx2 := NewContext(parentCtx)
	if ctx2 == nil {
		t.Fatal("NewContext(parentCtx) returned nil")
	}
}

// TestBasicContextMethods 测试基本上下文方法
func TestBasicContextMethods(t *testing.T) {
	// 创建上下文
	ctx := NewContext(context.Background())

	// 测试Deadline方法
	_, hasDeadline := ctx.Deadline()
	if hasDeadline {
		t.Error("Expected no deadline, but got one")
	}

	// 测试Done方法
	done := ctx.Done()
	if done != nil {
		select {
		case <-done:
			t.Error("Context should not be done")
		default:
			// 预期行为
		}
	}

	// 测试Err方法
	if err := ctx.Err(); err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	// 测试Value方法
	testKey := "testKey"
	if value := ctx.Value(testKey); value != nil {
		t.Errorf("Expected nil value for key %s, but got: %v", testKey, value)
	}
}

// TestUserAndSessionMethods 测试用户和会话相关方法
func TestUserAndSessionMethods(t *testing.T) {
	// 创建上下文
	ctx := NewContext(context.Background())

	// 测试用户ID
	userId := "user123"
	ctx.SetUserID(userId)
	if got := ctx.GetUserID(); got != userId {
		t.Errorf("Expected UserID %s, got %s", userId, got)
	}

	// 测试会话ID
	sessionId := "session456"
	ctx.SetSessionID(sessionId)
	if got := ctx.GetSessionID(); got != sessionId {
		t.Errorf("Expected SessionID %s, got %s", sessionId, got)
	}
}

// TestRequestMetadata 测试请求元数据相关方法
func TestRequestMetadata(t *testing.T) {
	// 创建上下文
	ctx := NewContext(context.Background())

	// 测试请求ID
	requestId := "req789"
	ctx.SetRequestID(requestId)
	if got := ctx.GetRequestID(); got != requestId {
		t.Errorf("Expected RequestID %s, got %s", requestId, got)
	}

	// 测试跟踪ID
	traceId := "trace-abc"
	ctx.SetTraceID(traceId)
	if got := ctx.GetTraceID(); got != traceId {
		t.Errorf("Expected TraceID %s, got %s", traceId, got)
	}

	// 测试开始时间和耗时
	now := time.Now()
	ctx.SetStartTime(now)
	if got := ctx.GetStartTime(); !got.Equal(now) {
		t.Errorf("Expected StartTime %v, got %v", now, got)
	}

	time.Sleep(10 * time.Millisecond)
	elapsed := ctx.GetElapsedTime()
	if elapsed < 10*time.Millisecond {
		t.Errorf("Expected ElapsedTime >= 10ms, got %v", elapsed)
	}

	// 测试客户端IP
	ip := "192.168.1.1"
	ctx.SetClientIP(ip)
	if got := ctx.GetClientIP(); got != ip {
		t.Errorf("Expected ClientIP %s, got %s", ip, got)
	}

	// 测试用户代理
	userAgent := "Mozilla/5.0"
	ctx.SetUserAgent(userAgent)
	if got := ctx.GetUserAgent(); got != userAgent {
		t.Errorf("Expected UserAgent %s, got %s", userAgent, got)
	}

	// 测试请求路径和方法
	path := "/api/users"
	ctx.SetPath(path)
	if got := ctx.GetPath(); got != path {
		t.Errorf("Expected Path %s, got %s", path, got)
	}

	method := "POST"
	ctx.SetMethod(method)
	if got := ctx.GetMethod(); got != method {
		t.Errorf("Expected Method %s, got %s", method, got)
	}
}

// TestTransactionAndCustomValues 测试事务和自定义值
func TestTransactionAndCustomValues(t *testing.T) {
	// 创建上下文
	ctx := NewContext(context.Background())

	// 测试事务对象
	type mockTx struct{ id int }
	tx := &mockTx{id: 12345}
	ctx.SetTransaction(tx)
	if got := ctx.GetTransaction(); got != tx {
		t.Errorf("Expected Transaction %v, got %v", tx, got)
	}

	// 测试自定义键值
	ctx.Set("customKey", "customValue")
	if got, ok := ctx.Get("customKey"); !ok || got != "customValue" {
		t.Errorf("Expected custom value 'customValue', got %v", got)
	}

	// 测试类型安全的Getter
	ctx.Set("stringKey", "string")
	if got := ctx.GetString("stringKey"); got != "string" {
		t.Errorf("Expected 'string', got %s", got)
	}

	ctx.Set("intKey", 42)
	if got := ctx.GetInt("intKey"); got != 42 {
		t.Errorf("Expected 42, got %d", got)
	}

	ctx.Set("boolKey", true)
	if got := ctx.GetBool("boolKey"); !got {
		t.Errorf("Expected true, got %v", got)
	}

	// 测试不存在的键
	if got := ctx.GetString("nonExisting"); got != "" {
		t.Errorf("Expected empty string, got %s", got)
	}
	if got := ctx.GetInt("nonExisting"); got != 0 {
		t.Errorf("Expected 0, got %d", got)
	}
	if got := ctx.GetBool("nonExisting"); got {
		t.Errorf("Expected false, got %v", got)
	}
}

// TestContextExtensions 测试上下文扩展方法
func TestContextExtensions(t *testing.T) {
	// 创建上下文
	ctx := NewContext(context.Background())

	// 测试WithValue
	newCtx := ctx.WithValue("testKey", "testValue")
	if got, ok := newCtx.Get("testKey"); !ok || got != "testValue" {
		t.Errorf("Expected value 'testValue' for key 'testKey', got %v", got)
	}

	// 验证原上下文不受影响
	if got, ok := ctx.Get("testKey"); ok {
		t.Errorf("Original context should not have testKey, but got %v", got)
	}

	// 测试WithCancel
	cancelCtx, cancel := ctx.WithCancel()
	// 确保cancel函数被调用
	defer cancel()

	if cancelCtx.Err() != nil {
		t.Errorf("Expected nil error before cancel, got %v", cancelCtx.Err())
	}

	// 调用cancel后，上下文应该被取消
	cancel()
	// 可能需要等待一小段时间才能确保上下文被取消
	time.Sleep(10 * time.Millisecond)

	if cancelCtx.Err() == nil {
		t.Error("Expected error after cancel, but got nil")
	}

	// 测试WithTimeout
	timeoutCtx, cancel := ctx.WithTimeout(50 * time.Millisecond)
	defer cancel()

	if timeoutCtx.Err() != nil {
		t.Errorf("Expected nil error before timeout, got %v", timeoutCtx.Err())
	}

	// 等待超时
	time.Sleep(100 * time.Millisecond)

	if timeoutCtx.Err() == nil {
		t.Error("Expected error after timeout, but got nil")
	}

	// 测试ToContext
	stdCtx := ctx.ToContext()
	if stdCtx == nil {
		t.Error("Expected non-nil standard context from ToContext()")
	}
}
