package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TestDataFactory 测试数据工厂
// 用于生成各种测试场景的代码，包含完整的类型定义和注解
type TestDataFactory struct {
	tempDir string
	files   []string
}

// NewTestDataFactory 创建新的测试数据工厂
func NewTestDataFactory() *TestDataFactory {
	return &TestDataFactory{
		tempDir: os.TempDir(),
		files:   make([]string, 0),
	}
}

// CreateTestController 创建测试用的控制器代码
func (f *TestDataFactory) CreateTestController(name string) string {
	code := `package test

import "github.com/gin-gonic/gin"

// User 用户类型定义
type User struct {
	ID   uint   ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

// ` + name + `Controller 控制器
// @frame:controller
type ` + name + `Controller struct{}

// Index 首页方法
// @frame:route
func (c *` + name + `Controller) Index(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "Hello from ` + name + `"})
}

// Show 显示方法
// @frame:route
func (c *` + name + `Controller) Show(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.JSON(200, gin.H{"id": id, "message": "Show ` + name + `"})
}

// Store 存储方法
// @frame:route
func (c *` + name + `Controller) Store(ctx *gin.Context) {
	ctx.JSON(201, gin.H{"message": "Created ` + name + `"})
}

// Update 更新方法
// @frame:route
func (c *` + name + `Controller) Update(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.JSON(200, gin.H{"id": id, "message": "Updated ` + name + `"})
}

// Destroy 删除方法
// @frame:route
func (c *` + name + `Controller) Destroy(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.JSON(200, gin.H{"id": id, "message": "Deleted ` + name + `"})
}
`

	filePath := filepath.Join(f.tempDir, fmt.Sprintf("%s_controller.go", strings.ToLower(name)))
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		panic(err)
	}

	f.files = append(f.files, filePath)
	return filePath
}

// CreateTestService 创建测试用的服务代码
func (f *TestDataFactory) CreateTestService(name string) string {
	code := `package test

// User 用户类型定义
type User struct {
	ID   uint   ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

// ` + name + `Service 服务
// @frame:service
type ` + name + `Service struct{}

// DoSomething 执行某些操作
// @frame:method
func (s *` + name + `Service) DoSomething() string {
	return "Hello from ` + name + ` service"
}

// GetUser 获取用户
// @frame:method
func (s *` + name + `Service) GetUser(id uint) *User {
	return &User{ID: id, Name: "Test User"}
}
`

	filePath := filepath.Join(f.tempDir, fmt.Sprintf("%s_service.go", strings.ToLower(name)))
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		panic(err)
	}

	f.files = append(f.files, filePath)
	return filePath
}

// CreateTestEntity 创建测试用的实体代码
func (f *TestDataFactory) CreateTestEntity(name string) string {
	code := `package test

// ` + name + ` 实体
// @frame:entity
type ` + name + ` struct {
	ID   uint   ` + "`json:\"id\" db:\"id,primaryKey\"`" + `
	Name string ` + "`json:\"name\" db:\"name\"`" + `
	Age  int    ` + "`json:\"age\" db:\"age\"`" + `
}
`

	filePath := filepath.Join(f.tempDir, fmt.Sprintf("%s_entity.go", strings.ToLower(name)))
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		panic(err)
	}

	f.files = append(f.files, filePath)
	return filePath
}

// CreateTestRepository 创建测试用的仓库代码
func (f *TestDataFactory) CreateTestRepository(name string) string {
	code := `package test

import "context"

// User 用户类型定义
type User struct {
	ID   uint   ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

// ` + name + `Repository 仓库接口
// @frame:repository
type ` + name + `Repository interface {
	FindByID(ctx context.Context, id uint) (*User, error)
	FindAll(ctx context.Context) ([]*User, error)
	Save(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error
}

// Default` + name + `Repository 默认仓库实现
type Default` + name + `Repository struct{}

// FindByID 根据ID查找
func (r *Default` + name + `Repository) FindByID(ctx context.Context, id uint) (*User, error) {
	return &User{ID: id, Name: "Test User"}, nil
}

// FindAll 查找所有
func (r *Default` + name + `Repository) FindAll(ctx context.Context) ([]*User, error) {
	return []*User{{ID: 1, Name: "User 1"}, {ID: 2, Name: "User 2"}}, nil
}

// Save 保存
func (r *Default` + name + `Repository) Save(ctx context.Context, user *User) error {
	return nil
}

// Update 更新
func (r *Default` + name + `Repository) Update(ctx context.Context, user *User) error {
	return nil
}

// Delete 删除
func (r *Default` + name + `Repository) Delete(ctx context.Context, id uint) error {
	return nil
}
`

	filePath := filepath.Join(f.tempDir, fmt.Sprintf("%s_repository.go", strings.ToLower(name)))
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		panic(err)
	}

	f.files = append(f.files, filePath)
	return filePath
}

