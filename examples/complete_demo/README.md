# gRain 完整示例项目

这是一个完整的 gRain 框架示例项目，展示了框架的所有核心注解功能。

## 项目特性

- ✅ **控制器注解**：自动路由注册
- ✅ **实体注解**：自动生成 GORM 模型和仓库
- ✅ **服务注解**：自动生成服务层代码
- ✅ **权限控制**：基于角色的访问控制
- ✅ **限流功能**：请求频率限制
- ✅ **事务管理**：自动事务处理
- ✅ **日志记录**：结构化日志
- ✅ **缓存支持**：Redis 缓存集成
- ✅ **验证绑定**：请求数据验证

## 项目结构

```
complete_demo/
├── README.md                 # 项目说明
├── go.mod                   # Go 模块文件
├── go.sum                   # 依赖版本锁定
├── main.go                  # 主程序入口
├── config/                  # 配置管理
│   └── config.go
├── models/                  # 数据模型
│   ├── user.go             # 用户模型
│   └── product.go          # 产品模型
├── controllers/             # 控制器层
│   ├── user_controller.go  # 用户控制器
│   └── product_controller.go # 产品控制器
├── services/                # 服务层
│   ├── user_service.go     # 用户服务
│   └── product_service.go  # 产品服务
├── repositories/            # 仓库层
│   ├── user_repository.go  # 用户仓库
│   └── product_repository.go # 产品仓库
├── middleware/              # 中间件
│   ├── auth.go             # 认证中间件
│   └── cors.go             # CORS 中间件
├── database/                # 数据库配置
│   └── database.go
├── utils/                   # 工具函数
│   └── response.go
└── generated/               # 生成的代码（自动生成）
    ├── routes/
    ├── services/
    ├── repositories/
    └── middleware/
```

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 配置数据库

```bash
# 创建 SQLite 数据库（开发环境）
sqlite3 demo.db < database/schema.sql

# 或者使用 MySQL/PostgreSQL
# 修改 config/config.go 中的数据库配置
```

### 3. 生成代码

```bash
# 使用 gRain 代码生成器
go run cmd/ginframe-gen/main.go -input . -output ./generated
```

### 4. 运行项目

```bash
go run main.go
```

### 5. 测试 API

```bash
# 创建用户
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","email":"admin@example.com","password":"password123"}'

# 获取用户列表
curl http://localhost:8080/api/users

# 创建产品
curl -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -d '{"name":"测试产品","price":99.99,"description":"这是一个测试产品"}'
```

## 注解使用示例

### 控制器注解

```go
// frame:controller(name="UserController")
type UserController struct {
    userService *UserService
}

// frame:route(method="GET", path="/users")
// frame:auth(roles={"ADMIN", "USER"})
// frame:rateLimit(limit=100, period="1m")
func (c *UserController) GetUsers(ctx *gin.Context) {
    // 自动权限检查
    // 自动限流控制
    // 自动路由注册
}
```

### 实体注解

```go
// frame:entity(table="users")
type User struct {
    ID       uint      `json:"id" gorm:"primaryKey"`
    Username string    `json:"username" gorm:"uniqueIndex;not null"`
    Email    string    `json:"email" gorm:"uniqueIndex;not null"`
    Password string    `json:"-" gorm:"not null"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 服务注解

```go
// frame:service
type UserService struct {
    userRepo *UserRepository
}

// frame:transaction
func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    // 自动事务管理
    return s.userRepo.Create(ctx, user)
}
```

### 仓库注解

```go
// frame:repository
type UserRepository struct {
    db *gorm.DB
}

// frame:transaction(readOnly=true)
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*User, error) {
    // 自动只读事务
    var user User
    return &user, r.db.WithContext(ctx).First(&user, id).Error
}
```

## 功能特性

### 1. 自动路由注册
- 基于注解自动生成路由配置
- 支持 RESTful API 设计
- 自动中间件集成

### 2. 权限控制
- 基于角色的访问控制 (RBAC)
- 细粒度权限管理
- JWT 令牌认证

### 3. 限流保护
- 基于 IP 的请求限流
- 可配置的限流策略
- 支持多种限流算法

### 4. 事务管理
- 自动事务边界管理
- 支持嵌套事务
- 自动回滚机制

### 5. 缓存支持
- Redis 缓存集成
- 自动缓存失效
- 缓存穿透保护

### 6. 数据验证
- 请求数据自动验证
- 自定义验证规则
- 错误信息本地化

## 性能特性

- **并发处理**：支持高并发请求
- **连接池**：数据库连接池管理
- **缓存优化**：多级缓存策略
- **异步处理**：非阻塞 I/O 操作

## 安全特性

- **SQL 注入防护**：参数化查询
- **XSS 防护**：输入输出过滤
- **CSRF 防护**：跨站请求伪造防护
- **权限验证**：严格的访问控制

## 监控和日志

- **结构化日志**：JSON 格式日志输出
- **性能监控**：请求响应时间统计
- **错误追踪**：详细的错误堆栈信息
- **健康检查**：系统状态监控

## 部署说明

### 开发环境

```bash
# 使用 SQLite 数据库
export DB_TYPE=sqlite
export DB_PATH=./demo.db
go run main.go
```

### 生产环境

```bash
# 使用 MySQL 数据库
export DB_TYPE=mysql
export DB_HOST=localhost
export DB_PORT=3306
export DB_NAME=demo
export DB_USER=root
export DB_PASSWORD=password

# 使用 Redis 缓存
export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_PASSWORD=

# 启动应用
./demo
```

## 故障排除

### 常见问题

1. **数据库连接失败**
   - 检查数据库服务是否启动
   - 验证连接参数是否正确
   - 确认数据库用户权限

2. **代码生成失败**
   - 检查注解语法是否正确
   - 确认 gRain 工具版本
   - 查看错误日志信息

3. **权限验证失败**
   - 检查 JWT 令牌是否有效
   - 确认用户角色配置
   - 验证权限表达式

### 调试模式

```bash
# 启用调试日志
export LOG_LEVEL=debug
export GRAIN_DEBUG=true

# 启动应用
go run main.go
```

## 贡献指南

欢迎提交 Issue 和 Pull Request 来改进这个示例项目。

## 许可证

MIT License
