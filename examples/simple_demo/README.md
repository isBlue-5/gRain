# gRain 框架简单示例 - 开箱即用

这个示例展示了 gRain 框架的"开箱即用"特性。**无需运行任何命令，框架会在应用启动时自动完成所有工作！**

## 🚀 快速开始

### 1. 项目结构
```
simple_demo/
├── main.go                    # 主程序入口
├── controllers/               # 控制器目录
│   └── user_controller.go    # 用户控制器
├── models/                    # 模型目录
│   └── user.go               # 用户模型
└── README.md                 # 说明文档
```

### 2. 运行应用
```bash
# 进入示例目录
cd examples/simple_demo

# 运行应用（就这么简单！）
go run main.go
```

### 3. 访问应用
- **应用信息**: http://localhost:8080/info
- **健康检查**: http://localhost:8080/health
- **Swagger文档**: http://localhost:8080/swagger/index.html
- **API接口**: http://localhost:8080/api/users

## ✨ 框架自动完成的工作

### 🔧 自动代码生成
当应用启动时，框架会自动：

1. **扫描项目目录** - 自动发现控制器和模型
2. **解析注解** - 提取路由、参数、响应等信息
3. **生成依赖注入代码** - 自动创建服务容器
4. **生成路由注册代码** - 自动注册所有API路由
5. **生成Swagger文档** - 自动生成OpenAPI 3.0规范
6. **注册Swagger UI** - 提供交互式API文档

### 🎭 注解驱动开发
只需要在代码中添加注解，框架就会自动处理：

```go
// frame:controller(path="/api/users", tags={"用户管理"})
type UserController struct {
    UserService interface{} `inject:""`
}

// frame:route(method="GET", path="/")
// frame:summary(获取用户列表)
// frame:response(200, []UserResponse, "成功获取用户列表")
func (c *UserController) GetUsers(ctx *gin.Context) {
    // 业务逻辑在这里
    // 框架自动处理路由注册、依赖注入、参数验证等
}
```

### 📚 自动文档生成
框架会自动从代码和注解生成完整的API文档：

- **路由信息** - 自动提取HTTP方法、路径、参数
- **请求模型** - 自动解析结构体字段和验证规则
- **响应模型** - 自动生成响应示例和状态码
- **参数验证** - 自动提取binding标签和验证规则

## 🎯 核心特性

### 1. **零配置启动**
```go
func main() {
    // 创建应用实例 - 就是这么简单！
    app := core.New(
        core.WithPort("8080"),
        core.WithDebug(true),
        core.WithAutoGenerate(true), // 启用自动代码生成
    )

    // 运行应用 - 框架会自动处理一切！
    app.Run()
}
```

### 2. **智能注解解析**
- 支持注释式注解 (`// frame:route(...)`)
- 支持结构体标签注解 (`inject:""`)
- 自动解析Go AST，提取所有信息

### 3. **自动化依赖注入**
- 编译时依赖注入，零反射开销
- 自动发现和注册服务
- 类型安全的服务解析

### 4. **智能路由注册**
- 自动注册所有注解的路由
- 支持路径参数、查询参数
- 自动生成路由分组

### 5. **完整Swagger集成**
- 自动生成OpenAPI 3.0规范
- 集成Swagger UI界面
- 支持实时API测试

## 🔍 生成的代码

框架会在 `generated/` 目录下自动生成以下文件：

- `dependency_injection.go` - 依赖注入容器
- `route_registration.go` - 路由注册代码
- `main_registration.go` - 主注册函数
- `swagger.json` - OpenAPI规范文档

## 📊 性能特点

### 启动时开销
- **代码生成**: 一次性，启动时完成
- **注解解析**: 基于AST，快速高效
- **依赖注入**: 编译时完成，零运行时开销

### 运行时性能
- **路由匹配**: 与原生Gin相同
- **中间件**: 标准Gin中间件
- **请求处理**: 无额外开销

## 🚀 扩展功能

### 1. 添加新控制器
```go
// frame:controller(path="/api/products", tags={"产品管理"})
type ProductController struct {
    ProductService interface{} `inject:""`
}

// frame:route(method="GET", path="/")
// frame:summary(获取产品列表)
func (c *ProductController) GetProducts(ctx *gin.Context) {
    // 实现业务逻辑
}
```

### 2. 添加新模型
```go
type Product struct {
    ID    uint    `json:"id" example:"1"`
    Name  string  `json:"name" binding:"required" example:"iPhone 15"`
    Price float64 `json:"price" binding:"required,gt=0" example:"999.99"`
}
```

### 3. 自定义中间件
```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 实现认证逻辑
        c.Next()
    }
}

// 在main.go中添加
app.AddMiddleware(AuthMiddleware())
```

## 🔧 配置选项

### 应用配置
```go
app := core.New(
    core.WithHost("0.0.0.0"),           // 主机地址
    core.WithPort("8080"),               // 端口
    core.WithMode("debug"),              // 运行模式
    core.WithDebug(true),                // 调试模式
    core.WithShutdownTimeout(30),        // 关闭超时
    core.WithAutoGenerate(true),         // 自动代码生成
)
```

### 环境变量支持
```bash
export GRAIN_HOST=0.0.0.0
export GRAIN_PORT=8080
export GRAIN_MODE=debug
export GRAIN_DEBUG=true
```

## 📚 最佳实践

### 1. 注解组织
```go
// 将相关注解分组，提高可读性
// frame:route(method="POST", path="/")
// frame:summary(创建新用户)
// frame:bind(source="json", model="CreateUserRequest")
// frame:response(201, UserResponse, "用户创建成功")
// frame:response(400, ValidationError, "请求参数验证失败")
func (c *UserController) CreateUser(ctx *gin.Context) {
    // 实现逻辑
}
```

### 2. 模型设计
```go
// 使用清晰的字段命名
// 添加适当的验证规则
// 提供有意义的示例值
type CreateUserRequest struct {
    Username string `json:"username" binding:"required,min=3,max=50" example:"johndoe"`
    Email    string `json:"email" binding:"required,email" example:"john@example.com"`
    Age      int    `json:"age" binding:"gte=0,lte=150" example:"25"`
}
```

### 3. 错误处理
```go
// 定义清晰的错误响应
// frame:response(400, ValidationError, "请求参数验证失败")
// frame:response(404, ErrorResponse, "用户不存在")
// frame:response(500, ErrorResponse, "服务器内部错误")
```

## 🎉 总结

gRain 框架的"开箱即用"特性让您：

1. **专注业务逻辑** - 无需关心框架配置和代码生成
2. **零学习成本** - 使用标准Go语法和Gin框架
3. **自动文档生成** - 代码即文档，永远保持同步
4. **高性能运行** - 编译时优化，零运行时开销
5. **企业级特性** - 完整的依赖注入、中间件、配置管理

**开始使用 gRain 框架，让开发变得简单而高效！** 🚀 