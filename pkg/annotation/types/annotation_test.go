package types_test

import (
	"testing"

	"github.com/isBlue-5/grain/pkg/annotation/types"
	"github.com/stretchr/testify/assert"
)

// 测试属性处理函数
func TestGetAttributeByName(t *testing.T) {
	// 准备测试数据
	attrs := []types.AnnotationAttribute{
		{Name: "name", Value: "test"},
		{Name: "age", Value: 30},
		{Name: "active", Value: true},
	}

	// 测试存在的属性
	value, found := types.GetAttributeByName(attrs, "name")
	assert.True(t, found, "应该找到属性")
	assert.Equal(t, "test", value, "属性值应该匹配")

	// 测试不存在的属性
	value, found = types.GetAttributeByName(attrs, "unknown")
	assert.False(t, found, "不应该找到不存在的属性")
	assert.Nil(t, value, "不存在的属性值应该是nil")
}

func TestGetStringAttribute(t *testing.T) {
	// 准备测试数据
	attrs := []types.AnnotationAttribute{
		{Name: "name", Value: "test"},
		{Name: "id", Value: 123},
	}

	// 测试字符串类型属性
	value := types.GetStringAttribute(attrs, "name", "default")
	assert.Equal(t, "test", value, "应该返回字符串值")

	// 测试类型转换失败时返回默认值
	value = types.GetStringAttribute(attrs, "id", "default")
	assert.Equal(t, "default", value, "非字符串类型应该返回默认值")

	// 测试不存在的属性返回默认值
	value = types.GetStringAttribute(attrs, "unknown", "default")
	assert.Equal(t, "default", value, "不存在的属性应该返回默认值")
}

func TestGetBoolAttribute(t *testing.T) {
	// 准备测试数据
	attrs := []types.AnnotationAttribute{
		{Name: "active", Value: true},
		{Name: "name", Value: "test"},
	}

	// 测试布尔类型属性
	value := types.GetBoolAttribute(attrs, "active", false)
	assert.True(t, value, "应该返回布尔值")

	// 测试类型转换失败时返回默认值
	value = types.GetBoolAttribute(attrs, "name", false)
	assert.False(t, value, "非布尔类型应该返回默认值")

	// 测试不存在的属性返回默认值
	value = types.GetBoolAttribute(attrs, "unknown", true)
	assert.True(t, value, "不存在的属性应该返回默认值")
}

func TestGetIntAttribute(t *testing.T) {
	// 准备测试数据
	attrs := []types.AnnotationAttribute{
		{Name: "count", Value: 42},
		{Name: "float", Value: 42.5},
		{Name: "int64", Value: int64(100)},
		{Name: "name", Value: "test"},
	}

	// 测试整数类型属性
	value := types.GetIntAttribute(attrs, "count", 0)
	assert.Equal(t, 42, value, "应该返回整数值")

	// 测试float转为int
	value = types.GetIntAttribute(attrs, "float", 0)
	assert.Equal(t, 42, value, "浮点数应该转为整数")

	// 测试int64转为int
	value = types.GetIntAttribute(attrs, "int64", 0)
	assert.Equal(t, 100, value, "int64应该转为int")

	// 测试类型转换失败时返回默认值
	value = types.GetIntAttribute(attrs, "name", 99)
	assert.Equal(t, 99, value, "非数字类型应该返回默认值")

	// 测试不存在的属性返回默认值
	value = types.GetIntAttribute(attrs, "unknown", 88)
	assert.Equal(t, 88, value, "不存在的属性应该返回默认值")
}

func TestGetStringSliceAttribute(t *testing.T) {
	// 准备测试数据
	attrs := []types.AnnotationAttribute{
		{Name: "tags", Value: []string{"tag1", "tag2"}},
		{Name: "mixed", Value: []interface{}{"item1", "item2"}},
		{Name: "name", Value: "test"},
	}

	// 测试字符串切片类型属性
	value, found := types.GetStringSliceAttribute(attrs, "tags")
	assert.True(t, found, "应该找到属性")
	assert.Equal(t, []string{"tag1", "tag2"}, value, "应该返回字符串切片")

	// 测试接口切片类型属性
	value, found = types.GetStringSliceAttribute(attrs, "mixed")
	assert.True(t, found, "应该找到属性")
	assert.Equal(t, []string{"item1", "item2"}, value, "应该返回转换后的字符串切片")

	// 测试类型转换失败时返回nil和false
	value, found = types.GetStringSliceAttribute(attrs, "name")
	assert.False(t, found, "非切片类型应该返回false")
	assert.Nil(t, value, "非切片类型应该返回nil")

	// 测试不存在的属性返回nil和false
	value, found = types.GetStringSliceAttribute(attrs, "unknown")
	assert.False(t, found, "不存在的属性应该返回false")
	assert.Nil(t, value, "不存在的属性应该返回nil")
}
