package transaction

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/grain-framework/grain/pkg/data"
)

// TransactionLogger 事务日志记录器
type TransactionLogger struct {
	isEnabled bool
	logLevel  string
}

// NewTransactionLogger 创建事务日志记录器
func NewTransactionLogger(isEnabled bool, logLevel string) *TransactionLogger {
	return &TransactionLogger{
		isEnabled: isEnabled,
		logLevel:  logLevel,
	}
}

// BeforeCommit 提交前钩子
func (l *TransactionLogger) BeforeCommit(ctx context.Context, tx data.Transaction) {
	if !l.isEnabled {
		return
	}

	// 从上下文获取用户信息（简化处理）
	userID := "demo_user"

	log.Printf("[%s] 事务即将提交 - ID:%d, 用户:%s, 层级:%d",
		l.logLevel, tx.ID(), userID, tx.Level())
}

// AfterCommit 提交后钩子
func (l *TransactionLogger) AfterCommit(ctx context.Context, tx data.Transaction) {
	if !l.isEnabled {
		return
	}

	// 记录事务执行时间和其他指标
	if startTime, ok := ctx.Value("tx_start_time").(time.Time); ok {
		duration := time.Since(startTime)
		log.Printf("[%s] 事务已提交 - ID:%d, 状态:%s, 耗时:%v",
			l.logLevel, tx.ID(), tx.Status(), duration)
	}
}

// BeforeRollback 回滚前钩子
func (l *TransactionLogger) BeforeRollback(ctx context.Context, tx data.Transaction) {
	if !l.isEnabled {
		return
	}

	// 获取错误信息
	errMsg := "未知错误"
	if err, ok := ctx.Value("tx_error").(error); ok {
		errMsg = err.Error()
	}

	log.Printf("[%s] 事务即将回滚 - ID:%d, 错误:%s",
		l.logLevel, tx.ID(), errMsg)
}

// AfterRollback 回滚后钩子
func (l *TransactionLogger) AfterRollback(ctx context.Context, tx data.Transaction) {
	if !l.isEnabled {
		return
	}

	log.Printf("[%s] 事务已回滚 - ID:%d, 状态:%s",
		l.logLevel, tx.ID(), tx.Status())
}

// EnhancedTransactionTemplate 增强的事务模板
func EnhancedTransactionTemplate(
	ctx context.Context,
	session data.DBSession,
	fn func(ctx context.Context) (interface{}, error),
	logger *TransactionLogger,
) (interface{}, error) {
	// 记录开始时间
	startCtx := context.WithValue(ctx, "tx_start_time", time.Now())

	// 使用 TransactionTemplateV2 并注册事务钩子
	return data.TransactionTemplateV2(
		startCtx,
		session,
		func(txCtx context.Context) (interface{}, error) {
			// 执行业务逻辑
			result, err := fn(txCtx)
			// 如有错误，记录到上下文
			if err != nil {
				txCtx = context.WithValue(txCtx, "tx_error", err)
			}
			return result, err
		},
		logger.BeforeCommit,
		logger.AfterCommit,
		logger.BeforeRollback,
		logger.AfterRollback,
		func(format string, args ...interface{}) {
			if logger.isEnabled {
				log.Printf("[TRACE] "+format, args...)
			}
		},
	)
}

// 示例：结合日志和AOP的事务方法
// frame:transaction(propagation="REQUIRED")
func CreateOrder(ctx context.Context, db data.DBSession, order Order) (Order, error) {
	// 实现创建订单的逻辑...
	return order, nil
}

// Order 订单模型
type Order struct {
	ID     int64
	UserID int64
	Amount float64
	Status string
}

// DemoTransactionWithLogging 演示带日志的事务
func DemoTransactionWithLogging(ctx context.Context, db data.DBSession) {
	// 创建事务日志记录器
	txLogger := NewTransactionLogger(true, "INFO")

	// 准备订单数据
	order := Order{
		UserID: 1001,
		Amount: 199.99,
		Status: "待支付",
	}

	// 使用增强的事务模板
	result, err := EnhancedTransactionTemplate(
		ctx,
		db,
		func(txCtx context.Context) (interface{}, error) {
			// 创建订单
			savedOrder, err := CreateOrder(txCtx, db, order)
			if err != nil {
				return nil, fmt.Errorf("创建订单失败: %w", err)
			}

			// 模拟其他操作，如库存扣减等
			log.Printf("订单创建成功，ID: %d", savedOrder.ID)

			return savedOrder, nil
		},
		txLogger,
	)

	if err != nil {
		log.Printf("订单处理失败: %v", err)
		return
	}

	savedOrder := result.(Order)
	log.Printf("订单处理完成，订单ID: %d, 状态: %s", savedOrder.ID, savedOrder.Status)
}
