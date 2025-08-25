// Package middleware provides HTTP middleware components
package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/grain-framework/grain/pkg/web/binding"
	"github.com/grain-framework/grain/pkg/web/render"
	"github.com/grain-framework/grain/pkg/web/validate"
)

// ErrInvalidRequestModel 请求模型无效错误
var ErrInvalidRequestModel = errors.New("invalid request model")

// RequestBindingOptions 请求绑定选项
type RequestBindingOptions struct {
	// 请求绑定源
	Source string

	// 禁止未知字段
	DisallowUnknownFields bool

	// 自动验证
	AutoValidate bool

	// 验证组
	ValidateGroups []string

	// 如果验证失败，是否中止请求处理
	AbortOnValidationFailure bool
}

// RequestBindingOption 请求绑定选项函数
type RequestBindingOption func(*RequestBindingOptions)

// WithSource 设置绑定源
func WithSource(source string) RequestBindingOption {
	return func(o *RequestBindingOptions) {
		o.Source = source
	}
}

// DisallowUnknownFields 禁止未知字段
func DisallowUnknownFields() RequestBindingOption {
	return func(o *RequestBindingOptions) {
		o.DisallowUnknownFields = true
	}
}

// WithAutoValidate 启用自动验证
func WithAutoValidate(autoValidate bool) RequestBindingOption {
	return func(o *RequestBindingOptions) {
		o.AutoValidate = autoValidate
	}
}

// WithValidateGroups 设置验证组
func WithValidateGroups(groups ...string) RequestBindingOption {
	return func(o *RequestBindingOptions) {
		o.ValidateGroups = groups
	}
}

// WithAbortOnValidationFailure 设置验证失败是否中止请求
func WithAbortOnValidationFailure(abort bool) RequestBindingOption {
	return func(o *RequestBindingOptions) {
		o.AbortOnValidationFailure = abort
	}
}

// defaultRequestBindingOptions 默认请求绑定选项
func defaultRequestBindingOptions() *RequestBindingOptions {
	return &RequestBindingOptions{
		Source:                   "json",
		DisallowUnknownFields:    false,
		AutoValidate:             true,
		ValidateGroups:           []string{},
		AbortOnValidationFailure: true,
	}
}

// RequestBinder 请求绑定中间件
// 用法：
//
//	router.POST("/users", RequestBinder(&UserRequest{}, WithSource("json"), WithAutoValidate(true)), controller.CreateUser)
func RequestBinder(model interface{}, opts ...RequestBindingOption) gin.HandlerFunc {
	// 应用选项
	options := defaultRequestBindingOptions()
	for _, opt := range opts {
		opt(options)
	}

	return func(c *gin.Context) {
		// 创建一个模型实例
		if model == nil {
			render.Error(c, http.StatusInternalServerError, ErrInvalidRequestModel)
			return
		}

		// 绑定请求
		bindOptions := []binding.BindingOption{}
		if options.DisallowUnknownFields {
			bindOptions = append(bindOptions, binding.DisallowUnknownFields())
		}

		// 执行绑定
		if err := binding.Bind(c, model, options.Source, bindOptions...); err != nil {
			render.Error(c, http.StatusBadRequest, err)
			if options.AbortOnValidationFailure {
				c.Abort()
			}
			return
		}

		// 验证
		if options.AutoValidate {
			validator := validate.New()
			if errs := validator.Validate(model); len(errs) > 0 {
				render.Error(c, http.StatusBadRequest, errs)
				if options.AbortOnValidationFailure {
					c.Abort()
				}
				return
			}
		}

		// 将模型存储到上下文中
		c.Set("requestModel", model)

		// 继续处理请求
		c.Next()
	}
}

// ExtractModel 从上下文中提取模型
// 用法：
//
//	func (c *UserController) CreateUser(ctx *gin.Context) {
//	    req := ExtractModel[UserRequest](ctx)
//	    if req == nil {
//	        // 处理错误
//	        return
//	    }
//	    // 使用请求模型
//	}
func ExtractModel[T any](c *gin.Context) *T {
	model, exists := c.Get("requestModel")
	if !exists {
		return nil
	}

	if m, ok := model.(*T); ok {
		return m
	}

	return nil
}
