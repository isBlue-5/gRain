package repositories

import (
	"context"
	"fmt"
	"time"

	"complete_demo/models"

	"gorm.io/gorm"
)

// UserRepository 用户仓库接口
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*models.User, error)
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindAll(ctx context.Context) ([]*models.User, error)
	FindByConditions(ctx context.Context, filters map[string]interface{}, search string, orderBy string, limit, offset int) ([]*models.User, int64, error)
	Count(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
	CountByDateRange(ctx context.Context, start, end time.Time) (int64, error)
}

// DefaultUserRepository 默认用户仓库实现
// frame:repository
type DefaultUserRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓库
func NewUserRepository(db *gorm.DB) UserRepository {
	return &DefaultUserRepository{db: db}
}

// Create 创建用户
// frame:transaction
func (r *DefaultUserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update 更新用户
// frame:transaction
func (r *DefaultUserRepository) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete 删除用户
// frame:transaction
func (r *DefaultUserRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.User{}, id).Error
}

// FindByID 根据ID查找用户
// frame:transaction(readOnly=true)
func (r *DefaultUserRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByUsername 根据用户名查找用户
// frame:transaction(readOnly=true)
func (r *DefaultUserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail 根据邮箱查找用户
// frame:transaction(readOnly=true)
func (r *DefaultUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindAll 查找所有用户
// frame:transaction(readOnly=true)
func (r *DefaultUserRepository) FindAll(ctx context.Context) ([]*models.User, error) {
	var users []*models.User
	err := r.db.WithContext(ctx).Find(&users).Error
	return users, err
}

// FindByConditions 根据条件查找用户
// frame:transaction(readOnly=true)
func (r *DefaultUserRepository) FindByConditions(ctx context.Context, filters map[string]interface{}, search string, orderBy string, limit, offset int) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	// 构建查询
	query := r.db.WithContext(ctx).Model(&models.User{})

	// 应用过滤器
	for key, value := range filters {
		if value != "" {
			query = query.Where(key+" = ?", value)
		}
	}

	// 应用搜索
	if search != "" {
		searchQuery := fmt.Sprintf("username LIKE '%%%s%%' OR email LIKE '%%%s%%' OR first_name LIKE '%%%s%%' OR last_name LIKE '%%%s%%'", search, search, search, search)
		query = query.Where(searchQuery)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 应用排序和分页
	if orderBy != "" {
		query = query.Order(orderBy)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// 执行查询
	err := query.Find(&users).Error
	return users, total, err
}

// Count 统计用户总数
// frame:transaction(readOnly=true)
func (r *DefaultUserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error
	return count, err
}

// CountByStatus 根据状态统计用户数
// frame:transaction(readOnly=true)
func (r *DefaultUserRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

// CountByDateRange 根据日期范围统计用户数
// frame:transaction(readOnly=true)
func (r *DefaultUserRepository) CountByDateRange(ctx context.Context, start, end time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("created_at BETWEEN ? AND ?", start, end).Count(&count).Error
	return count, err
}

// FindByRole 根据角色查找用户
// frame:transaction(readOnly=true)
func (r *DefaultUserRepository) FindByRole(ctx context.Context, role string) ([]*models.User, error) {
	var users []*models.User
	err := r.db.WithContext(ctx).Where("role = ?", role).Find(&users).Error
	return users, err
}

// FindActiveUsers 查找活跃用户
// frame:transaction(readOnly=true)
func (r *DefaultUserRepository) FindActiveUsers(ctx context.Context) ([]*models.User, error) {
	var users []*models.User
	err := r.db.WithContext(ctx).Where("status = ?", "ACTIVE").Find(&users).Error
	return users, err
}

// FindUsersByCreatedDate 根据创建日期查找用户
// frame:transaction(readOnly=true)
func (r *DefaultUserRepository) FindUsersByCreatedDate(ctx context.Context, date time.Time) ([]*models.User, error) {
	var users []*models.User
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	err := r.db.WithContext(ctx).Where("created_at BETWEEN ? AND ?", startOfDay, endOfDay).Find(&users).Error
	return users, err
}

// UpdateUserStatus 更新用户状态
// frame:transaction
func (r *DefaultUserRepository) UpdateUserStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateUserRole 更新用户角色
// frame:transaction
func (r *DefaultUserRepository) UpdateUserRole(ctx context.Context, id uint, role string) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Update("role", role).Error
}

// BulkUpdateUsers 批量更新用户
// frame:transaction
func (r *DefaultUserRepository) BulkUpdateUsers(ctx context.Context, ids []uint, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id IN ?", ids).Updates(updates).Error
}

// GetUserStats 获取用户统计信息
// frame:transaction(readOnly=true)
func (r *DefaultUserRepository) GetUserStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总用户数
	var totalCount int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}
	stats["total_users"] = totalCount

	// 按角色统计
	var roleStats []struct {
		Role  string `json:"role"`
		Count int64  `json:"count"`
	}
	if err := r.db.WithContext(ctx).Model(&models.User{}).Select("role, count(*) as count").Group("role").Scan(&roleStats).Error; err != nil {
		return nil, err
	}
	stats["role_stats"] = roleStats

	// 按状态统计
	var statusStats []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	if err := r.db.WithContext(ctx).Model(&models.User{}).Select("status, count(*) as count").Group("status").Scan(&statusStats).Error; err != nil {
		return nil, err
	}
	stats["status_stats"] = statusStats

	// 今日新增用户数
	today := time.Now().Truncate(24 * time.Hour)
	var todayCount int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Where("created_at >= ?", today).Count(&todayCount).Error; err != nil {
		return nil, err
	}
	stats["today_new_users"] = todayCount

	return stats, nil
}
