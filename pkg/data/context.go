// Package data 提供数据访问层和事务管理功能
package data

import (
	"context"
	"fmt"
)

// 上下文键类型
type contextKey string

const (
	// 事务在上下文中的键
	txKey contextKey = "transaction"
)

// WithTransaction 将事务放入上下文
//
// ctx: 原始上下文
// tx: 要存储的事务对象
// 返回: 包含事务的新上下文
func WithTransaction(ctx context.Context, tx Transaction) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// GetTransaction 从上下文获取事务
//
// ctx: 包含事务的上下文
// 返回: 事务对象及是否存在标志
func GetTransaction(ctx context.Context) (Transaction, bool) {
	tx, ok := ctx.Value(txKey).(Transaction)
	return tx, ok
}

// RequireTransaction 从上下文获取事务，如不存在则返回错误
//
// ctx: 包含事务的上下文
// 返回: 事务对象或错误
func RequireTransaction(ctx context.Context) (Transaction, error) {
	tx, ok := GetTransaction(ctx)
	if !ok {
		return nil, fmt.Errorf("事务未在当前上下文中找到")
	}
	return tx, nil
}

// IsTransactionActive 检查上下文是否包含活跃的事务
//
// ctx: 要检查的上下文
// 返回: 是否存在活跃事务
func IsTransactionActive(ctx context.Context) bool {
	_, ok := GetTransaction(ctx)
	return ok
}

// TransactionTemplate 提供事务模板方法，自动处理事务开始、提交和回滚
//
// ctx: 上下文
// session: 数据库会话
// fn: 在事务中执行的函数
// 返回: 执行结果和可能的错误
func TransactionTemplate(ctx context.Context, session DBSession, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	// 检查上下文中是否已有事务
	if _, ok := GetTransaction(ctx); ok {
		// 重用现有事务
		return fn(ctx)
	}

	// 创建新事务
	tx, err := session.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("无法开启事务: %w", err)
	}

	// 将事务放入上下文
	txCtx := WithTransaction(ctx, tx)

	// 执行业务逻辑
	result, err := fn(txCtx)
	if err != nil {
		// 发生错误，回滚事务
		rbErr := tx.Rollback()
		if rbErr != nil {
			return nil, fmt.Errorf("执行失败且事务回滚失败: %w (回滚错误: %v)", err, rbErr)
		}
		return nil, err
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("事务提交失败: %w", err)
	}

	return result, nil
}

// TransactionEvent 钩子类型
type TransactionEvent func(ctx context.Context, tx Transaction)

// TransactionTemplateV2 支持事件钩子的事务模板
func TransactionTemplateV2(
	ctx context.Context,
	session DBSession,
	fn func(ctx context.Context) (interface{}, error),
	beforeCommit TransactionEvent,
	afterCommit TransactionEvent,
	beforeRollback TransactionEvent,
	afterRollback TransactionEvent,
	logger func(format string, args ...interface{}),
) (interface{}, error) {
	if logger == nil {
		logger = func(format string, args ...interface{}) { fmt.Printf(format+"\n", args...) }
	}
	if _, ok := GetTransaction(ctx); ok {
		logger("[TxTemplateV2] 已有事务，直接执行业务逻辑")
		return fn(ctx)
	}
	// 创建新事务
	tx, err := session.BeginTx(ctx, nil)
	if err != nil {
		logger("[TxTemplateV2] 开启事务失败: %v", err)
		return nil, fmt.Errorf("无法开启事务: %w", err)
	}
	txCtx := WithTransaction(ctx, tx)
	logger("[TxTemplateV2] 开启新事务: ID=%d, Level=%d", tx.ID(), tx.Level())
	result, err := fn(txCtx)
	if err != nil {
		if beforeRollback != nil {
			beforeRollback(txCtx, tx)
		}
		rbErr := tx.Rollback()
		if rbErr != nil {
			logger("[TxTemplateV2] 回滚事务失败: %v", rbErr)
			return nil, fmt.Errorf("执行失败且事务回滚失败: %w (回滚错误: %v)", err, rbErr)
		}
		logger("[TxTemplateV2] 事务已回滚: ID=%d, 状态=%s", tx.ID(), tx.Status())
		if afterRollback != nil {
			afterRollback(txCtx, tx)
		}
		return nil, err
	}
	if beforeCommit != nil {
		beforeCommit(txCtx, tx)
	}
	if err := tx.Commit(); err != nil {
		logger("[TxTemplateV2] 提交事务失败: %v", err)
		return nil, fmt.Errorf("事务提交失败: %w", err)
	}
	logger("[TxTemplateV2] 事务已提交: ID=%d, 状态=%s", tx.ID(), tx.Status())
	if afterCommit != nil {
		afterCommit(txCtx, tx)
	}
	return result, nil
}
