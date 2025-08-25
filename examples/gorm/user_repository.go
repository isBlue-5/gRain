// Package gorm 演示GORM适配器的使用示例
package gorm

import (
	"context"
	"fmt"
	"time"

	"github.com/isBlue-5/grain/pkg/data"
)

// User 用户实体
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:100;not null;unique" json:"username"`
	Email     string    `gorm:"size:100;not null" json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// UserRepository 用户仓库接口
type UserRepository interface {
	FindByID(ctx context.Context, id uint) (*User, error)
	FindAll(ctx context.Context) ([]*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error
}

// GormUserRepository 基于GORM的用户仓库实现
// frame:inject
type GormUserRepository struct {
	// 使用GORM会话
	db data.DBSession `inject:""`
}

// FindByID 根据ID查询用户
// frame:transaction(readOnly=true)
func (r *GormUserRepository) FindByID(ctx context.Context, id uint) (*User, error) {
	tx, err := data.RequireTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取事务失败: %w", err)
	}

	user := &User{}
	err = tx.Query(ctx, user, "SELECT * FROM users WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	return user, nil
}

// FindAll 查询所有用户
// frame:transaction(readOnly=true)
func (r *GormUserRepository) FindAll(ctx context.Context) ([]*User, error) {
	tx, err := data.RequireTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取事务失败: %w", err)
	}

	var users []*User
	err = tx.Query(ctx, &users, "SELECT * FROM users")
	if err != nil {
		return nil, fmt.Errorf("查询所有用户失败: %w", err)
	}

	return users, nil
}

// Create 创建用户
// frame:transaction
func (r *GormUserRepository) Create(ctx context.Context, user *User) error {
	tx, err := data.RequireTransaction(ctx)
	if err != nil {
		return fmt.Errorf("获取事务失败: %w", err)
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err = tx.Exec(
		ctx,
		"INSERT INTO users (username, email, created_at, updated_at) VALUES (?, ?, ?, ?)",
		user.Username, user.Email, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}

	return nil
}

// Update 更新用户
// frame:transaction
func (r *GormUserRepository) Update(ctx context.Context, user *User) error {
	tx, err := data.RequireTransaction(ctx)
	if err != nil {
		return fmt.Errorf("获取事务失败: %w", err)
	}

	user.UpdatedAt = time.Now()

	_, err = tx.Exec(
		ctx,
		"UPDATE users SET username = ?, email = ?, updated_at = ? WHERE id = ?",
		user.Username, user.Email, user.UpdatedAt, user.ID,
	)
	if err != nil {
		return fmt.Errorf("更新用户失败: %w", err)
	}

	return nil
}

// Delete 删除用户
// frame:transaction
func (r *GormUserRepository) Delete(ctx context.Context, id uint) error {
	tx, err := data.RequireTransaction(ctx)
	if err != nil {
		return fmt.Errorf("获取事务失败: %w", err)
	}

	_, err = tx.Exec(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}

	return nil
}

// NewGormUserRepository 创建基于GORM的用户仓库
func NewGormUserRepository(db data.DBSession) UserRepository {
	return &GormUserRepository{db: db}
}
