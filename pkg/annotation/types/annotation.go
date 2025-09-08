// Package types 提供注解系统的基本类型定义
package types

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
)

// AnnotationType 注解类型
type AnnotationType string

// 注解类型常量
const (
	// 通用注解类型
	CustomType AnnotationType = "custom" // 自定义类型

	// 核心注解类型
	InjectType     AnnotationType = "inject"     // 依赖注入注解
	ControllerType AnnotationType = "controller" // 控制器注解
	RouteType      AnnotationType = "route"      // 路由注解
	ServiceType    AnnotationType = "service"    // 服务注解

	// 请求处理注解类型
	BindType     AnnotationType = "bind"     // 绑定注解
	ValidateType AnnotationType = "validate" // 验证注解

	// 横切关注点注解类型
	LogType         AnnotationType = "log"         // 日志注解
	TransactionType AnnotationType = "transaction" // 事务注解
	AuthType        AnnotationType = "auth"        // 认证授权注解
	CacheType       AnnotationType = "cacheable"   // 缓存注解
	RateLimitType   AnnotationType = "rateLimit"   // 速率限制注解

	// 持久化注解类型
	EntityType     AnnotationType = "entity"     // 实体注解
	TableType      AnnotationType = "table"      // 表注解
	ColumnType     AnnotationType = "column"     // 列注解
	IdType         AnnotationType = "id"         // 主键注解
	RepositoryType AnnotationType = "repository" // 仓库注解
	QueryType      AnnotationType = "query"      // 查询注解
)

// TargetType 定义注解可以应用的目标类型
type TargetType string

// 支持的注解目标类型
const (
	TypeTarget      TargetType = "type"      // 类型（结构体）
	MethodTarget    TargetType = "method"    // 方法
	FieldTarget     TargetType = "field"     // 字段
	ParameterTarget TargetType = "parameter" // 参数
	PackageTarget   TargetType = "package"   // 包
	FunctionTarget  TargetType = "function"  // 函数（非方法）
	VariableTarget  TargetType = "variable"  // 变量或常量
)

// Position 表示注解在源代码中的位置
type Position struct {
	Filename string // 文件名
	Line     int    // 行号
	Column   int    // 列号
}

// SourcePosition 创建源代码位置信息
func SourcePosition(fset *token.FileSet, pos token.Pos) Position {
	position := fset.Position(pos)
	return Position{
		Filename: position.Filename,
		Line:     position.Line,
		Column:   position.Column,
	}
}

// AnnotationAttribute 表示注解属性
type AnnotationAttribute struct {
	Name  string      // 属性名称
	Value interface{} // 属性值
}

// StructTagAnnotation 表示结构体标签注解
type StructTagAnnotation struct {
	Type        AnnotationType        // 注解类型
	Target      interface{}           // 注解目标
	TargetType  TargetType            // 目标类型
	TargetName  string                // 目标名称
	TagName     string                // 标签名称
	TagValue    string                // 标签值
	Attributes  []AnnotationAttribute // 注解属性
	Position    Position              // 源码位置
	StructField *ast.Field            // AST中的结构体字段
	StructType  *ast.StructType       // AST中的结构体类型
	StructName  string                // 结构体名称

	// 新增：类型信息
	FieldType types.Type // 字段的类型信息
}

// CommentAnnotation 表示注释注解
type CommentAnnotation struct {
	Type         AnnotationType        // 注解类型
	Target       interface{}           // 注解目标
	TargetType   TargetType            // 目标类型
	TargetName   string                // 目标名称
	Raw          string                // 原始注释文本
	Attributes   []AnnotationAttribute // 注解属性
	Position     Position              // 源码位置
	Comment      *ast.Comment          // AST中的注释
	Node         ast.Node              // 相关的AST节点
	ReceiverType string                // 方法接收者类型（仅对方法有效）

	// 新增：类型信息
	MethodSignature *types.Signature // 方法签名（仅对方法有效）
	TypeInfo        types.Type       // 相关的类型信息
}

