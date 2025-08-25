package processor

import (
	"testing"

	"github.com/grain-framework/grain/pkg/annotation/registry"
	"github.com/grain-framework/grain/pkg/annotation/types"
	"github.com/stretchr/testify/assert"
)

// TestBasicFunctionality 测试基本功能
func TestBasicFunctionality(t *testing.T) {
	// 测试注解解析器创建
	parser := NewAnnotationParser("@frame:")
	assert.NotNil(t, parser)

	// 测试注册表创建
	reg := registry.NewRegistry()
	assert.NotNil(t, reg)

	// 测试生成器创建
	generator := NewGenerator(parser, reg, "/tmp", "@frame:")
	assert.NotNil(t, generator)

	// 测试实体处理器创建
	entityProcessor := NewEntityProcessor(reg, parser, "/tmp")
	assert.NotNil(t, entityProcessor)

	// 测试路由处理器创建
	routeProcessor := NewRouteProcessor(reg, parser, "/tmp")
	assert.NotNil(t, routeProcessor)

	// 测试服务处理器创建
	serviceProcessor := NewServiceProcessor(reg, parser, "/tmp")
	assert.NotNil(t, serviceProcessor)
}

// TestAnnotationParsing 测试注解解析
func TestAnnotationParsing(t *testing.T) {
	parser := NewAnnotationParser("@frame:")

	// 测试简单注解
	code := `// @frame:controller
type UserController struct{}`

	annotations, err := parser.ParseFile("test.go")
	assert.NoError(t, err)
	assert.Len(t, annotations, 1)
	assert.Equal(t, types.ControllerType, annotations[0].GetType())
}

// TestRegistryOperations 测试注册表操作
func TestRegistryOperations(t *testing.T) {
	reg := registry.NewRegistry()

	// 测试添加注解
	annotation := &types.CommentAnnotation{
		Type:       types.ControllerType,
		TargetName: "UserController",
	}

	reg.Register(annotation)

	// 测试获取注解
	annotations := reg.FindByType(types.ControllerType)
	assert.Len(t, annotations, 1)
	assert.Equal(t, "UserController", annotations[0].GetTargetName())
}

// TestProcessorInitialization 测试处理器初始化
func TestProcessorInitialization(t *testing.T) {
	parser := NewAnnotationParser("@frame:")
	reg := registry.NewRegistry()

	// 测试所有处理器的初始化
	processors := []interface{}{
		NewEntityProcessor(reg, parser, "/tmp"),
		NewRouteProcessor(reg, parser, "/tmp"),
		NewServiceProcessor(reg, parser, "/tmp"),
	}

	for i, processor := range processors {
		assert.NotNil(t, processor, "Processor %d should not be nil", i)
	}
}
