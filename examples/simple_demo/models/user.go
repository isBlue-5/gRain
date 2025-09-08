package models

import "time"

// User 用户模型
// 框架会自动解析这个结构体，生成Swagger文档
type User struct {
	// 用户ID，数据库主键
	ID uint `json:"id" example:"1"`

	// 用户名，必须唯一
	Username string `json:"username" binding:"required,min=3,max=50" example:"johndoe"`

	// 邮箱地址，必须有效
	Email string `json:"email" binding:"required,email" example:"john@example.com"`

	// 真实姓名
	RealName string `json:"realName" binding:"required,min=2,max=100" example:"John Doe"`

	// 年龄，0-150岁
	Age int `json:"age" binding:"gte=0,lte=150" example:"25"`

	// 用户角色
	Role string `json:"role" binding:"oneof=USER ADMIN MODERATOR" example:"USER"`

	// 用户状态
	Status int `json:"status" binding:"oneof=0 1" example:"1"`

	// 创建时间
	CreatedAt time.Time `json:"createdAt" example:"2024-01-01T00:00:00Z"`

	// 更新时间
	UpdatedAt time.Time `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	// 用户名
	Username string `json:"username" binding:"required,min=3,max=50" example:"johndoe"`

	// 邮箱
	Email string `json:"email" binding:"required,email" example:"john@example.com"`

	// 真实姓名
	RealName string `json:"realName" binding:"required,min=2,max=100" example:"John Doe"`

	// 年龄
	Age int `json:"age" binding:"gte=0,lte=150" example:"25"`

	// 角色
	Role string `json:"role" binding:"oneof=USER ADMIN MODERATOR" example:"USER"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	// 真实姓名
	RealName *string `json:"realName" binding:"omitempty,min=2,max=100" example:"John Doe"`

	// 年龄
	Age *int `json:"age" binding:"omitempty,gte=0,lte=150" example:"25"`

	// 角色
	Role *string `json:"role" binding:"omitempty,oneof=USER ADMIN MODERATOR" example:"USER"`

	// 状态
	Status *int `json:"status" binding:"omitempty,oneof=0 1" example:"1"`
}

// UserResponse 用户响应
type UserResponse struct {
	// 用户ID
	ID uint `json:"id" example:"1"`

	// 用户名
	Username string `json:"username" example:"johndoe"`

	// 邮箱
	Email string `json:"email" example:"john@example.com"`

	// 真实姓名
	RealName string `json:"realName" example:"John Doe"`

	// 年龄
	Age int `json:"age" example:"25"`

	// 角色
	Role string `json:"role" example:"USER"`

	// 状态
	Status int `json:"status" example:"1"`

	// 创建时间
	CreatedAt time.Time `json:"createdAt" example:"2024-01-01T00:00:00Z"`

	// 更新时间
	UpdatedAt time.Time `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	// 错误代码
	Code string `json:"code" example:"USER_NOT_FOUND"`

	// 错误消息
	Message string `json:"message" example:"用户不存在"`

	// 错误详情
	Details map[string]interface{} `json:"details,omitempty" example:"{\"userId\": \"123\"}"`
}

// ValidationError 验证错误
type ValidationError struct {
	// 错误代码
	Code string `json:"code" example:"VALIDATION_ERROR"`

	// 错误消息
	Message string `json:"message" example:"请求参数验证失败"`

	// 字段错误
	FieldErrors []FieldError `json:"fieldErrors" example:"[{\"field\": \"username\", \"message\": \"用户名不能为空\"}]"`
}

// FieldError 字段错误
type FieldError struct {
	// 字段名
	Field string `json:"field" example:"username"`

	// 错误消息
	Message string `json:"message" example:"用户名不能为空"`

	// 字段值
	Value interface{} `json:"value,omitempty" example:"\"\""`
}

// ConflictError 冲突错误
type ConflictError struct {
	// 错误代码
	Code string `json:"code" example:"USER_ALREAD_EXISTS"`

	// 错误消息
	Message string `json:"message" example:"用户已存在"`

	// 冲突字段
	ConflictField string `json:"conflictField" example:"username"`

	// 冲突值
	ConflictValue string `json:"conflictValue" example:"johndoe"`
}

// SuccessResponse 成功响应
type SuccessResponse struct {
	// 成功消息
	Message string `json:"message" example:"操作成功"`

	// 响应数据
	Data interface{} `json:"data,omitempty"`
}
