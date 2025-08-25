// Package transaction 演示事务注解的使用示例
package transaction

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/isBlue-5/grain/pkg/data"
)

// User 用户实体
type User struct {
	ID       int64  `db:"id"`
	Username string `db:"username"`
	Email    string `db:"email"`
}

// UserService 用户服务
// frame:inject
type UserService struct {
	// 数据库会话，被依赖注入框架自动注入
	db data.DBSession `inject:""`
}

// CreateUser 创建用户
// frame:transaction(timeout="5s", rollbackFor={"*errors.Error", "*ValidationError"})
func (s *UserService) CreateUser(ctx context.Context, user *User) (*User, error) {
	if user.Username == "" {
		return nil, &ValidationError{Message: "用户名不能为空"}
	}

	// 使用上下文中的事务
	tx, err := data.RequireTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取事务失败: %w", err)
	}

	result, err := tx.Exec(
		ctx,
		"INSERT INTO users (username, email) VALUES (?, ?)",
		user.Username, user.Email,
	)
	if err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取插入ID失败: %w", err)
	}

	user.ID = id
	return user, nil
}

// TransferCredits 在用户之间转移积分，演示事务的使用
// frame:transaction(propagation="REQUIRED")
func (s *UserService) TransferCredits(ctx context.Context, fromUserID, toUserID int64, amount float64) error {
	// 检查是否有足够的积分
	tx, err := data.RequireTransaction(ctx)
	if err != nil {
		return fmt.Errorf("获取事务失败: %w", err)
	}

	var balance float64
	err = tx.Query(
		ctx,
		&balance,
		"SELECT credit_balance FROM user_credits WHERE user_id = ?",
		fromUserID,
	)
	if err != nil {
		return fmt.Errorf("查询用户积分失败: %w", err)
	}

	if balance < amount {
		return &ValidationError{Message: "用户积分不足"}
	}

	// 减少发送方积分
	_, err = tx.Exec(
		ctx,
		"UPDATE user_credits SET credit_balance = credit_balance - ? WHERE user_id = ?",
		amount, fromUserID,
	)
	if err != nil {
		return fmt.Errorf("减少发送方积分失败: %w", err)
	}

	// 增加接收方积分
	_, err = tx.Exec(
		ctx,
		"UPDATE user_credits SET credit_balance = credit_balance + ? WHERE user_id = ?",
		amount, toUserID,
	)
	if err != nil {
		return fmt.Errorf("增加接收方积分失败: %w", err)
	}

	return nil
}

// ReadOnlyMethod 演示只读事务
// frame:transaction(readOnly=true)
func (s *UserService) ReadOnlyMethod(ctx context.Context, userID int64) (*User, error) {
	tx, err := data.RequireTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取事务失败: %w", err)
	}

	user := &User{}
	err = tx.Query(
		ctx,
		user,
		"SELECT id, username, email FROM users WHERE id = ?",
		userID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &NotFoundError{Message: "用户不存在"}
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	return user, nil
}

// InnerRequiresNew 演示 REQUIRES_NEW 传播行为
// frame:transaction(propagation="REQUIRES_NEW")
func (s *UserService) InnerRequiresNew(ctx context.Context, userID int64) error {
	tx, err := data.RequireTransaction(ctx)
	if err != nil {
		return fmt.Errorf("[REQUIRES_NEW] 获取事务失败: %w", err)
	}
	fmt.Printf("[REQUIRES_NEW] 事务ID: %d, 层级: %d, 状态: %s\n", tx.ID(), tx.Level(), tx.Status())
	// 这里可以做简单的更新或插入
	return nil
}

// InnerSupports 演示 SUPPORTS 传播行为
// frame:transaction(propagation="SUPPORTS")
func (s *UserService) InnerSupports(ctx context.Context, userID int64) error {
	if tx, ok := data.GetTransaction(ctx); ok {
		fmt.Printf("[SUPPORTS] 事务ID: %d, 层级: %d, 状态: %s (已存在事务)\n", tx.ID(), tx.Level(), tx.Status())
	} else {
		fmt.Println("[SUPPORTS] 当前无事务")
	}
	return nil
}

// InnerNotSupported 演示 NOT_SUPPORTED 传播行为
// frame:transaction(propagation="NOT_SUPPORTED")
func (s *UserService) InnerNotSupported(ctx context.Context, userID int64) error {
	if tx, ok := data.GetTransaction(ctx); ok {
		fmt.Printf("[NOT_SUPPORTED] 事务ID: %d, 层级: %d, 状态: %s (应被挂起)\n", tx.ID(), tx.Level(), tx.Status())
	} else {
		fmt.Println("[NOT_SUPPORTED] 当前无事务 (预期)")
	}
	return nil
}

// InnerMandatory 演示 MANDATORY 传播行为
// frame:transaction(propagation="MANDATORY")
func (s *UserService) InnerMandatory(ctx context.Context, userID int64) error {
	tx, err := data.RequireTransaction(ctx)
	if err != nil {
		return fmt.Errorf("[MANDATORY] 必须有事务: %w", err)
	}
	fmt.Printf("[MANDATORY] 事务ID: %d, 层级: %d, 状态: %s\n", tx.ID(), tx.Level(), tx.Status())
	return nil
}

// InnerNested 演示 NESTED 传播行为
// frame:transaction(propagation="NESTED")
func (s *UserService) InnerNested(ctx context.Context, userID int64) error {
	tx, err := data.RequireTransaction(ctx)
	if err != nil {
		return fmt.Errorf("[NESTED] 获取事务失败: %w", err)
	}
	fmt.Printf("[NESTED] 事务ID: %d, 层级: %d, 状态: %s (嵌套事务)\n", tx.ID(), tx.Level(), tx.Status())
	return nil
}

// ValidationError 表示验证错误
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NotFoundError 表示资源未找到错误
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}
