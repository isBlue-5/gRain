// Package data 提供数据访问层的抽象和实现
package data

import (
	"context"
	"reflect"
	"strings"
	"sync"

	"gorm.io/gorm"
)

// GormProvider GORM提供者实现
type GormProvider struct {
	// db GORM数据库连接
	db *gorm.DB

	// config 配置信息
	config map[string]interface{}

	// metadataCache 实体元数据缓存
	metadataCache map[string]*EntityMetadata

	// mutex 互斥锁
	mutex sync.RWMutex
}

// NewGormProvider 创建GORM提供者
func NewGormProvider(db *gorm.DB) *GormProvider {
	return &GormProvider{
		db:            db,
		config:        make(map[string]interface{}),
		metadataCache: make(map[string]*EntityMetadata),
	}
}

// Name 返回ORM提供者名称
func (p *GormProvider) Name() string {
	return "gorm"
}

// Init 初始化ORM提供者
func (p *GormProvider) Init(config map[string]interface{}) error {
	p.config = config
	return nil
}

// GetSession 获取数据库会话
func (p *GormProvider) GetSession() (DBSession, error) {
	return NewGormSession(p.db), nil
}

// Close 关闭ORM提供者
func (p *GormProvider) Close() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetEntityManager 获取实体管理器
func (p *GormProvider) GetEntityManager() (EntityManager, error) {
	return NewGormEntityManager(p), nil
}

// GormEntityManager GORM实体管理器实现
type GormEntityManager struct {
	// provider GORM提供者
	provider *GormProvider

	// db GORM数据库连接
	db *gorm.DB
}

// NewGormEntityManager 创建GORM实体管理器
func NewGormEntityManager(provider *GormProvider) *GormEntityManager {
	return &GormEntityManager{
		provider: provider,
		db:       provider.db,
	}
}

// Find 根据主键查找实体
func (em *GormEntityManager) Find(ctx context.Context, entityType string, id interface{}) (interface{}, error) {
	// 获取实体元数据
	metadata, err := em.GetMetadata(entityType)
	if err != nil {
		return nil, err
	}

	// 创建实体实例
	entityValue := reflect.New(reflect.TypeOf(entityType)).Elem()
	entity := entityValue.Interface()

	// 查询实体
	result := em.db.WithContext(ctx).First(entity, id)
	if result.Error != nil {
		return nil, result.Error
	}

	// 处理关联关系
	if err := em.loadAssociations(ctx, entity, metadata); err != nil {
		return nil, err
	}

	return entity, nil
}

// FindBy 根据条件查找实体
func (em *GormEntityManager) FindBy(ctx context.Context, entityType string, conditions map[string]interface{}) (interface{}, error) {
	// 获取实体元数据
	metadata, err := em.GetMetadata(entityType)
	if err != nil {
		return nil, err
	}

	// 创建实体实例
	entityValue := reflect.New(reflect.TypeOf(entityType)).Elem()
	entity := entityValue.Interface()

	// 查询实体
	result := em.db.WithContext(ctx).Where(conditions).First(entity)
	if result.Error != nil {
		return nil, result.Error
	}

	// 处理关联关系
	if err := em.loadAssociations(ctx, entity, metadata); err != nil {
		return nil, err
	}

	return entity, nil
}

// FindAll 查找所有实体
func (em *GormEntityManager) FindAll(ctx context.Context, entityType string) ([]interface{}, error) {
	// 获取实体元数据
	metadata, err := em.GetMetadata(entityType)
	if err != nil {
		return nil, err
	}

	// 创建实体切片
	sliceType := reflect.SliceOf(reflect.TypeOf(entityType))
	sliceValue := reflect.MakeSlice(sliceType, 0, 0)
	entitiesPtr := reflect.New(sliceType)
	entitiesPtr.Elem().Set(sliceValue)

	// 查询实体
	result := em.db.WithContext(ctx).Find(entitiesPtr.Interface())
	if result.Error != nil {
		return nil, result.Error
	}

	// 转换为接口切片
	entities := entitiesPtr.Elem()
	interfaceSlice := make([]interface{}, entities.Len())
	for i := 0; i < entities.Len(); i++ {
		entity := entities.Index(i).Interface()

		// 处理关联关系
		if err := em.loadAssociations(ctx, entity, metadata); err != nil {
			return nil, err
		}

		interfaceSlice[i] = entity
	}

	return interfaceSlice, nil
}