// CreateTestWithAuthz 创建带权限控制的测试代码
func (f *TestDataFactory) CreateTestWithAuthz(name string) string {
	code := `package test

import "github.com/gin-gonic/gin"

// @frame:controller
type ` + name + `Controller struct{}

// @frame:route
// @frame:authz(roles=["admin", "manager"], permissions=["read"])
func (c *` + name + `Controller) Index(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "Hello from ` + name + `"})
}

// @frame:route
// @frame:authz(roles=["admin"], permissions=["write"])
func (c *` + name + `Controller) Store(ctx *gin.Context) {
	ctx.JSON(201, gin.H{"message": "Created ` + name + `"})
}

// @frame:route
// @frame:authz(roles=["admin"], permissions=["delete"])
func (c *` + name + `Controller) Destroy(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.JSON(200, gin.H{"id": id, "message": "Deleted ` + name + `"})
}
`

	filePath := filepath.Join(f.tempDir, fmt.Sprintf("%s_authz_controller.go", strings.ToLower(name)))
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		panic(err)
	}

	f.files = append(f.files, filePath)
	return filePath
}

// CreateTestWithRateLimit 创建带限流的测试代码
func (f *TestDataFactory) CreateTestWithRateLimit(name string) string {
	code := `package test

import "github.com/gin-gonic/gin"

// @frame:controller
type ` + name + `Controller struct{}

// @frame:route
// @frame:rateLimit(limit=100, window="1m")
func (c *` + name + `Controller) Index(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "Hello from ` + name + `"})
}

// @frame:route
// @frame:rateLimit(limit=10, window="1m")
func (c *` + name + `Controller) Store(ctx *gin.Context) {
	ctx.JSON(201, gin.H{"message": "Created ` + name + `"})
}

// @frame:route
// @frame:rateLimit(limit=5, window="1m")
func (c *` + name + `Controller) Destroy(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.JSON(200, gin.H{"id": id, "message": "Deleted ` + name + `"})
}
`

	filePath := filepath.Join(f.tempDir, fmt.Sprintf("%s_ratelimit_controller.go", strings.ToLower(name)))
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		panic(err)
	}

	f.files = append(f.files, filePath)
	return filePath
}

// CreateTestWithTransaction 创建带事务的测试代码
func (f *TestDataFactory) CreateTestWithTransaction(name string) string {
	code := `package test

import "github.com/gin-gonic/gin"

// @frame:service
type ` + name + `Service struct{}

// @frame:method
// @frame:transaction
func (s *` + name + `Service) CreateWithTransaction(data string) error {
	// 事务逻辑
	return nil
}

// @frame:method
// @frame:transaction
func (s *` + name + `Service) UpdateWithTransaction(id uint, data string) error {
	// 事务逻辑
	return nil
}
`

	filePath := filepath.Join(f.tempDir, fmt.Sprintf("%s_transaction_service.go", strings.ToLower(name)))
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		panic(err)
	}

	f.files = append(f.files, filePath)
	return filePath
}

// CreateTestWithCache 创建带缓存的测试代码
func (f *TestDataFactory) CreateTestWithCache(name string) string {
	code := `package test

import "github.com/gin-gonic/gin"

// @frame:service
type ` + name + `Service struct{}

// @frame:method
// @frame:cacheable(key="user:` + name + `", ttl="5m")
func (s *` + name + `Service) GetUser(id uint) string {
	return "User data for ID: " + fmt.Sprintf("%%d", id)
}

// @frame:method
// @frame:cacheable(key="users:list", ttl="10m")
func (s *` + name + `Service) GetAllUsers() []string {
	return []string{"User1", "User2", "User3"}
}
`

	filePath := filepath.Join(f.tempDir, fmt.Sprintf("%s_cache_service.go", strings.ToLower(name)))
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		panic(err)
	}

	f.files = append(f.files, filePath)
	return filePath
}

