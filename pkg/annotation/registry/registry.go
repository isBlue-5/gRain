// Package registry 提供注解注册和查找功能
package registry

import (
	"github.com/grain-framework/grain/pkg/annotation/types"
)

// Registry 注解注册中心
// 存储所有解析得到的注解，并提供查询接口
type Registry interface {
	// Register 注册一个注解
	Register(annotation types.Annotation)

	// RegisterAll 批量注册注解
	RegisterAll(annotations []types.Annotation)

	// FindByType 根据注解类型查找注解
	FindByType(annotationType types.AnnotationType) []types.Annotation

	// FindByTargetType 根据目标类型查找注解
	FindByTargetType(targetType types.TargetType) []types.Annotation

	// FindByTarget 根据目标类型和名称查找注解
	FindByTarget(targetType types.TargetType, targetName string) []types.Annotation

	// FindByTypeAndTarget 根据注解类型和目标查找注解
	FindByTypeAndTarget(annotationType types.AnnotationType, targetType types.TargetType, targetName string) []types.Annotation

	// GetAll 获取所有注册的注解
	GetAll() []types.Annotation
}

// DefaultRegistry 默认注解注册中心实现
type DefaultRegistry struct {
	// 所有注解
	annotations []types.Annotation

	// 按注解类型索引
	byType map[types.AnnotationType][]types.Annotation

	// 按目标类型索引
	byTargetType map[types.TargetType][]types.Annotation

	// 按目标类型和名称索引
	byTarget map[string][]types.Annotation

	// 按注解类型和目标索引
	byTypeAndTarget map[string][]types.Annotation
}

// NewRegistry 创建新的注解注册中心
func NewRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		annotations:     make([]types.Annotation, 0),
		byType:          make(map[types.AnnotationType][]types.Annotation),
		byTargetType:    make(map[types.TargetType][]types.Annotation),
		byTarget:        make(map[string][]types.Annotation),
		byTypeAndTarget: make(map[string][]types.Annotation),
	}
}

// Register 注册一个注解
func (r *DefaultRegistry) Register(annotation types.Annotation) {
	// 添加到所有注解列表
	r.annotations = append(r.annotations, annotation)

	// 按注解类型索引
	annoType := annotation.GetType()
	r.byType[annoType] = append(r.byType[annoType], annotation)

	// 按目标类型索引
	targetType := annotation.GetTargetType()
	r.byTargetType[targetType] = append(r.byTargetType[targetType], annotation)

	// 按目标类型和名称索引
	targetKey := makeTargetKey(targetType, annotation.GetTargetName())
	r.byTarget[targetKey] = append(r.byTarget[targetKey], annotation)

	// 按注解类型和目标索引
	typeTargetKey := makeTypeTargetKey(annoType, targetType, annotation.GetTargetName())
	r.byTypeAndTarget[typeTargetKey] = append(r.byTypeAndTarget[typeTargetKey], annotation)
}

// RegisterAll 批量注册注解
func (r *DefaultRegistry) RegisterAll(annotations []types.Annotation) {
	for _, annotation := range annotations {
		r.Register(annotation)
	}
}

// FindByType 根据注解类型查找注解
func (r *DefaultRegistry) FindByType(annotationType types.AnnotationType) []types.Annotation {
	return r.byType[annotationType]
}

// FindByTargetType 根据目标类型查找注解
func (r *DefaultRegistry) FindByTargetType(targetType types.TargetType) []types.Annotation {
	return r.byTargetType[targetType]
}

// FindByTarget 根据目标类型和名称查找注解
func (r *DefaultRegistry) FindByTarget(targetType types.TargetType, targetName string) []types.Annotation {
	key := makeTargetKey(targetType, targetName)
	return r.byTarget[key]
}

// FindByTypeAndTarget 根据注解类型和目标查找注解
func (r *DefaultRegistry) FindByTypeAndTarget(annotationType types.AnnotationType, targetType types.TargetType, targetName string) []types.Annotation {
	key := makeTypeTargetKey(annotationType, targetType, targetName)
	return r.byTypeAndTarget[key]
}

// GetAll 获取所有注册的注解
func (r *DefaultRegistry) GetAll() []types.Annotation {
	return r.annotations
}

// 创建目标键
func makeTargetKey(targetType types.TargetType, targetName string) string {
	return string(targetType) + ":" + targetName
}

// 创建类型和目标组合键
func makeTypeTargetKey(annotationType types.AnnotationType, targetType types.TargetType, targetName string) string {
	return string(annotationType) + ":" + string(targetType) + ":" + targetName
}
