// Package query 提供自定义查询注解示例
package query

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// User 用户实体
// frame:entity(table="users")
type User struct {
	ID        uint      `db:"id,primaryKey" json:"id"`
	Username  string    `db:"username,unique" json:"username" validate:"required,min=3"`
	Email     string    `db:"email" json:"email" validate:"required,email"`
	Age       int       `db:"age" json:"age" validate:"gte=0"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// UserRepository 用户仓库接口
type UserRepository interface {
	// 基本CRUD方法
	FindByID(ctx context.Context, id uint) (*User, error)
	FindAll(ctx context.Context) ([]*User, error)
	Save(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error

	// 自定义查询方法
	// frame:query(entity="User", type="findBy")
	FindByUsername(ctx context.Context, username string) (*User, error)

	// frame:query(entity="User", type="findBy", errorHandling="nil")
	FindByEmail(ctx context.Context, email string) (*User, error)

	// frame:query(entity="User", type="findAllBy", pagination=true)
	FindAllByStatus(ctx context.Context, status string, limit int, offset int) ([]*User, error)

	// frame:query(entity="User", type="findAllBy")
	FindAllByAgeGreaterThan(ctx context.Context, age int) ([]*User, error)

	// frame:query(entity="User", type="findAllBy")
	FindAllByAgeBetween(ctx context.Context, ageStart int, ageEnd int) ([]*User, error)

	// frame:query(entity="User", type="countBy")
	CountByStatus(ctx context.Context, status string) (int64, error)

	// frame:query(entity="User", type="existsBy")
	ExistsByUsername(ctx context.Context, username string) (bool, error)

	// frame:query(entity="User", type="deleteBy")
	DeleteByStatus(ctx context.Context, status string) (int64, error)
}

// DefaultUserRepository 默认用户仓库实现
type DefaultUserRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓库
func NewUserRepository(db *gorm.DB) UserRepository {
	return &DefaultUserRepository{
		db: db,
	}
}

// FindByID 根据ID查找用户
func (r *DefaultUserRepository) FindByID(ctx context.Context, id uint) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindAll 查找所有用户
func (r *DefaultUserRepository) FindAll(ctx context.Context) ([]*User, error) {
	var users []*User
	err := r.db.WithContext(ctx).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

// Save 保存用户
func (r *DefaultUserRepository) Save(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update 更新用户
func (r *DefaultUserRepository) Update(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete 删除用户
func (r *DefaultUserRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&User{}, id).Error
}

// 注意：以下方法将由查询注解处理器自动生成
// 这里只是为了演示而手动实现

// FindByUsername 根据用户名查找用户
func (r *DefaultUserRepository) FindByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail 根据邮箱查找用户
func (r *DefaultUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, nil // 注意这里返回nil, nil，而不是错误
	}
	return &user, nil
}

// FindAllByStatus 根据状态查找用户
func (r *DefaultUserRepository) FindAllByStatus(ctx context.Context, status string, limit int, offset int) ([]*User, error) {
	var users []*User
	err := r.db.WithContext(ctx).Where("status = ?", status).Limit(limit).Offset(offset).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

// FindAllByAgeGreaterThan 查找年龄大于指定值的用户
func (r *DefaultUserRepository) FindAllByAgeGreaterThan(ctx context.Context, age int) ([]*User, error) {
	var users []*User
	err := r.db.WithContext(ctx).Where("age > ?", age).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

// FindAllByAgeBetween 查找年龄在指定范围的用户
func (r *DefaultUserRepository) FindAllByAgeBetween(ctx context.Context, ageStart int, ageEnd int) ([]*User, error) {
	var users []*User
	err := r.db.WithContext(ctx).Where("age BETWEEN ? AND ?", ageStart, ageEnd).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

// CountByStatus 统计指定状态的用户数量
func (r *DefaultUserRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&User{}).Where("status = ?", status).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ExistsByUsername 检查用户名是否存在
func (r *DefaultUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&User{}).Where("username = ?", username).Limit(1).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DeleteByStatus 删除指定状态的用户
func (r *DefaultUserRepository) DeleteByStatus(ctx context.Context, status string) (int64, error) {
	result := r.db.WithContext(ctx).Where("status = ?", status).Delete(&User{})
	return result.RowsAffected, result.Error
}