// CreateTestWithLog 创建带日志的测试代码
func (f *TestDataFactory) CreateTestWithLog(name string) string {
	code := `package test

import "github.com/gin-gonic/gin"

// @frame:service
type ` + name + `Service struct{}

// @frame:method
// @frame:log(level="info", message="Processing ` + name + ` request")
func (s *` + name + `Service) ProcessRequest(data string) string {
	return "Processed: " + data
}

// @frame:method
// @frame:log(level="error", message="Error in ` + name + ` service")
func (s *` + name + `Service) HandleError(err error) error {
	return err
}
`

	filePath := filepath.Join(f.tempDir, fmt.Sprintf("%s_log_service.go", strings.ToLower(name)))
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		panic(err)
	}

	f.files = append(f.files, filePath)
	return filePath
}

// CreateTestWithLogging 创建带日志的测试代码（别名方法）
func (f *TestDataFactory) CreateTestWithLogging(name string) string {
	return f.CreateTestWithLog(name)
}

// CreateComplexTestFile 创建复杂的测试文件，包含多种注解
func (f *TestDataFactory) CreateComplexTestFile(name string) string {
	code := `package test

import (
	"github.com/gin-gonic/gin"
	"context"
)

// User 用户类型定义
type User struct {
	ID   uint   ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

// ` + name + ` 复杂实体
// @frame:entity
type ` + name + ` struct {
	ID          uint   ` + "`json:\"id\" db:\"id,primaryKey\"`" + `
	Name        string ` + "`json:\"name\" db:\"name\"`" + `
	Description string ` + "`json:\"description\" db:\"description\"`" + `
	Price       float64 ` + "`json:\"price\" db:\"price\"`" + `
}

// ` + name + `Controller 复杂控制器
// @frame:controller
type ` + name + `Controller struct{}

// Index 首页方法
// @frame:route
// @frame:log(level="info", message="Accessing ` + name + ` index")
func (c *` + name + `Controller) Index(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"message": "Hello from ` + name + `"})
}

// Show 显示方法
// @frame:route
// @frame:authz(roles=["admin", "user"], permissions=["read"])
func (c *` + name + `Controller) Show(ctx *gin.Context) {
	id := ctx.Param("id")
	ctx.JSON(200, gin.H{"id": id, "message": "Show ` + name + `"})
}

// Store 存储方法
// @frame:route
// @frame:authz(roles=["admin"], permissions=["write"])
// @frame:log(level="info", message="Creating ` + name + `")
func (c *` + name + `Controller) Store(ctx *gin.Context) {
	ctx.JSON(201, gin.H{"message": "Created ` + name + `"})
}

// ` + name + `Service 复杂服务
// @frame:service
type ` + name + `Service struct{}

// ProcessData 处理数据
// @frame:method
// @frame:transaction
// @frame:log(level="info", message="Processing ` + name + ` data")
func (s *` + name + `Service) ProcessData(data string) error {
	return nil
}

// ` + name + `Repository 复杂仓库接口
// @frame:repository
type ` + name + `Repository interface {
	FindByID(ctx context.Context, id uint) (*` + name + `, error)
	FindAll(ctx context.Context) ([]*` + name + `, error)
	Save(ctx context.Context, entity *` + name + `) error
}

// Default` + name + `Repository 默认仓库实现
type Default` + name + `Repository struct{}

// FindByID 根据ID查找
func (r *Default` + name + `Repository) FindByID(ctx context.Context, id uint) (*` + name + `, error) {
	return &` + name + `{ID: id, Name: "Test ` + name + `"}, nil
}
`

	filePath := filepath.Join(f.tempDir, fmt.Sprintf("%s_complex.go", strings.ToLower(name)))
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		panic(err)
	}

	f.files = append(f.files, filePath)
	return filePath
}

// Cleanup 清理所有生成的文件
func (f *TestDataFactory) Cleanup() {
	for _, file := range f.files {
		os.Remove(file)
	}
	f.files = f.files[:0]
}
