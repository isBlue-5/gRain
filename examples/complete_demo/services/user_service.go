package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"complete_demo/config"
	"complete_demo/models"
	"complete_demo/repositories"

	"github.com/golang-jwt/jwt/v5"
)

// UserService 用户服务
// frame:service
type UserService struct {
	userRepo repositories.UserRepository
	config   *config.Config
}

// NewUserService 创建用户服务
func NewUserService(userRepo repositories.UserRepository, config *config.Config) *UserService {
	return &UserService{
		userRepo: userRepo,
		config:   config,
	}
}

// GetUsers 获取用户列表
// frame:transaction(readOnly=true)
func (s *UserService) GetUsers(ctx context.Context, filters map[string]interface{}, search string, page, pageSize int) ([]*models.User, int64, error) {
	return s.userRepo.FindByConditions(ctx, filters, search, "created_at DESC", pageSize, (page-1)*pageSize)
}

// GetUserByID 根据ID获取用户
// frame:transaction(readOnly=true)
func (s *UserService) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

// CreateUser 创建用户
// frame:transaction
func (s *UserService) CreateUser(ctx context.Context, req *models.UserCreateRequest) (*models.User, error) {
	// 检查用户名是否已存在
	existingUser, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("用户名已存在")
	}

	// 检查邮箱是否已存在
	existingUser, err = s.userRepo.FindByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("邮箱已存在")
	}

	// 创建用户
	user := &models.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  s.hashPassword(req.Password),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      req.Role,
		Phone:     req.Phone,
		Address:   req.Address,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return user, nil
}

// UpdateUser 更新用户
// frame:transaction
func (s *UserService) UpdateUser(ctx context.Context, id uint, req *models.UserUpdateRequest) (*models.User, error) {
	// 获取现有用户
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 更新字段
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Status != "" {
		user.Status = req.Status
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Address != "" {
		user.Address = req.Address
	}

	// 保存更新
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	return user, nil
}

// DeleteUser 删除用户
// frame:transaction
func (s *UserService) DeleteUser(ctx context.Context, id uint) error {
	// 检查用户是否存在
	_, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("用户不存在: %w", err)
	}

	// 软删除用户
	return s.userRepo.Delete(ctx, id)
}

// Login 用户登录
// frame:transaction(readOnly=true)
func (s *UserService) Login(ctx context.Context, req *models.UserLoginRequest) (string, *models.User, error) {
	// 查找用户
	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return "", nil, fmt.Errorf("用户名或密码错误")
	}

	// 验证密码
	if !s.verifyPassword(req.Password, user.Password) {
		return "", nil, fmt.Errorf("用户名或密码错误")
	}

	// 检查用户状态
	if !user.IsActive() {
		return "", nil, fmt.Errorf("用户账户已被禁用")
	}

	// 生成JWT令牌
	token, err := s.generateJWT(user)
	if err != nil {
		return "", nil, fmt.Errorf("生成令牌失败: %w", err)
	}

	return token, user, nil
}

// Register 用户注册
// frame:transaction
func (s *UserService) Register(ctx context.Context, req *models.UserCreateRequest) (*models.User, error) {
	// 设置默认角色
	if req.Role == "" {
		req.Role = "USER"
	}

	return s.CreateUser(ctx, req)
}

// ChangePassword 修改密码
// frame:transaction
func (s *UserService) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	// 获取用户
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("用户不存在: %w", err)
	}

	// 验证旧密码
	if !s.verifyPassword(oldPassword, user.Password) {
		return fmt.Errorf("旧密码错误")
	}

	// 更新密码
	user.Password = s.hashPassword(newPassword)
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}

	return nil
}

// ResetPassword 重置密码
// frame:transaction
func (s *UserService) ResetPassword(ctx context.Context, email string) error {
	// 查找用户
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("用户不存在: %w", err)
	}

	// 生成随机密码
	newPassword := s.generateRandomPassword()
	user.Password = s.hashPassword(newPassword)

	// 更新密码
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("重置密码失败: %w", err)
	}

	// 发送邮件通知用户新密码
	// 这里应该实现邮件发送逻辑
	// 可以通过配置的邮件服务发送通知
	if err := s.sendPasswordResetEmail(user.Email, newPassword); err != nil {
		log.Printf("Warning: Failed to send password reset email to %s: %v", user.Email, err)
		// 邮件发送失败不影响密码重置，只记录警告
	}

	return nil
}

// GetUserStats 获取用户统计信息
// frame:transaction(readOnly=true)
func (s *UserService) GetUserStats(ctx context.Context) (map[string]interface{}, error) {
	// 获取总用户数
	totalUsers, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取用户总数失败: %w", err)
	}

	// 获取活跃用户数
	activeUsers, err := s.userRepo.CountByStatus(ctx, "ACTIVE")
	if err != nil {
		return nil, fmt.Errorf("获取活跃用户数失败: %w", err)
	}

	// 获取今日新增用户数
	today := time.Now().Truncate(24 * time.Hour)
	todayUsers, err := s.userRepo.CountByDateRange(ctx, today, time.Now())
	if err != nil {
		return nil, fmt.Errorf("获取今日新增用户数失败: %w", err)
	}

	return map[string]interface{}{
		"total_users":     totalUsers,
		"active_users":    activeUsers,
		"today_new_users": todayUsers,
		"inactive_users":  totalUsers - activeUsers,
	}, nil
}

// hashPassword 哈希密码
func (s *UserService) hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// verifyPassword 验证密码
func (s *UserService) verifyPassword(password, hashedPassword string) bool {
	return s.hashPassword(password) == hashedPassword
}

// generateJWT 生成JWT令牌
func (s *UserService) generateJWT(user *models.User) (string, error) {
	// 创建声明
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(s.config.JWT.Expiration).Unix(),
		"iat":      time.Now().Unix(),
	}

	// 创建令牌
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名令牌
	tokenString, err := token.SignedString([]byte(s.config.JWT.Secret))
	if err != nil {
		return "", fmt.Errorf("签名令牌失败: %w", err)
	}

	return tokenString, nil
}

// sendPasswordResetEmail 发送密码重置邮件
func (s *UserService) sendPasswordResetEmail(email, newPassword string) error {
	// 这里应该实现真实的邮件发送逻辑
	// 可以通过配置的SMTP服务或其他邮件服务提供商发送邮件

	// 示例实现：记录邮件发送日志
	log.Printf("Password reset email sent to %s with new password: %s", email, newPassword)

	// 在实际生产环境中，这里应该：
	// 1. 连接到SMTP服务器
	// 2. 构建邮件内容
	// 3. 发送邮件
	// 4. 处理发送结果

	return nil
}

// generateRandomPassword 生成随机密码
func (s *UserService) generateRandomPassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, 12)
	for i := range password {
		randomByte := make([]byte, 1)
		rand.Read(randomByte)
		password[i] = charset[int(randomByte[0])%len(charset)]
	}
	return string(password)
}
