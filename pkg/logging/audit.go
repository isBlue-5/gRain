package logging

import (
	"context"
	"time"
)

// AuditLogger 审计日志记录器
type AuditLogger struct {
	logger Logger
}

// NewAuditLogger 创建审计日志记录器
func NewAuditLogger(logger Logger) *AuditLogger {
	return &AuditLogger{
		logger: logger.With(Field{Key: "log_type", Value: "audit"}),
	}
}

// LogAction 记录审计行为
func (a *AuditLogger) LogAction(ctx context.Context, action string, resource string, resourceID string, outcome string, details map[string]interface{}) {
	// 提取用户信息
	userID := "anonymous"
	userName := "anonymous"

	if identity := ctx.Value("user_id"); identity != nil {
		if id, ok := identity.(string); ok {
			userID = id
		}
	}

	// 生成审计字段
	fields := []Field{
		{Key: "user_id", Value: userID},
		{Key: "user_name", Value: userName},
		{Key: "action", Value: action},
		{Key: "resource", Value: resource},
		{Key: "resource_id", Value: resourceID},
		{Key: "outcome", Value: outcome},
		{Key: "timestamp", Value: time.Now().Format(time.RFC3339)},
	}

	// 添加详细信息
	if details != nil {
		for k, v := range details {
			fields = append(fields, Field{Key: k, Value: v})
		}
	}

	// 记录审计日志
	a.logger.Info("Audit Event", fields...)
}

// LogLogin 记录登录审计
func (a *AuditLogger) LogLogin(ctx context.Context, userID string, username string, success bool, details map[string]interface{}) {
	outcome := "success"
	if !success {
		outcome = "failure"
	}

	fields := []Field{
		{Key: "user_id", Value: userID},
		{Key: "username", Value: username},
		{Key: "action", Value: "login"},
		{Key: "resource", Value: "auth"},
		{Key: "outcome", Value: outcome},
	}

	if details != nil {
		for k, v := range details {
			fields = append(fields, Field{Key: k, Value: v})
		}
	}

	a.logger.Info("Login Event", fields...)
}

// LogLogout 记录登出审计
func (a *AuditLogger) LogLogout(ctx context.Context, userID string, username string) {
	fields := []Field{
		{Key: "user_id", Value: userID},
		{Key: "username", Value: username},
		{Key: "action", Value: "logout"},
		{Key: "resource", Value: "auth"},
		{Key: "outcome", Value: "success"},
	}

	a.logger.Info("Logout Event", fields...)
}

// LogDataAccess 记录数据访问审计
func (a *AuditLogger) LogDataAccess(ctx context.Context, userID string, action string, resource string, resourceID string, success bool) {
	outcome := "success"
	if !success {
		outcome = "failure"
	}

	fields := []Field{
		{Key: "user_id", Value: userID},
		{Key: "action", Value: action},
		{Key: "resource", Value: resource},
		{Key: "resource_id", Value: resourceID},
		{Key: "outcome", Value: outcome},
	}

	a.logger.Info("Data Access Event", fields...)
}

// PerformanceLogger 性能日志记录器
type PerformanceLogger struct {
	logger           Logger
	thresholdMap     map[string]time.Duration
	defaultThreshold time.Duration
}

// NewPerformanceLogger 创建性能日志记录器
func NewPerformanceLogger(logger Logger, defaultThreshold time.Duration) *PerformanceLogger {
	return &PerformanceLogger{
		logger:           logger.With(Field{Key: "log_type", Value: "performance"}),
		thresholdMap:     make(map[string]time.Duration),
		defaultThreshold: defaultThreshold,
	}
}

// SetThreshold 设置特定操作的阈值
func (p *PerformanceLogger) SetThreshold(operation string, threshold time.Duration) {
	p.thresholdMap[operation] = threshold
}

// LogOperation 记录操作性能
func (p *PerformanceLogger) LogOperation(ctx context.Context, operation string, duration time.Duration, details map[string]interface{}) {
	// 获取操作阈值
	threshold, ok := p.thresholdMap[operation]
	if !ok {
		threshold = p.defaultThreshold
	}

	// 只记录超过阈值的操作
	if duration < threshold {
		return
	}

	// 生成性能日志字段
	fields := []Field{
		{Key: "operation", Value: operation},
		{Key: "duration", Value: duration.String()},
		{Key: "threshold", Value: threshold.String()},
	}

	// 添加详细信息
	if details != nil {
		for k, v := range details {
			fields = append(fields, Field{Key: k, Value: v})
		}
	}

	// 提取请求信息
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			fields = append(fields, Field{Key: "request_id", Value: id})
		}
	}

	// 记录性能日志
	p.logger.Warn("Slow Operation", fields...)
}

// LogDatabaseQuery 记录数据库查询性能
func (p *PerformanceLogger) LogDatabaseQuery(ctx context.Context, query string, duration time.Duration, rows int, success bool) {
	fields := []Field{
		{Key: "operation", Value: "database_query"},
		{Key: "query", Value: query},
		{Key: "duration", Value: duration.String()},
		{Key: "rows", Value: rows},
		{Key: "success", Value: success},
	}

	// 提取请求信息
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			fields = append(fields, Field{Key: "request_id", Value: id})
		}
	}

	// 根据成功状态决定日志级别
	if success {
		p.logger.Info("Database Query", fields...)
	} else {
		p.logger.Error("Database Query Failed", fields...)
	}
}

// LogHTTPRequest 记录HTTP请求性能
func (p *PerformanceLogger) LogHTTPRequest(ctx context.Context, method string, url string, duration time.Duration, statusCode int) {
	fields := []Field{
		{Key: "operation", Value: "http_request"},
		{Key: "method", Value: method},
		{Key: "url", Value: url},
		{Key: "duration", Value: duration.String()},
		{Key: "status_code", Value: statusCode},
	}

	// 提取请求信息
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			fields = append(fields, Field{Key: "request_id", Value: id})
		}
	}

	// 根据状态码决定日志级别
	switch {
	case statusCode >= 500:
		p.logger.Error("HTTP Request Failed", fields...)
	case statusCode >= 400:
		p.logger.Warn("HTTP Request Client Error", fields...)
	default:
		p.logger.Info("HTTP Request", fields...)
	}
}

// ErrorLogger 错误日志记录器
type ErrorLogger struct {
	logger Logger
}

// NewErrorLogger 创建错误日志记录器
func NewErrorLogger(logger Logger) *ErrorLogger {
	return &ErrorLogger{
		logger: logger.With(Field{Key: "log_type", Value: "error"}),
	}
}

// LogError 记录错误
func (e *ErrorLogger) LogError(ctx context.Context, err error, message string, details map[string]interface{}) {
	if err == nil {
		return
	}

	// 生成错误字段
	fields := []Field{
		{Key: "error", Value: err.Error()},
	}

	// 添加详细信息
	if details != nil {
		for k, v := range details {
			fields = append(fields, Field{Key: k, Value: v})
		}
	}

	// 提取请求信息
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			fields = append(fields, Field{Key: "request_id", Value: id})
		}
	}

	// 记录错误日志
	e.logger.Error(message, fields...)
}

// LogPanic 记录Panic
func (e *ErrorLogger) LogPanic(ctx context.Context, r interface{}, stack []byte) {
	fields := []Field{
		{Key: "panic", Value: r},
		{Key: "stack", Value: string(stack)},
	}

	// 提取请求信息
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			fields = append(fields, Field{Key: "request_id", Value: id})
		}
	}

	e.logger.Error("Panic Recovered", fields...)
}