// Annotation 通用注解接口
type Annotation interface {
	GetType() AnnotationType
	GetTargetType() TargetType
	GetTargetName() string
	GetAttributes() []AnnotationAttribute
	GetPosition() Position
}

// GetType 获取注解类型
func (a *StructTagAnnotation) GetType() AnnotationType {
	return a.Type
}

// GetTargetType 获取目标类型
func (a *StructTagAnnotation) GetTargetType() TargetType {
	return a.TargetType
}

// GetTargetName 获取目标名称
func (a *StructTagAnnotation) GetTargetName() string {
	return a.TargetName
}

// GetAttributes 获取属性列表
func (a *StructTagAnnotation) GetAttributes() []AnnotationAttribute {
	return a.Attributes
}

// GetPosition 获取源码位置
func (a *StructTagAnnotation) GetPosition() Position {
	return a.Position
}

// GetType 获取注解类型
func (a *CommentAnnotation) GetType() AnnotationType {
	return a.Type
}

// GetTargetType 获取目标类型
func (a *CommentAnnotation) GetTargetType() TargetType {
	return a.TargetType
}

// GetTargetName 获取目标名称
func (a *CommentAnnotation) GetTargetName() string {
	return a.TargetName
}

// GetAttributes 获取属性列表
func (a *CommentAnnotation) GetAttributes() []AnnotationAttribute {
	return a.Attributes
}

// GetPosition 获取源码位置
func (a *CommentAnnotation) GetPosition() Position {
	return a.Position
}

// GetAttributeByName 根据名称获取属性
func GetAttributeByName(attrs []AnnotationAttribute, name string) (interface{}, bool) {
	for _, attr := range attrs {
		if attr.Name == name {
			return attr.Value, true
		}
	}
	return nil, false
}

// GetStringAttribute 获取字符串类型的属性
func GetStringAttribute(attrs []AnnotationAttribute, name string, defaultValue string) string {
	value, found := GetAttributeByName(attrs, name)
	if !found {
		return defaultValue
	}

	if strValue, ok := value.(string); ok {
		return strValue
	}

	return defaultValue
}

// GetBoolAttribute 获取布尔类型的属性
func GetBoolAttribute(attrs []AnnotationAttribute, name string, defaultValue bool) bool {
	value, found := GetAttributeByName(attrs, name)
	if !found {
		return defaultValue
	}

	if boolValue, ok := value.(bool); ok {
		return boolValue
	}

	return defaultValue
}

// GetIntAttribute 获取整型的属性
func GetIntAttribute(attrs []AnnotationAttribute, name string, defaultValue int) int {
	value, found := GetAttributeByName(attrs, name)
	if !found {
		return defaultValue
	}

	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return defaultValue
	}
}

// GetFloatAttribute 获取浮点型的属性
func GetFloatAttribute(attrs []AnnotationAttribute, name string, defaultValue float64) float64 {
	value, found := GetAttributeByName(attrs, name)
	if !found {
		return defaultValue
	}

	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return defaultValue
	}
}

// GetStringSliceAttribute 获取字符串切片类型的属性
func GetStringSliceAttribute(attrs []AnnotationAttribute, name string) ([]string, bool) {
	value, found := GetAttributeByName(attrs, name)
	if !found {
		return nil, false
	}

	switch v := value.(type) {
	case []string:
		return v, true
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if strItem, ok := item.(string); ok {
				result = append(result, strItem)
			}
		}
		return result, true
	default:
		if reflect.TypeOf(value).Kind() == reflect.Slice {
			s := reflect.ValueOf(value)
			result := make([]string, 0, s.Len())
			for i := 0; i < s.Len(); i++ {
				item := s.Index(i).Interface()
				if strItem, ok := item.(string); ok {
					result = append(result, strItem)
				}
			}
			return result, true
		}
		return nil, false
	}
}
