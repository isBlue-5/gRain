// Package data 提供统一的数据访问与事务接口，支持多种ORM和原生SQL
package data

import (
	"context"
	"database/sql"
	"sync/atomic"
)

// DBSession 定义数据库会话的通用接口，支持查询、执行和事务
type DBSession interface {
	Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	BeginTx(ctx context.Context, opts *sql.TxOptions) (Transaction, error)
}

// Transaction 事务接口，继承DBSession，增加提交和回滚
// 新增事务状态跟踪方法
type Transaction interface {
	DBSession
	Commit() error
	Rollback() error
	ID() uint64
	Level() int
	Status() string
}

// SQLSession 基于原生sql.DB的会话实现
type SQLSession struct {
	db *sql.DB
}

// NewSQLSession 创建SQLSession实例
func NewSQLSession(db *sql.DB) *SQLSession {
	return &SQLSession{db: db}
}

// Query 实现DBSession的查询方法
func (s *SQLSession) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	// 这里只做演示，实际应结合反射或ORM进行结果映射
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// 简单的结果映射实现
	if rows.Next() {
		// 根据dest的类型进行扫描
		switch v := dest.(type) {
		case *string:
			return rows.Scan(v)
		case *int:
			return rows.Scan(v)
		case *int64:
			return rows.Scan(v)
		default:
			// 对于复杂类型，使用interface{}接收
			var result interface{}
			err := rows.Scan(&result)
			if err != nil {
				return err
			}
			// 这里可以扩展为更复杂的映射逻辑
			return nil
		}
	}
	return sql.ErrNoRows
}

// Exec 实现DBSession的执行方法
func (s *SQLSession) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

// BeginTx 实现DBSession的开启事务方法
func (s *SQLSession) BeginTx(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	tx, err := s.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newSQLTx(tx), nil
}

// SQLTx 基于原生sql.Tx的事务实现
type SQLTx struct {
	tx     *sql.Tx
	id     uint64
	status string
}

var globalSQLTxCounter uint64

func newSQLTx(tx *sql.Tx) *SQLTx {
	return &SQLTx{
		tx:     tx,
		id:     atomic.AddUint64(&globalSQLTxCounter, 1),
		status: "active",
	}
}

func (t *SQLTx) ID() uint64     { return t.id }
func (t *SQLTx) Level() int     { return 0 }
func (t *SQLTx) Status() string { return t.status }

// Query 实现DBSession的查询方法（事务内）
func (t *SQLTx) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// 简单的结果映射实现（与SQLSession相同逻辑）
	if rows.Next() {
		// 根据dest的类型进行扫描
		switch v := dest.(type) {
		case *string:
			return rows.Scan(v)
		case *int:
			return rows.Scan(v)
		case *int64:
			return rows.Scan(v)
		default:
			// 对于复杂类型，使用interface{}接收
			var result interface{}
			err := rows.Scan(&result)
			if err != nil {
				return err
			}
			// 这里可以扩展为更复杂的映射逻辑
			return nil
		}
	}
	return sql.ErrNoRows
}

// Exec 实现DBSession的执行方法（事务内）
func (t *SQLTx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

// BeginTx 事务内再次开启事务（嵌套事务，通常不支持）
func (t *SQLTx) BeginTx(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	return nil, sql.ErrTxDone // 不支持嵌套事务
}

// Commit 提交事务
func (t *SQLTx) Commit() error {
	err := t.tx.Commit()
	if err == nil {
		t.status = "committed"
	}
	return err
}

// Rollback 回滚事务
func (t *SQLTx) Rollback() error {
	err := t.tx.Rollback()
	if err == nil {
		t.status = "rolledback"
	}
	return err
}

// 预留GORM等ORM扩展点
// type GormSession struct { ... }
// func NewGormSession(db *gorm.DB) *GormSession { ... }