// Save 保存实体
func (em *GormEntityManager) Save(ctx context.Context, entity interface{}) error {
	// 获取实体类型
	entityType := reflect.TypeOf(entity).Elem().Name()

	// 获取实体元数据
	metadata, err := em.GetMetadata(entityType)
	if err != nil {
		return err
	}

	// 处理级联保存
	if err := em.cascadeSave(ctx, entity, metadata); err != nil {
		return err
	}

	// 保存实体
	result := em.db.WithContext(ctx).Create(entity)
	return result.Error
}

// Update 更新实体
func (em *GormEntityManager) Update(ctx context.Context, entity interface{}) error {
	// 获取实体类型
	entityType := reflect.TypeOf(entity).Elem().Name()

	// 获取实体元数据
	metadata, err := em.GetMetadata(entityType)
	if err != nil {
		return err
	}

	// 处理级联更新
	if err := em.cascadeUpdate(ctx, entity, metadata); err != nil {
		return err
	}

	// 更新实体
	result := em.db.WithContext(ctx).Save(entity)
	return result.Error
}

// Delete 删除实体
func (em *GormEntityManager) Delete(ctx context.Context, entity interface{}) error {
	// 获取实体类型
	entityType := reflect.TypeOf(entity).Elem().Name()

	// 获取实体元数据
	metadata, err := em.GetMetadata(entityType)
	if err != nil {
		return err
	}

	// 处理级联删除
	if err := em.cascadeDelete(ctx, entity, metadata); err != nil {
		return err
	}

	// 删除实体
	result := em.db.WithContext(ctx).Delete(entity)
	return result.Error
}

// DeleteById 根据主键删除实体
func (em *GormEntityManager) DeleteById(ctx context.Context, entityType string, id interface{}) error {
	// 验证实体元数据存在
	_, err := em.GetMetadata(entityType)
	if err != nil {
		return err
	}

	// 查找实体
	entity, err := em.Find(ctx, entityType, id)
	if err != nil {
		return err
	}

	// 删除实体
	return em.Delete(ctx, entity)
}

// Count 统计实体数量
func (em *GormEntityManager) Count(ctx context.Context, entityType string, conditions map[string]interface{}) (int64, error) {
	// 验证实体元数据存在
	_, err := em.GetMetadata(entityType)
	if err != nil {
		return 0, err
	}

	// 创建实体实例
	entityValue := reflect.New(reflect.TypeOf(entityType)).Elem()
	entity := entityValue.Interface()

	// 统计实体
	var count int64
	result := em.db.WithContext(ctx).Model(entity).Where(conditions).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}

	return count, nil
}

// CreateQuery 创建查询
func (em *GormEntityManager) CreateQuery(ctx context.Context, query string) (Query, error) {
	return NewGormQuery(em.db.WithContext(ctx), query), nil
}

// GetReference 获取实体引用
func (em *GormEntityManager) GetReference(ctx context.Context, entityType string, id interface{}) (interface{}, error) {
	// 获取实体元数据
	metadata, err := em.GetMetadata(entityType)
	if err != nil {
		return nil, err
	}

	// 创建实体实例
	entityValue := reflect.New(reflect.TypeOf(entityType)).Elem()
	entity := entityValue.Interface()

	// 设置主键
	primaryKeyField := entityValue.FieldByName(metadata.PrimaryKey[0])
	primaryKeyField.Set(reflect.ValueOf(id))

	return entity, nil
}

// Flush 刷新变更
func (em *GormEntityManager) Flush(ctx context.Context) error {
	// GORM自动提交，无需刷新
	return nil
}

// Clear 清除实体管理器
func (em *GormEntityManager) Clear() error {
	// 重置数据库连接
	em.db = em.provider.db
	return nil
}

