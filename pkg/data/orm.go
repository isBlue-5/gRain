// Package data 提供数据访问层的抽象和实现
package data

import (
	"context"
	"errors"
)

// ORMProvider 定义ORM提供者接口，用于支持不同的ORM框架
type ORMProvider interface {
	// Name 返回ORM提供者名称
	Name() string

	// Init 初始化ORM提供者
	Init(config map[string]interface{}) error

	// GetSession 获取数据库会话
	GetSession() (DBSession, error)

	// Close 关闭ORM提供者
	Close() error
}

// ORMRegistry ORM提供者注册中心
type ORMRegistry struct {
	providers map[string]ORMProvider
	default_  string
}

// NewORMRegistry 创建ORM提供者注册中心
func NewORMRegistry() *ORMRegistry {
	return &ORMRegistry{
		providers: make(map[string]ORMProvider),
	}
}

// Register 注册ORM提供者
func (r *ORMRegistry) Register(provider ORMProvider) {
	r.providers[provider.Name()] = provider
}

// SetDefault 设置默认ORM提供者
func (r *ORMRegistry) SetDefault(name string) error {
	if _, ok := r.providers[name]; !ok {
		return errors.New("未注册的ORM提供者: " + name)
	}
	r.default_ = name
	return nil
}

// GetProvider 获取ORM提供者
func (r *ORMRegistry) GetProvider(name string) (ORMProvider, error) {
	if name == "" {
		name = r.default_
	}

	provider, ok := r.providers[name]
	if !ok {
		return nil, errors.New("未注册的ORM提供者: " + name)
	}

	return provider, nil
}

// GetDefaultProvider 获取默认ORM提供者
func (r *ORMRegistry) GetDefaultProvider() (ORMProvider, error) {
	if r.default_ == "" {
		return nil, errors.New("未设置默认ORM提供者")
	}

	return r.providers[r.default_], nil
}

// EntityRelation 实体关联关系类型
type EntityRelation string

const (
	// OneToOne 一对一关系
	OneToOne EntityRelation = "one_to_one"

	// OneToMany 一对多关系
	OneToMany EntityRelation = "one_to_many"

	// ManyToOne 多对一关系
	ManyToOne EntityRelation = "many_to_one"

	// ManyToMany 多对多关系
	ManyToMany EntityRelation = "many_to_many"
)

// FetchType 关联数据加载类型
type FetchType string

const (
	// Eager 立即加载
	Eager FetchType = "eager"

	// Lazy 延迟加载
	Lazy FetchType = "lazy"
)

// CascadeType 级联操作类型
type CascadeType string

const (
	// CascadeAll 所有操作级联
	CascadeAll CascadeType = "all"

	// CascadePersist 保存操作级联
	CascadePersist CascadeType = "persist"

	// CascadeMerge 合并操作级联
	CascadeMerge CascadeType = "merge"

	// CascadeRemove 删除操作级联
	CascadeRemove CascadeType = "remove"

	// CascadeRefresh 刷新操作级联
	CascadeRefresh CascadeType = "refresh"

	// CascadeDetach 分离操作级联
	CascadeDetach CascadeType = "detach"
)

// AssociationOptions 关联选项
type AssociationOptions struct {
	// ForeignKey 外键字段名
	ForeignKey string

	// References 引用字段名
	References string

	// JoinTable 中间表名（多对多关系）
	JoinTable string

	// JoinForeignKey 中间表外键字段名（多对多关系）
	JoinForeignKey string

	// JoinReferences 中间表引用字段名（多对多关系）
	JoinReferences string

	// FetchType 加载类型
	Fetch FetchType

	// Cascade 级联操作
	Cascade []CascadeType

	// Polymorphic 是否多态关联
	Polymorphic bool

	// PolymorphicValue 多态关联值
	PolymorphicValue string
}

// EntityMetadata 实体元数据
type EntityMetadata struct {
	// EntityName 实体名称
	EntityName string

	// TableName 表名
	TableName string

	// Fields 字段映射
	Fields map[string]*FieldMetadata

	// PrimaryKey 主键字段名
	PrimaryKey []string

	// Associations 关联关系
	Associations map[string]*AssociationMetadata

	// Indexes 索引定义
	Indexes map[string]*IndexMetadata
}

