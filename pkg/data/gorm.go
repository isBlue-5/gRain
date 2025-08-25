// Package data 提供数据访问层和事务管理功能
package data

import (
	"context"
	"database/sql"
	"fmt"

	"sync/atomic"

	"gorm.io/gorm"
)

// 事务状态类型
const (
	TxStatusActive     = "active"
	TxStatusCommitted  = "committed"
	TxStatusRolledBack = "rolledback"
)

var globalTxCounter uint64

// gormResult 包装 GORM 的 RowsAffected 以实现 sql.Result
type gormResult struct {
	raw          sql.Result
	rowsAffected int64
}

func (r *gormResult) LastInsertId() (int64, error) {
	if r.raw != nil {
		return r.raw.LastInsertId()
	}
	return 0, nil // GORM 不总能提供
}
func (r *gormResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

// GormSession 基于GORM的会话实现
type GormSession struct {
	db *gorm.DB
}

// NewGormSession 创建GormSession实例
func NewGormSession(db *gorm.DB) *GormSession {
	return &GormSession{db: db}
}

// Query 实现DBSession的查询方法
func (s *GormSession) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	// 使用原生SQL查询
	return s.db.WithContext(ctx).Raw(query, args...).Scan(dest).Error
}

// Exec 实现DBSession的执行方法
func (s *GormSession) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	// 使用原生SQL执行
	result := s.db.WithContext(ctx).Exec(query, args...)
	return &gormResult{rowsAffected: result.RowsAffected}, result.Error
}

// BeginTx 实现DBSession的开启事务方法
func (s *GormSession) BeginTx(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	var sessionOpt *gorm.Session
	if opts != nil {
		sessionOpt = &gorm.Session{
			Context: ctx,
			NewDB:   true,
			// GORM v2 没有直接的 Isolation/ReadOnly 字段，但可以通过后续扩展
		}
	} else {
		sessionOpt = &gorm.Session{Context: ctx, NewDB: true}
	}

	db := s.db.Session(sessionOpt)
	tx := db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("开启GORM事务失败: %w", tx.Error)
	}

	return NewGormTransaction(tx, 0), nil
}

// GormTransaction 基于GORM的事务实现
type GormTransaction struct {
	tx     *gorm.DB
	id     uint64
	level  int
	status string
}

// NewGormTransaction 创建新的 GormTransaction
func NewGormTransaction(tx *gorm.DB, level int) *GormTransaction {
	return &GormTransaction{
		tx:     tx,
		id:     atomic.AddUint64(&globalTxCounter, 1),
		level:  level,
		status: TxStatusActive,
	}
}

// Query 实现DBSession的查询方法（事务内）
func (t *GormTransaction) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	// 使用原生SQL查询（在事务内）
	return t.tx.WithContext(ctx).Raw(query, args...).Scan(dest).Error
}

// Exec 实现DBSession的执行方法（事务内）
func (t *GormTransaction) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	// 使用原生SQL执行（在事务内）
	result := t.tx.WithContext(ctx).Exec(query, args...)
	return &gormResult{rowsAffected: result.RowsAffected}, result.Error
}

// BeginTx 事务内再次开启事务（嵌套事务）
func (t *GormTransaction) BeginTx(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	// GORM支持SavePoint，可以实现嵌套事务
	savepointName := fmt.Sprintf("sp_%p", t)
	if err := t.tx.WithContext(ctx).SavePoint(savepointName).Error; err != nil {
		return nil, fmt.Errorf("创建保存点失败: %w", err)
	}

	// 返回带保存点的事务
	nested := &GormNestedTransaction{
		parent:        t,
		savepointName: savepointName,
		tx:            t.tx,
		id:            atomic.AddUint64(&globalTxCounter, 1),
		level:         t.level + 1,
		status:        TxStatusActive,
	}
	return nested, nil
}

// Commit 提交事务
func (t *GormTransaction) Commit() error {
	err := t.tx.Commit().Error
	if err == nil {
		t.status = TxStatusCommitted
	}
	return err
}

// Rollback 回滚事务
func (t *GormTransaction) Rollback() error {
	err := t.tx.Rollback().Error
	if err == nil {
		t.status = TxStatusRolledBack
	}
	return err
}

// GormNestedTransaction 基于GORM的嵌套事务实现
type GormNestedTransaction struct {
	parent        *GormTransaction
	savepointName string
	tx            *gorm.DB
	id            uint64
	level         int
	status        string
}

// Query 实现DBSession的查询方法（嵌套事务内）
func (t *GormNestedTransaction) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return t.tx.WithContext(ctx).Raw(query, args...).Scan(dest).Error
}

// Exec 实现DBSession的执行方法（嵌套事务内）
func (t *GormNestedTransaction) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	result := t.tx.WithContext(ctx).Exec(query, args...)
	return &gormResult{rowsAffected: result.RowsAffected}, result.Error
}

// BeginTx 嵌套事务内再次开启事务（多层嵌套）
func (t *GormNestedTransaction) BeginTx(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	// 创建新的保存点
	savepointName := fmt.Sprintf("%s_child_%p", t.savepointName, t)
	if err := t.tx.WithContext(ctx).SavePoint(savepointName).Error; err != nil {
		return nil, fmt.Errorf("创建嵌套保存点失败: %w", err)
	}

	// 返回带保存点的事务
	return &GormNestedTransaction{
		parent:        t.parent,
		savepointName: savepointName,
		tx:            t.tx,
		id:            atomic.AddUint64(&globalTxCounter, 1),
		level:         t.level + 1,
		status:        TxStatusActive,
	}, nil
}

// Commit 提交嵌套事务（释放保存点）
func (t *GormNestedTransaction) Commit() error {
	t.status = TxStatusCommitted
	return nil
}

// Rollback 回滚嵌套事务（回滚到保存点）
func (t *GormNestedTransaction) Rollback() error {
	err := t.tx.RollbackTo(t.savepointName).Error
	if err == nil {
		t.status = TxStatusRolledBack
	}
	return err
}

// GormModel 提供GORM模型的基础结构
type GormModel struct {
	Model interface{}
	DB    *gorm.DB
}

// NewGormModel 创建GORM模型
func NewGormModel(model interface{}, db *gorm.DB) *GormModel {
	return &GormModel{
		Model: model,
		DB:    db,
	}
}

// WithContext 设置上下文
func (m *GormModel) WithContext(ctx context.Context) *GormModel {
	return &GormModel{
		Model: m.Model,
		DB:    m.DB.WithContext(ctx),
	}
}

// WithTransaction 在事务中执行操作
func (m *GormModel) WithTransaction(tx Transaction) *GormModel {
	if gormTx, ok := tx.(*GormTransaction); ok {
		return &GormModel{
			Model: m.Model,
			DB:    gormTx.tx,
		}
	}
	// 如果不是GormTransaction，则返回原始模型
	return m
}

func (t *GormTransaction) ID() uint64     { return t.id }
func (t *GormTransaction) Level() int     { return t.level }
func (t *GormTransaction) Status() string { return t.status }

func (t *GormNestedTransaction) ID() uint64     { return t.id }
func (t *GormNestedTransaction) Level() int     { return t.level }
func (t *GormNestedTransaction) Status() string { return t.status }
