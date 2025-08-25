// Package gorm 演示GORM适配器的使用示例
package gorm

import (
	"context"
	"fmt"
)

// UserService 用户服务接口
type UserService interface {
	GetUser(ctx context.Context, id uint) (*User, error)
	GetAllUsers(ctx context.Context) ([]*User, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id uint) error
}

// UserServiceImpl 用户服务实现
// frame:inject
type UserServiceImpl struct {
	// 注入用户仓库
	repo UserRepository `inject:""`
}

// GetUser 获取单个用户
// frame:transaction(readOnly=true)
func (s *UserServiceImpl) GetUser(ctx context.Context, id uint) (*User, error) {
	// 这里会自动开启只读事务
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %w", err)
	}
	return user, nil
}

// GetAllUsers 获取所有用户
// frame:transaction(readOnly=true)
func (s *UserServiceImpl) GetAllUsers(ctx context.Context) ([]*User, error) {
	// 这里会自动开启只读事务
	users, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取所有用户失败: %w", err)
	}
	return users, nil
}

// CreateUser 创建用户
// frame:transaction
func (s *UserServiceImpl) CreateUser(ctx context.Context, user *User) error {
	// 业务规则验证
	if user.Username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if user.Email == "" {
		return fmt.Errorf("邮箱不能为空")
	}

	// 这里会自动开启事务
	if err := s.repo.Create(ctx, user); err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}
	return nil
}

// UpdateUser 更新用户
// frame:transaction
func (s *UserServiceImpl) UpdateUser(ctx context.Context, user *User) error {
	// 业务规则验证
	if user.ID == 0 {
		return fmt.Errorf("用户ID不能为空")
	}
	if user.Username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if user.Email == "" {
		return fmt.Errorf("邮箱不能为空")
	}

	// 检查用户是否存在
	existingUser, err := s.repo.FindByID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if existingUser == nil {
		return fmt.Errorf("用户不存在")
	}

	// 这里会自动开启事务
	if err := s.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("更新用户失败: %w", err)
	}
	return nil
}

// DeleteUser 删除用户
// frame:transaction
func (s *UserServiceImpl) DeleteUser(ctx context.Context, id uint) error {
	// 检查用户是否存在
	existingUser, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if existingUser == nil {
		return fmt.Errorf("用户不存在")
	}

	// 这里会自动开启事务
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}
	return nil
}

// NewUserService 创建用户服务
func NewUserService(repo UserRepository) UserService {
	return &UserServiceImpl{repo: repo}
}