// FieldMetadata 字段元数据
type FieldMetadata struct {
	// FieldName 字段名
	FieldName string

	// ColumnName 列名
	ColumnName string

	// FieldType 字段类型
	FieldType string

	// IsPrimaryKey 是否主键
	IsPrimaryKey bool

	// IsUnique 是否唯一
	IsUnique bool

	// IsNullable 是否可空
	IsNullable bool

	// DefaultValue 默认值
	DefaultValue string

	// Comment 注释
	Comment string
}

// AssociationMetadata 关联元数据
type AssociationMetadata struct {
	// FieldName 关联字段名
	FieldName string

	// RelationType 关联类型
	RelationType EntityRelation

	// TargetEntity 目标实体
	TargetEntity string

	// Options 关联选项
	Options AssociationOptions
}

// IndexMetadata 索引元数据
type IndexMetadata struct {
	// IndexName 索引名
	IndexName string

	// Fields 索引字段
	Fields []string

	// IsUnique 是否唯一索引
	IsUnique bool
}

// EntityManager 实体管理器接口
type EntityManager interface {
	// Find 根据主键查找实体
	Find(ctx context.Context, entityType string, id interface{}) (interface{}, error)

	// FindBy 根据条件查找实体
	FindBy(ctx context.Context, entityType string, conditions map[string]interface{}) (interface{}, error)

	// FindAll 查找所有实体
	FindAll(ctx context.Context, entityType string) ([]interface{}, error)

	// Save 保存实体
	Save(ctx context.Context, entity interface{}) error

	// Update 更新实体
	Update(ctx context.Context, entity interface{}) error

	// Delete 删除实体
	Delete(ctx context.Context, entity interface{}) error

	// DeleteById 根据主键删除实体
	DeleteById(ctx context.Context, entityType string, id interface{}) error

	// Count 统计实体数量
	Count(ctx context.Context, entityType string, conditions map[string]interface{}) (int64, error)

	// CreateQuery 创建查询
	CreateQuery(ctx context.Context, query string) (Query, error)

	// GetReference 获取实体引用
	GetReference(ctx context.Context, entityType string, id interface{}) (interface{}, error)

	// Flush 刷新变更
	Flush(ctx context.Context) error

	// Clear 清除实体管理器
	Clear() error

	// GetMetadata 获取实体元数据
	GetMetadata(entityType string) (*EntityMetadata, error)
}

// Query 查询接口
type Query interface {
	// SetParameter 设置参数
	SetParameter(name string, value interface{}) Query

	// SetFirstResult 设置结果偏移量
	SetFirstResult(firstResult int) Query

	// SetMaxResults 设置最大结果数
	SetMaxResults(maxResults int) Query

	// GetResultList 获取结果列表
	GetResultList(ctx context.Context) ([]interface{}, error)

	// GetSingleResult 获取单个结果
	GetSingleResult(ctx context.Context) (interface{}, error)

	// ExecuteUpdate 执行更新操作
	ExecuteUpdate(ctx context.Context) (int, error)
}

// SQLXProvider SQLX提供者实现
type SQLXProvider struct {
	// 具体实现省略
}

// 全局ORM注册中心
var globalORMRegistry = NewORMRegistry()

// RegisterORMProvider 注册ORM提供者
func RegisterORMProvider(provider ORMProvider) {
	globalORMRegistry.Register(provider)
}

// SetDefaultORMProvider 设置默认ORM提供者
func SetDefaultORMProvider(name string) error {
	return globalORMRegistry.SetDefault(name)
}

// GetORMProvider 获取ORM提供者
func GetORMProvider(name string) (ORMProvider, error) {
	return globalORMRegistry.GetProvider(name)
}

// GetDefaultORMProvider 获取默认ORM提供者
func GetDefaultORMProvider() (ORMProvider, error) {
	return globalORMRegistry.GetDefaultProvider()
}