// GetMetadata 获取实体元数据
func (em *GormEntityManager) GetMetadata(entityType string) (*EntityMetadata, error) {
	// 从缓存获取
	em.provider.mutex.RLock()
	metadata, ok := em.provider.metadataCache[entityType]
	em.provider.mutex.RUnlock()

	if ok {
		return metadata, nil
	}

	// 解析实体元数据
	metadata, err := em.parseEntityMetadata(entityType)
	if err != nil {
		return nil, err
	}

	// 缓存元数据
	em.provider.mutex.Lock()
	em.provider.metadataCache[entityType] = metadata
	em.provider.mutex.Unlock()

	return metadata, nil
}

// parseEntityMetadata 解析实体元数据
func (em *GormEntityManager) parseEntityMetadata(entityType string) (*EntityMetadata, error) {
	// 创建实体实例
	entityValue := reflect.New(reflect.TypeOf(entityType)).Elem()
	entity := entityValue.Interface()

	// 获取表名
	stmt := &gorm.Statement{DB: em.db}
	if err := stmt.Parse(entity); err != nil {
		return nil, err
	}

	// 创建元数据
	metadata := &EntityMetadata{
		EntityName:   entityType,
		TableName:    stmt.Table,
		Fields:       make(map[string]*FieldMetadata),
		PrimaryKey:   make([]string, 0),
		Associations: make(map[string]*AssociationMetadata),
		Indexes:      make(map[string]*IndexMetadata),
	}

	// 解析字段
	for _, field := range stmt.Schema.Fields {
		// 创建字段元数据
		fieldMeta := &FieldMetadata{
			FieldName:    field.Name,
			ColumnName:   field.DBName,
			FieldType:    field.FieldType.String(),
			IsPrimaryKey: field.PrimaryKey,
			IsUnique:     field.Unique,
			IsNullable:   !field.NotNull,
			DefaultValue: field.DefaultValue,
			Comment:      field.Comment,
		}

		// 添加字段元数据
		metadata.Fields[field.Name] = fieldMeta

		// 记录主键
		if field.PrimaryKey {
			metadata.PrimaryKey = append(metadata.PrimaryKey, field.Name)
		}

		// 关联关系解析会在Schema级别处理，这里跳过
	}

	// 解析关联关系（Schema级别）
	relations := stmt.Schema.Relationships

	// 处理HasOne关联
	for _, rel := range relations.HasOne {
		assocMeta := &AssociationMetadata{
			FieldName:    rel.Field.Name,
			TargetEntity: rel.FieldSchema.ModelType.Name(),
			RelationType: OneToOne,
			Options:      AssociationOptions{},
		}
		if len(rel.References) > 0 {
			assocMeta.Options.ForeignKey = rel.References[0].ForeignKey.DBName
			assocMeta.Options.References = rel.References[0].PrimaryKey.DBName
		}
		metadata.Associations[rel.Field.Name] = assocMeta
	}

	// 处理BelongsTo关联
	for _, rel := range relations.BelongsTo {
		assocMeta := &AssociationMetadata{
			FieldName:    rel.Field.Name,
			TargetEntity: rel.FieldSchema.ModelType.Name(),
			RelationType: ManyToOne,
			Options:      AssociationOptions{},
		}
		if len(rel.References) > 0 {
			assocMeta.Options.ForeignKey = rel.References[0].ForeignKey.DBName
			assocMeta.Options.References = rel.References[0].PrimaryKey.DBName
		}
		metadata.Associations[rel.Field.Name] = assocMeta
	}

	// 处理HasMany关联
	for _, rel := range relations.HasMany {
		assocMeta := &AssociationMetadata{
			FieldName:    rel.Field.Name,
			TargetEntity: rel.FieldSchema.ModelType.Name(),
			RelationType: OneToMany,
			Options:      AssociationOptions{},
		}
		if len(rel.References) > 0 {
			assocMeta.Options.ForeignKey = rel.References[0].ForeignKey.DBName
			assocMeta.Options.References = rel.References[0].PrimaryKey.DBName
		}
		metadata.Associations[rel.Field.Name] = assocMeta
	}

	// 处理Many2Many关联
	for _, rel := range relations.Many2Many {
		assocMeta := &AssociationMetadata{
			FieldName:    rel.Field.Name,
			TargetEntity: rel.FieldSchema.ModelType.Name(),
			RelationType: ManyToMany,
			Options:      AssociationOptions{},
		}
		if len(rel.References) > 0 {
			assocMeta.Options.ForeignKey = rel.References[0].ForeignKey.DBName
			assocMeta.Options.References = rel.References[0].PrimaryKey.DBName
		}
		if rel.JoinTable != nil {
			assocMeta.Options.JoinTable = rel.JoinTable.Name
		}
		metadata.Associations[rel.Field.Name] = assocMeta
	}

	// 解析索引
	for _, index := range stmt.Schema.ParseIndexes() {
		// 创建索引元数据
		indexMeta := &IndexMetadata{
			IndexName: index.Name,
			Fields:    make([]string, len(index.Fields)),
			IsUnique:  index.Class == "UNIQUE",
		}

		// 记录索引字段
		for i, field := range index.Fields {
			if field.Field != nil {
				indexMeta.Fields[i] = field.Field.Name
			} else if field.Expression != "" {
				indexMeta.Fields[i] = field.Expression
			}
		}

		// 添加索引元数据
		metadata.Indexes[index.Name] = indexMeta
	}

	return metadata, nil
}

