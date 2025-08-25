// Package cache 提供缓存注解示例
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/isBlue-5/grain/pkg/cache"
)

// User 用户模型
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
}

// UserService 用户服务
type UserService struct {
	// 用户数据存储（模拟数据库）
	users map[int64]*User
	// 缓存管理器
	cacheManager cache.CacheProvider
}

// NewUserService 创建用户服务
func NewUserService() *UserService {
	// 初始化一些测试数据
	users := make(map[int64]*User)
	users[1] = &User{ID: 1, Username: "admin", Email: "admin@example.com", Age: 30}
	users[2] = &User{ID: 2, Username: "user", Email: "user@example.com", Age: 25}
	users[3] = &User{ID: 3, Username: "test", Email: "test@example.com", Age: 20}

	return &UserService{
		users:        users,
		cacheManager: cache.DefaultCacheManager,
	}
}

// frame:cacheable(name="users", key="user", ttl="5*time.Minute")
// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(ctx context.Context, id int64) (*User, error) {
	fmt.Println("从数据库中获取用户:", id) // 模拟数据库访问日志

	// 模拟数据库延迟
	time.Sleep(100 * time.Millisecond)

	user, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("用户不存在: %d", id)
	}

	return user, nil
}

// frame:cacheable(name="users", key="users:by-username", keyParams={"username"})
// GetUserByUsername 根据用户名获取用户
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	fmt.Println("从数据库中获取用户:", username) // 模拟数据库访问日志

	// 模拟数据库延迟
	time.Sleep(100 * time.Millisecond)

	for _, user := range s.users {
		if user.Username == username {
			return user, nil
		}
	}

	return nil, fmt.Errorf("用户不存在: %s", username)
}

// frame:cacheable(name="users", key="users:list", condition="shouldCache")
// GetAllUsers 获取所有用户
func (s *UserService) GetAllUsers(ctx context.Context) ([]*User, error) {
	fmt.Println("从数据库中获取所有用户") // 模拟数据库访问日志

	// 模拟数据库延迟
	time.Sleep(200 * time.Millisecond)

	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}

	return users, nil
}

// shouldCache 判断是否应该缓存结果
func (s *UserService) shouldCache(ctx context.Context) bool {
	// 这里可以实现自定义的缓存条件逻辑
	// 例如，只在工作时间缓存，或者根据用户权限决定是否缓存
	return len(s.users) > 0 // 示例：只有当有用户数据时才缓存
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(ctx context.Context, user *User) error {
	if user == nil {
		return fmt.Errorf("用户不能为空")
	}

	if _, ok := s.users[user.ID]; !ok {
		return fmt.Errorf("用户不存在: %d", user.ID)
	}

	// 更新用户
	s.users[user.ID] = user

	// 注意：以下方法由缓存注解处理器自动生成
	// 手动清除缓存
	s.evictGetUserByID(user.ID)
	s.evictGetUserByUsername(user.Username)
	s.evictGetAllUsers()

	return nil
}

// CreateUser 创建新用户
func (s *UserService) CreateUser(ctx context.Context, user *User) error {
	if user == nil {
		return fmt.Errorf("用户不能为空")
	}

	if user.ID == 0 {
		// 生成新ID
		maxID := int64(0)
		for id := range s.users {
			if id > maxID {
				maxID = id
			}
		}
		user.ID = maxID + 1
	}

	// 检查用户名是否已存在
	for _, existingUser := range s.users {
		if existingUser.Username == user.Username {
			return fmt.Errorf("用户名已存在: %s", user.Username)
		}
	}

	// 保存用户
	s.users[user.ID] = user

	// 注意：以下方法由缓存注解处理器自动生成
	// 清除列表缓存
	s.evictGetAllUsers()

	return nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(ctx context.Context, id int64) error {
	user, ok := s.users[id]
	if !ok {
		return fmt.Errorf("用户不存在: %d", id)
	}

	// 删除用户
	delete(s.users, id)

	// 注意：以下方法由缓存注解处理器自动生成
	// 清除缓存
	s.evictGetUserByID(id)
	s.evictGetUserByUsername(user.Username)
	s.evictGetAllUsers()

	return nil
}

// 缓存清理方法（通常由注解处理器自动生成）

// evictGetUserByID 清除用户ID缓存
func (s *UserService) evictGetUserByID(id int64) {
	ctx := context.Background()
	key := fmt.Sprintf("users:user:%d", id)
	userCache := s.cacheManager.GetCache("users")
	if userCache != nil {
		userCache.Delete(ctx, key)
	}
}

// evictGetUserByUsername 清除用户名缓存
func (s *UserService) evictGetUserByUsername(username string) {
	ctx := context.Background()
	key := fmt.Sprintf("users:user:%s", username)
	userCache := s.cacheManager.GetCache("users")
	if userCache != nil {
		userCache.Delete(ctx, key)
	}
}

// evictGetAllUsers 清除所有用户缓存
func (s *UserService) evictGetAllUsers() {
	// 由于接口限制，只能清除已知的缓存键
	// 在实际实现中，应该维护一个缓存键列表
	ctx := context.Background()
	userCache := s.cacheManager.GetCache("users")
	if userCache != nil {
		// 清除一些常见的缓存键
		userCache.Delete(ctx, "users:all:*")
	}
}
