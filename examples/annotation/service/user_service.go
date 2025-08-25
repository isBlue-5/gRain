package service

import (
	"errors"
	"fmt"
	"sync"

	"github.com/grain-framework/grain/examples/annotation/controller"
)

// UserRepository 用户存储接口
type UserRepository interface {
	FindByID(id string) (controller.User, error)
	Save(user controller.User) (string, error)
	FindAll() ([]controller.User, error)
}

// frame:service
type UserServiceImpl struct {
	// 通过inject注解标记依赖注入
	Repository UserRepository `inject:""`
}

// 确保UserServiceImpl实现了UserService接口
var _ controller.UserService = (*UserServiceImpl)(nil)

// GetByID 根据ID获取用户
// frame:log(level="DEBUG")
// frame:transaction(readOnly=true)
func (s *UserServiceImpl) GetByID(id string) (controller.User, error) {
	return s.Repository.FindByID(id)
}

// Create 创建新用户
// frame:transaction(readOnly=false)
// frame:validation(rules={"Name": "required,min=2", "Age": "required,min=18"})
func (s *UserServiceImpl) Create(user controller.User) (string, error) {
	return s.Repository.Save(user)
}

// List 获取所有用户
// frame:transaction(readOnly=true)
// frame:cache(ttl="10m", key="all_users")
func (s *UserServiceImpl) List() ([]controller.User, error) {
	return s.Repository.FindAll()
}

// InMemoryUserRepository 内存用户存储实现
// frame:repository
type InMemoryUserRepository struct {
	users  map[string]controller.User
	nextID int
	mutex  sync.RWMutex
}

// NewInMemoryUserRepository 创建内存用户存储
func NewInMemoryUserRepository() *InMemoryUserRepository {
	// 初始化一些测试数据
	repo := &InMemoryUserRepository{
		users:  make(map[string]controller.User),
		nextID: 1,
	}

	// 添加示例用户
	repo.users["1"] = controller.User{ID: "1", Name: "张三", Age: 30}
	repo.users["2"] = controller.User{ID: "2", Name: "李四", Age: 25}
	repo.users["3"] = controller.User{ID: "3", Name: "王五", Age: 28}

	repo.nextID = 4

	return repo
}

// FindByID 根据ID查找用户
func (r *InMemoryUserRepository) FindByID(id string) (controller.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if user, ok := r.users[id]; ok {
		return user, nil
	}

	return controller.User{}, errors.New("用户不存在")
}

// Save 保存用户
func (r *InMemoryUserRepository) Save(user controller.User) (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 如果没有ID，分配一个新ID
	if user.ID == "" {
		user.ID = fmt.Sprintf("%d", r.nextID)
		r.nextID++
	}

	r.users[user.ID] = user
	return user.ID, nil
}

// FindAll 获取所有用户
func (r *InMemoryUserRepository) FindAll() ([]controller.User, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	users := make([]controller.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}

	return users, nil
}