// loadAssociations 加载关联关系
func (em *GormEntityManager) loadAssociations(ctx context.Context, entity interface{}, metadata *EntityMetadata) error {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// 遍历关联关系
	for _, assoc := range metadata.Associations {
		// 跳过延迟加载
		if assoc.Options.Fetch == Lazy {
			continue
		}

		// 获取关联字段
		field := entityValue.FieldByName(assoc.FieldName)
		if !field.IsValid() {
			continue
		}

		// 加载关联数据
		err := em.db.WithContext(ctx).Model(entity).Association(assoc.FieldName).Find(field.Addr().Interface())
		if err != nil {
			return err
		}
	}

	return nil
}

// cascadeSave 级联保存
func (em *GormEntityManager) cascadeSave(ctx context.Context, entity interface{}, metadata *EntityMetadata) error {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// 遍历关联关系
	for _, assoc := range metadata.Associations {
		// 检查是否需要级联保存
		if !containsCascadeType(assoc.Options.Cascade, CascadePersist) &&
			!containsCascadeType(assoc.Options.Cascade, CascadeAll) {
			continue
		}

		// 获取关联字段
		field := entityValue.FieldByName(assoc.FieldName)
		if !field.IsValid() || field.IsZero() {
			continue
		}

		// 处理不同类型的关联
		switch assoc.RelationType {
		case OneToOne, ManyToOne:
			// 保存单个关联实体
			if err := em.Save(ctx, field.Interface()); err != nil {
				return err
			}
		case OneToMany, ManyToMany:
			// 保存多个关联实体
			for i := 0; i < field.Len(); i++ {
				if err := em.Save(ctx, field.Index(i).Interface()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// cascadeUpdate 级联更新
func (em *GormEntityManager) cascadeUpdate(ctx context.Context, entity interface{}, metadata *EntityMetadata) error {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// 遍历关联关系
	for _, assoc := range metadata.Associations {
		// 检查是否需要级联更新
		if !containsCascadeType(assoc.Options.Cascade, CascadeMerge) &&
			!containsCascadeType(assoc.Options.Cascade, CascadeAll) {
			continue
		}

		// 获取关联字段
		field := entityValue.FieldByName(assoc.FieldName)
		if !field.IsValid() || field.IsZero() {
			continue
		}

		// 处理不同类型的关联
		switch assoc.RelationType {
		case OneToOne, ManyToOne:
			// 更新单个关联实体
			if err := em.Update(ctx, field.Interface()); err != nil {
				return err
			}
		case OneToMany, ManyToMany:
			// 更新多个关联实体
			for i := 0; i < field.Len(); i++ {
				if err := em.Update(ctx, field.Index(i).Interface()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// cascadeDelete 级联删除
func (em *GormEntityManager) cascadeDelete(ctx context.Context, entity interface{}, metadata *EntityMetadata) error {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// 遍历关联关系
	for _, assoc := range metadata.Associations {
		// 检查是否需要级联删除
		if !containsCascadeType(assoc.Options.Cascade, CascadeRemove) &&
			!containsCascadeType(assoc.Options.Cascade, CascadeAll) {
			continue
		}

		// 获取关联字段
		field := entityValue.FieldByName(assoc.FieldName)
		if !field.IsValid() || field.IsZero() {
			continue
		}

		// 处理不同类型的关联
		switch assoc.RelationType {
		case OneToOne, ManyToOne:
			// 删除单个关联实体
			if err := em.Delete(ctx, field.Interface()); err != nil {
				return err
			}
		case OneToMany, ManyToMany:
			// 删除多个关联实体
			for i := 0; i < field.Len(); i++ {
				if err := em.Delete(ctx, field.Index(i).Interface()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// GormQuery GORM查询实现
type GormQuery struct {
	// db GORM数据库连接
	db *gorm.DB

	// query 查询语句
	query string

	// params 查询参数
	params map[string]interface{}

	// firstResult 结果偏移量
	firstResult int

	// maxResults 最大结果数
	maxResults int
}

// NewGormQuery 创建GORM查询
func NewGormQuery(db *gorm.DB, query string) *GormQuery {
	return &GormQuery{
		db:     db,
		query:  query,
		params: make(map[string]interface{}),
	}
}

// SetParameter 设置参数
func (q *GormQuery) SetParameter(name string, value interface{}) Query {
	q.params[name] = value
	return q
}

// SetFirstResult 设置结果偏移量
func (q *GormQuery) SetFirstResult(firstResult int) Query {
	q.firstResult = firstResult
	return q
}

// SetMaxResults 设置最大结果数
func (q *GormQuery) SetMaxResults(maxResults int) Query {
	q.maxResults = maxResults
	return q
}

// GetResultList 获取结果列表
func (q *GormQuery) GetResultList(ctx context.Context) ([]interface{}, error) {
	// 替换命名参数
	query, args := q.replaceNamedParams(q.query, q.params)

	// 执行查询
	var results []map[string]interface{}
	db := q.db.WithContext(ctx).Raw(query, args...)

	// 应用分页
	if q.firstResult > 0 {
		db = db.Offset(q.firstResult)
	}
	if q.maxResults > 0 {
		db = db.Limit(q.maxResults)
	}

	// 获取结果
	if err := db.Find(&results).Error; err != nil {
		return nil, err
	}

	// 转换为接口切片
	interfaceSlice := make([]interface{}, len(results))
	for i, result := range results {
		interfaceSlice[i] = result
	}

	return interfaceSlice, nil
}

// GetSingleResult 获取单个结果
func (q *GormQuery) GetSingleResult(ctx context.Context) (interface{}, error) {
	// 替换命名参数
	query, args := q.replaceNamedParams(q.query, q.params)

	// 执行查询
	var result map[string]interface{}
	if err := q.db.WithContext(ctx).Raw(query, args...).First(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

// ExecuteUpdate 执行更新操作
func (q *GormQuery) ExecuteUpdate(ctx context.Context) (int, error) {
	// 替换命名参数
	query, args := q.replaceNamedParams(q.query, q.params)

	// 执行更新
	result := q.db.WithContext(ctx).Exec(query, args...)
	if result.Error != nil {
		return 0, result.Error
	}

	return int(result.RowsAffected), nil
}

// replaceNamedParams 替换命名参数
func (q *GormQuery) replaceNamedParams(query string, params map[string]interface{}) (string, []interface{}) {
	args := make([]interface{}, 0, len(params))

	// 替换 :param 形式的参数
	for name, value := range params {
		placeholder := ":" + name
		query = strings.Replace(query, placeholder, "?", -1)
		args = append(args, value)
	}

	return query, args
}

// containsCascadeType 检查级联类型是否包含指定类型
func containsCascadeType(types []CascadeType, targetType CascadeType) bool {
	for _, t := range types {
		if t == targetType {
			return true
		}
	}
	return false
}
