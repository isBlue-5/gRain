package processor

import (
	"testing"

	"github.com/grain-framework/grain/pkg/annotation/registry"
	"github.com/stretchr/testify/assert"
)

func TestSwaggerGenerator_GenerateSwaggerSpec(t *testing.T) {
	// 创建注册表
	reg := registry.NewRegistry()

	// 创建Swagger生成器
	generator := NewSwaggerGenerator(reg, ".")

	// 生成Swagger规范
	spec, err := generator.GenerateSwaggerSpec()

	// 验证生成成功
	assert.NoError(t, err)
	assert.NotNil(t, spec)

	// 验证基本结构
	assert.Equal(t, "3.0.0", spec.OpenAPI)
	assert.NotNil(t, spec.Info)
	assert.Equal(t, "gRain Framework API", spec.Info.Title)
	assert.Equal(t, "1.0.0", spec.Info.Version)

	// 验证服务器配置
	assert.Len(t, spec.Servers, 1)
	assert.Equal(t, "http://localhost:8080", spec.Servers[0].URL)

	// 验证组件
	assert.NotNil(t, spec.Components)
}

func TestSwaggerGenerator_BuildFullPath(t *testing.T) {
	generator := &SwaggerGenerator{}

	tests := []struct {
		name     string
		prefix   string
		path     string
		expected string
	}{
		{
			name:     "no prefix",
			prefix:   "",
			path:     "/users",
			expected: "/users",
		},
		{
			name:     "with prefix",
			prefix:   "/api",
			path:     "/users",
			expected: "/api/users",
		},
		{
			name:     "prefix with trailing slash",
			prefix:   "/api/",
			path:     "/users",
			expected: "/api/users",
		},
		{
			name:     "path with leading slash",
			prefix:   "/api",
			path:     "/users",
			expected: "/api/users",
		},
		{
			name:     "both with slashes",
			prefix:   "/api/",
			path:     "/users",
			expected: "/api/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.buildFullPath(tt.prefix, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSwaggerGenerator_BuildDefaultResponses(t *testing.T) {
	generator := &SwaggerGenerator{}

	responses := generator.buildDefaultResponses()

	// 验证默认响应
	assert.NotNil(t, responses["200"])
	assert.Equal(t, "成功", responses["200"].Description)

	assert.NotNil(t, responses["400"])
	assert.Equal(t, "请求参数错误", responses["400"].Description)

	assert.NotNil(t, responses["500"])
	assert.Equal(t, "服务器内部错误", responses["500"].Description)
}

func TestSwaggerGenerator_BuildSecuritySchemes(t *testing.T) {
	generator := &SwaggerGenerator{}

	schemes := generator.buildSecuritySchemes()

	// 验证安全方案
	assert.NotNil(t, schemes["bearerAuth"])
	assert.Equal(t, "http", schemes["bearerAuth"].Type)
	assert.Equal(t, "Bearer token认证", schemes["bearerAuth"].Description)
}

func TestSwaggerGenerator_SetPathMethod(t *testing.T) {
	generator := &SwaggerGenerator{}
	pathItem := &OpenAPIPathItem{}

	operation := &OpenAPIOperation{
		Summary: "Test Operation",
	}

	// 测试设置GET方法
	generator.setPathMethod(pathItem, "GET", operation)
	assert.Equal(t, operation, pathItem.Get)

	// 测试设置POST方法
	generator.setPathMethod(pathItem, "POST", operation)
	assert.Equal(t, operation, pathItem.Post)

	// 测试设置PUT方法
	generator.setPathMethod(pathItem, "PUT", operation)
	assert.Equal(t, operation, pathItem.Put)

	// 测试设置DELETE方法
	generator.setPathMethod(pathItem, "DELETE", operation)
	assert.Equal(t, operation, pathItem.Delete)

	// 测试设置PATCH方法
	generator.setPathMethod(pathItem, "PATCH", operation)
	assert.Equal(t, operation, pathItem.Patch)
}

func TestSwaggerGenerator_BuildOperation(t *testing.T) {
	generator := &SwaggerGenerator{}

	route := &RouteInfo{
		Method:     "GET",
		Path:       "/users",
		Summary:    "获取用户列表",
		FuncName:   "GetUsers",
		Controller: "UserController",
	}

	controller := &ControllerInfo{
		StructName: "UserController",
		PathPrefix: "/api",
	}

	operation := generator.buildOperation(route, controller)

	// 验证操作信息
	assert.NotNil(t, operation)
	assert.Equal(t, []string{"UserController"}, operation.Tags)
	assert.Equal(t, "获取用户列表", operation.Summary)
	assert.Equal(t, "GetUsers", operation.OperationID)
	assert.NotNil(t, operation.Responses)
}

func TestSwaggerGenerator_FindControllerName(t *testing.T) {
	generator := &SwaggerGenerator{}

	// 创建一个模拟的注解
	// 这里需要根据实际的注解接口来实现
	// 暂时跳过这个测试
	t.Skip("需要实现实际的注解接口")
}

func TestSwaggerGenerator_ExtractResponseType(t *testing.T) {
	generator := &SwaggerGenerator{}

	// 创建一个模拟的注解
	// 这里需要根据实际的注解接口来实现
	// 暂时跳过这个测试
	t.Skip("需要实现实际的注解接口")
}

func TestSwaggerGenerator_ExtractShortTypeName(t *testing.T) {
	generator := &SwaggerGenerator{}

	tests := []struct {
		name     string
		typeName string
		expected string
	}{
		{
			name:     "simple type",
			typeName: "User",
			expected: "User",
		},
		{
			name:     "package type",
			typeName: "models.User",
			expected: "User",
		},
		{
			name:     "nested package type",
			typeName: "internal.models.User",
			expected: "User",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.extractShortTypeName(tt.typeName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
