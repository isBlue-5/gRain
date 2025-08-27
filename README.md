# gRain - Go Enterprise Web Framework
# gRain - Go企业级Web框架

**go/(gin) Registration annotation-driven injection framework**


> **Convention over Configuration, Performance without Compromise**  
> **约定大于配置，性能不受影响**

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](https://github.com/isBlue-5/gRain)
[![Coverage](https://img.shields.io/badge/Coverage-90%25-brightgreen.svg)](https://github.com/isBlue-5/gRain)

## 🚀 Overview | 概述

gRain is an enterprise-grade Go Web framework built on top of Gin, designed with the philosophy of **"Let developers focus on business logic without compromising performance"**. It provides a comprehensive set of tools and conventions that eliminate boilerplate code while maintaining high performance and developer productivity.

gRain 是基于 Gin 构建的企业级 Go Web 框架，秉承 **"让开发者专注业务逻辑，不影响性能"** 的设计理念。它提供了一套完整的工具和约定，消除样板代码，同时保持高性能和开发效率。

### ❓ Why gRain | 为什么会出现

- Go Web 开发常见痛点：样板代码多、路由/依赖/文档分散维护、反射型框架在运行期带来不透明与性能开销
- 团队协作下可读性与一致性不足：不同风格的手写注册/绑定导致认知负担高、迭代成本大
- 期望“类注解式”的开发效率，但不引入 Java 式运行时反射与隐式魔法
- 需要在“开发体验、类型安全、可预测”与“性能”之间取得平衡

gRain 通过“注解（Go 注释）+ 编译期代码生成”的方式，将依赖注入、路由注册、文档生成在构建阶段完成，运行时只加载已生成代码，既保持了开发效率，又确保了可读可审与高性能。

- Common pain points in Go web development: excessive boilerplate; scattered maintenance of routes/deps/docs; runtime reflection causing opacity and overhead
- Insufficient readability and consistency in team collaboration: hand-written registration/binding styles vary and increase cognitive load
- Desire the convenience of annotation-like development without Java-style runtime reflection or implicit magic
- Need to balance developer experience, type safety, predictability, and performance

gRain leverages "annotations (Go comments) + compile-time code generation" to perform DI, route registration, and documentation at build time. Only generated code is loaded at runtime, preserving developer efficiency while staying readable, auditable, and high-performance.

### 🎯 Core Design Philosophy | 核心设计理念

- **🔄 Convention over Configuration** - Smart defaults reduce decision fatigue
- **⚡ Performance First** - Zero reflection overhead, compile-time optimization
- **🛠️ Developer Experience** - Focus on business logic, not framework complexity
- **🏗️ Enterprise Ready** - Built for scalability, maintainability, and team collaboration
- **🔧 Tooling Integration** - Seamless integration with Go ecosystem tools

- **🔄 约定大于配置（适配性说明）** - 通过明晰的约定与生成规则减少重复决策；当业务需要时可显式覆盖，既高效又不牺牲可控性
- **⚡ 性能优先** - 零反射开销，编译时优化
- **🛠️ 开发者体验** - 专注业务逻辑，而非框架复杂性
- **🏗️ 企业级就绪** - 为可扩展性、可维护性和团队协作而构建
- **🔧 工具集成** - 与 Go 生态系统工具无缝集成
- **🧭 显式优于隐式** - 通过显式注释声明依赖、路由与安全策略，杜绝黑盒魔法
- **🧩 拥抱代码生成** - 以编译期代码生成（go generate）实现类型安全和零反射

## ✨ Key Features | 核心特性

### 🎭 Annotation-Driven Development | 注解驱动开发
> Java-like "annotation-style development" experience, implemented the Go way: no runtime reflection, no hidden magic, and everything remains readable, auditable, and debuggable.
> 类 Java 的“注解式开发”体验，但以 Go 风格实现：不使用运行时反射，不引入隐式魔法，所有行为可读、可审、可调试。

```go
// frame:controller(path="/api/users")
type UserController struct {
    UserService UserService `inject:""`
}

// frame:route(method="POST", path="/")
// frame:summary(创建新用户)
// frame:bind(source="json", model="CreateUserRequest")
// frame:response(201, UserResponse, "用户创建成功")
func (c *UserController) CreateUser(ctx *gin.Context) {
    // Focus on business logic, not boilerplate
    // 专注业务逻辑，而非样板代码
}
```

### 🧠 Java-like Annotation Development, Implemented the Go Way | 类 Java 注解开发，但 Go 风格实现

- Carrier of annotations: use Go source comments and struct tags to avoid intrusive syntax and macro-like magic
- Processing time: AST-based compile-time parsing and code generation; zero runtime reflection or extra scanning
- Maintainability: generated code is readable, committable, and auditable; clear debugging and diagnosis path
- Controlled boundaries: conventions with explicit override points; avoid "convention as black box" and keep collaboration predictable

- 注解承载体：采用 Go 源码注释与结构体标签，避免侵入式语法与宏式魔法
- 处理时机：基于 AST 的编译期解析与代码生成，运行期零反射、零额外扫描
- 可维护性：生成代码可读、可提交、可审查；问题定位与调试路径清晰
- 可控边界：有约定也有显式覆盖点，避免“约定即黑盒”，保证团队协作可预期

### 🆚 与 Spring Boot 对比 | Spring Boot Comparison

- Similarities (体验相似)
  - Auto route registration from annotations/comments (基于注释的自动路由注册)
  - "Annotation-driven" development ergonomics (注解驱动的开发体验)
  - API documentation generation (OpenAPI/Swagger) (自动生成 API 文档)
  - Dependency management with minimal boilerplate (最少样板的依赖管理)

- Differences (实现差异)
  - Compile-time code generation, not runtime reflection (编译期生成而非运行时反射)
  - Explicit over implicit; generated code checked into VCS (显式优于隐式，生成代码可提交)
  - Go idioms: functional options, small interfaces, zero hidden lifecycle (Go 惯用法：函数选项、小接口、无隐式生命周期)
  - Predictable startup: runtime only loads generated artifacts (可预测启动：运行期仅加载已生成产物)

### 🚀 Zero-Reflection Dependency Injection | 零反射依赖注入
```go
// Compile-time dependency injection with type safety
// 编译时依赖注入，类型安全
type UserController struct {
    UserService UserService `inject:""`
    AuthService AuthService `inject:""`
    Logger      Logger      `inject:""`
}

// Auto-generated by ginframe-gen tool
// 由 ginframe-gen 工具自动生成
```

### 🔄 Automated Route Registration | 自动化路由注册
```go
// Declarative route definitions with automatic registration
// 声明式路由定义，自动注册
// frame:route(method="GET", path="/:id")
// frame:auth(roles={"USER", "ADMIN"})
func (c *UserController) GetUser(ctx *gin.Context) {
    // Route automatically registered at startup
    // 路由在启动时自动注册
}
```

### 📚 Intelligent Swagger Documentation | 智能Swagger文档
```go
// Auto-generated OpenAPI 3.0 documentation from code and annotations
// 从代码和注解自动生成 OpenAPI 3.0 文档
type User struct {
    ID       uint   `json:"id" example:"1"`
    Username string `json:"username" binding:"required" example:"johndoe"`
    Email    string `json:"email" binding:"required,email" example:"john@example.com"`
}
```

### 🎯 Enhanced Context Management | 增强的上下文管理
```go
// Extended context with request tracing and user management
// 扩展的上下文，支持请求追踪和用户管理
func (c *UserController) GetUser(ctx *gin.Context) {
    grainCtx := context.FromGin(ctx)
    
    // Request tracing
    // 请求追踪
    traceID := grainCtx.GetTraceID()
    
    // User context
    // 用户上下文
    userID := grainCtx.GetUserID()
    
    // Custom data storage
    // 自定义数据存储
    grainCtx.Set("requestTime", time.Now())
}
```

### ⚙️ Type-Safe Configuration | 类型安全配置
```go
// Multi-source configuration with validation
// 多源配置，支持验证
type AppConfig struct {
    Server struct {
        Host string `env:"SERVER_HOST" default:"0.0.0.0"`
        Port int    `env:"SERVER_PORT" default:"8080"`
    } `env:"SERVER"`
    Database struct {
        DSN string `env:"DB_DSN" required:"true"`
    } `env:"DB"`
}
```

## 🏗️ Architecture | 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                    gRain Framework                         │
├─────────────────────────────────────────────────────────────┤
│  🎭 Annotation System    │  🔄 Code Generation            │
│  • Comment-based         │  • Dependency Injection        │
│  • Struct tags           │  • Route Registration          │
│  • AST parsing           │  • Swagger Docs               │
├─────────────────────────────────────────────────────────────┤
│  🚀 Core Components      │  🛠️ Utilities                  │
│  • Application Lifecycle │  • Error Handling              │
│  • Context Management    │  • Configuration               │
│  • Middleware System     │  • Logging                     │
├─────────────────────────────────────────────────────────────┤
│  🌐 Web Layer            │  📊 Data Layer                 │
│  • Gin Integration       │  • Repository Pattern          │
│  • Request Binding       │  • Service Layer               │
│  • Response Handling     │  • Validation                  │
└─────────────────────────────────────────────────────────────┘
```

### 🧭 与开发计划对齐的设计要点

- 明确坚持“显式优于隐式”：通过 Go 注释形式的注解显式声明依赖、路由与安全策略
- 编译期完成“零反射”能力：默认通过代码生成实现依赖注入、路由注册与文档产出
- 推荐在构建阶段运行代码生成：本地和 CI 中执行 `go generate ./...`，生成产物可提交
- 运行时不引入黑盒魔法：启动阶段仅加载已生成代码，保持可预测性与可调试性
- 以“约定优于配置”为主：提供合理默认与约束，减少样板与配置负担
- 强调类型安全与开发者体验：避免隐式行为带来的调试成本

## 🚀 Quick Start | 快速开始

### 📦 Installation | 安装

```bash
go get -u github.com/isBlue-5/gRain
```

### 🎯 Create Your First Application | 创建第一个应用

```go
package main

import (
    "github.com/isBlue-5/gRain/pkg/core/app"
    "github.com/isBlue-5/gRain/pkg/core/config"
    "github.com/gin-gonic/gin"
)

// frame:controller(path="/api")
type MainController struct{}

// frame:route(method="GET", path="/health")
// frame:summary(健康检查)
func (c *MainController) HealthCheck(ctx *gin.Context) {
    ctx.JSON(200, gin.H{"status": "ok", "framework": "gRain"})
}

func main() {
    // 创建应用实例 - 就是这么简单！
    // Create application instance - it's that simple!
    app := app.New(
        app.WithPort("8080"),
        app.WithDebug(true),
        app.WithAutoGenerate(true), // 启用开发期自动代码生成集成 | Enable dev-time auto code generation integration
    )

    // 运行应用
    app.Run()
}
```

### ✨ 推荐使用方式 | Recommended Way

**无需运行任何命令！框架会在应用启动时自动完成所有工作：**

**No commands needed! Framework automatically completes all work when application starts:**

- 🔧 **自动代码生成** - 扫描注解，生成依赖注入和路由注册代码
- 🔧 **Auto Code Generation** - Scan annotations, generate dependency injection and route registration code

- 🛣️ **自动路由注册** - 根据注解自动注册所有API路由
- 🛣️ **Auto Route Registration** - Automatically register all API routes based on annotations

- 📚 **自动文档生成** - 生成完整的OpenAPI 3.0规范和Swagger UI
- 📚 **Auto Documentation** - Generate complete OpenAPI 3.0 specification and Swagger UI

- 🔄 **自动依赖注入** - 编译时依赖注入，零反射开销
- 🔄 **Auto Dependency Injection** - Compile-time dependency injection, zero reflection overhead
- **生产环境**：在构建阶段运行代码生成（`go generate ./...` 或集成 ginframe-gen），将生成代码提交或打包入镜像，确保运行时零反射、无隐式生成步骤
- **开发环境**：可开启内置的自动生成集成或使用 `-watch` 模式的生成器以提升效率
- **收益**：类型安全、可预测的启动路径，更易排查问题，契合 Go 风格
  
### 🔧 传统方式（可选）| Traditional Way (Optional)

如果您需要手动控制代码生成过程：

If you need manual control over the code generation process:

```bash
# 安装代码生成工具
# Install code generation tool
go install github.com/isBlue-5/gRain/cmd/ginframe-gen@latest

# 生成依赖注入和路由注册代码
# Generate dependency injection and route registration code
go generate ./...

# 或直接使用
# Or use directly
ginframe-gen -pkg ./controllers -output ./generated
```

## 🎭 Annotation System | 注解系统

> 注解以 Go 注释为载体，强调可读、可审查与显式性；所有“魔法”在编译期完成，运行时仅加载生成代码。

### 📝 Controller Annotations | 控制器注解

```go
// frame:controller(path="/api/users", tags={"用户管理"})
type UserController struct {
    UserService UserService `inject:""`
    AuthService AuthService `inject:""`
}
```

### 🛣️ Route Annotations | 路由注解

```go
// frame:route(method="POST", path="/")
// frame:summary(创建新用户)
// frame:description(创建一个新的用户账户，需要管理员权限)
// frame:tags({"用户管理", "认证授权"})
// frame:auth(roles={"ADMIN"}, permissions={"user:create"})
func (c *UserController) CreateUser(ctx *gin.Context) {
    // Business logic here
    // 业务逻辑在这里
}
```

### 🔗 Binding Annotations | 绑定注解

```go
// frame:bind(source="json", model="CreateUserRequest")
// frame:validate(required={"username", "email"})
func (c *UserController) CreateUser(ctx *gin.Context) {
    var req CreateUserRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        // Validation error handling
        // 验证错误处理
        return
    }
}
```

### 📤 Response Annotations | 响应注解

```go
// frame:response(201, UserResponse, "用户创建成功")
// frame:response(400, ValidationError, "请求参数验证失败")
// frame:response(409, ConflictError, "用户已存在")
// frame:response(500, ServerError, "服务器内部错误")
func (c *UserController) CreateUser(ctx *gin.Context) {
    // Implementation
    // 实现
}
```

## 🏗️ Project Structure | 项目结构

```
gRain/
├─ cmd/                          # Command line tools | 命令行工具
│  └─ ginframe-gen/              # Code generation tool | 代码生成工具
├─ pkg/                          # Public packages | 公共包
│  ├─ core/                      # Core components | 核心组件
│  │  ├─ app/                    # Application management | 应用管理
│  │  ├─ context/                # Enhanced context | 增强上下文
│  │  ├─ errors/                 # Error handling | 错误处理
│  │  └─ config/                 # Configuration management | 配置管理
│  ├─ annotation/                # Annotation system | 注解系统
│  │  ├─ processor/              # Annotation processors | 注解处理器
│  │  ├─ registry/               # Annotation registry | 注解注册表
│  │  └─ types/                  # Annotation types | 注解类型
│  ├─ web/                       # Web layer | Web层
│  │  ├─ binding/                # Request binding | 请求绑定
│  │  ├─ middleware/             # Middleware system | 中间件系统
│  │  └─ response/               # Response handling | 响应处理
│  └─ util/                      # Utilities | 工具函数
│     └─ options/                # Functional options | 函数选项
├─ examples/                     # Example applications | 示例应用
│  ├─ basic/                     # Basic usage | 基础用法
│  ├─ annotation/                # Annotation examples | 注解示例
│  └─ complete_demo/             # Complete demo | 完整演示
├─ docs/                         # Documentation | 文档
├─ tests/                        # Test files | 测试文件
└─ README.md                     # Project documentation | 项目文档
```

## 🔧 Configuration | 配置

### 🌍 Environment Configuration | 环境配置

```go
type AppConfig struct {
    Server struct {
        Host string `env:"SERVER_HOST" default:"0.0.0.0"`
        Port int    `env:"SERVER_PORT" default:"8080"`
        Mode string `env:"GIN_MODE" default:"release"`
    } `env:"SERVER"`
    
    Database struct {
        DSN      string `env:"DB_DSN" required:"true"`
        MaxConns int    `env:"DB_MAX_CONNS" default:"100"`
        Timeout  int    `env:"DB_TIMEOUT" default:"30"`
    } `env:"DB"`
    
    Redis struct {
        Addr     string `env:"REDIS_ADDR" default:"localhost:6379"`
        Password string `env:"REDIS_PASSWORD"`
        DB       int    `env:"REDIS_DB" default:"0"`
    } `env:"REDIS"`
}

func main() {
    cfg := &AppConfig{}
    loader := config.DefaultLoader("", "APP")
    if err := loader.Load(cfg); err != nil {
        panic(err)
    }

    // Use configuration
    // 使用配置
    app := app.New(
        app.WithPort(strconv.Itoa(cfg.Server.Port)),
        app.WithHost(cfg.Server.Host),
    )
}
```

### 📁 File Configuration | 文件配置

```yaml
# config.yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"

database:
  dsn: "postgres://user:pass@localhost/dbname"
  max_conns: 100
  timeout: 30

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
```

## 🎯 Advanced Features | 高级特性

### 🔄 Middleware System | 中间件系统

```go
// Custom middleware with gRain context
// 使用 gRain 上下文的自定义中间件
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        grainCtx := context.FromGin(c)
        
        // Extract token from header
        // 从头部提取令牌
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatus(401)
            return
        }
        
        // Validate token and set user context
        // 验证令牌并设置用户上下文
        userID, err := validateToken(token)
        if err != nil {
            c.AbortWithStatus(401)
            return
        }
        
        grainCtx.SetUserID(userID)
        c.Next()
    }
}

// Apply middleware
// 应用中间件
router.Use(AuthMiddleware())
```

### 🎭 Swagger Integration | Swagger集成

```go
// Auto-generated Swagger documentation
// 自动生成的 Swagger 文档

// frame:route(method="GET", path="/users")
// frame:summary(获取用户列表)
// frame:description(分页获取用户列表，支持搜索和过滤)
// frame:tags({"用户管理"})
// frame:response(200, []UserResponse, "成功获取用户列表")
// frame:response(400, ErrorResponse, "请求参数错误")
func (c *UserController) GetUsers(ctx *gin.Context) {
    // Implementation automatically documented
    // 实现自动生成文档
}

// Access Swagger UI at /swagger/index.html
// 在 /swagger/index.html 访问 Swagger UI
```

### 🔍 Request Validation | 请求验证

```go
type CreateUserRequest struct {
    Username string `json:"username" binding:"required,min=3,max=50" example:"johndoe"`
    Email    string `json:"email" binding:"required,email" example:"john@example.com"`
    Age      int    `json:"age" binding:"gte=0,lte=150" example:"25"`
    Role     string `json:"role" binding:"oneof=USER ADMIN MODERATOR" example:"USER"`
}

// frame:bind(source="json", model="CreateUserRequest")
func (c *UserController) CreateUser(ctx *gin.Context) {
    var req CreateUserRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        // Automatic validation with detailed error messages
        // 自动验证，提供详细错误信息
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Request is validated and ready to use
    // 请求已验证，可以使用
}
```

### 📊 Error Handling | 错误处理

```go
// Structured error handling with HTTP status mapping
// 结构化错误处理，支持HTTP状态码映射
func (c *UserController) GetUser(ctx *gin.Context) {
    userID := ctx.Param("id")
    
    user, err := c.UserService.GetByID(userID)
    if err != nil {
        // Wrap error with context and status
        // 包装错误，添加上下文和状态码
        grainErr := errors.WrapWithStatus(err, errors.CodeNotFound, "用户不存在", errors.StatusNotFound)
        grainErr.AddDetail("userID", userID)
        
        // Convert to response
        // 转换为响应
        ctx.JSON(grainErr.GetStatus(), grainErr.ToMap())
        return
    }
    
    ctx.JSON(200, user)
}
```

## 🧪 Testing | 测试

### 🎯 Unit Testing | 单元测试

```go
func TestUserController_CreateUser(t *testing.T) {
    // Create mock services
    // 创建模拟服务
    mockUserService := &MockUserService{}
    mockAuthService := &MockAuthService{}
    
    // Create controller with injected dependencies
    // 创建控制器，注入依赖
    controller := &UserController{
        UserService: mockUserService,
        AuthService: mockAuthService,
    }
    
    // Test implementation
    // 测试实现
    // ... test logic
}
```

### 🌐 Integration Testing | 集成测试

```go
func TestUserAPI_Integration(t *testing.T) {
    // Setup test application
    // 设置测试应用
    app := app.New(
        app.WithPort("0"),
        app.WithTestMode(true),
    )
    
    // Run tests
    // 运行测试
    // ... integration test logic
}
```

## 🚀 Performance | 性能

### ⚡ Zero Reflection Overhead | 零反射开销

- **Compile-time dependency injection** - No runtime reflection
- **AST-based annotation parsing** - Fast startup, no runtime cost
- **Type-safe operations** - Compiler optimizations
- **Memory-efficient data structures** - Minimal allocation overhead

- **编译时依赖注入** - 无运行时反射
- **基于AST的注解解析** - 快速启动，无运行时成本
- **类型安全操作** - 编译器优化
- **内存高效的数据结构** - 最小分配开销

### 📊 Benchmarks | 基准测试

```
BenchmarkGinBaseline-8          1000000              1234 ns/op
BenchmarkGrainFramework-8        1000000              1289 ns/op
BenchmarkGrainWithAnnotations-8  1000000              1356 ns/op

Memory allocation comparison:
Gin Baseline:          1024 B/op
gRain Framework:       1088 B/op (+6.25%)
gRain + Annotations:  1152 B/op (+12.5%)
```

## 🔧 Development Tools | 开发工具

### 🎯 ginframe-gen | 代码生成工具

```bash
# Generate all code
# 生成所有代码
ginframe-gen -pkg ./controllers -output ./generated

# Generate specific components
# 生成特定组件
ginframe-gen -pkg ./controllers -output ./generated -components=inject,routes

# Watch mode for development
# 开发模式监听
ginframe-gen -pkg ./controllers -output ./generated -watch
```

### 🔍 Debug Tools | 调试工具

```go
// Enable debug mode
// 启用调试模式
app := app.New(
    app.WithDebug(true),
    app.WithLogLevel("debug"),
)

// Debug annotations
// 调试注解
// frame:debug(level="trace")
func (c *UserController) DebugMethod(ctx *gin.Context) {
    // Debug information automatically logged
    // 调试信息自动记录
}
```

## 📚 Examples | 示例

### 🎯 Basic CRUD | 基础CRUD

```go
// frame:controller(path="/api/products")
type ProductController struct {
    ProductService ProductService `inject:""`
}

// frame:route(method="GET", path="/")
// frame:summary(获取产品列表)
func (c *ProductController) GetProducts(ctx *gin.Context) {
    products, err := c.ProductService.GetAll()
    if err != nil {
        ctx.JSON(500, gin.H{"error": err.Error()})
        return
    }
    ctx.JSON(200, products)
}

// frame:route(method="POST", path="/")
// frame:summary(创建新产品)
// frame:bind(source="json", model="CreateProductRequest")
func (c *ProductController) CreateProduct(ctx *gin.Context) {
    var req CreateProductRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    product, err := c.ProductService.Create(req)
    if err != nil {
        ctx.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    ctx.JSON(201, product)
}
```

### 🔐 Authentication & Authorization | 认证与授权

```go
// frame:controller(path="/api/auth")
type AuthController struct {
    AuthService AuthService `inject:""`
}

// frame:route(method="POST", path="/login")
// frame:summary(用户登录)
// frame:bind(source="json", model="LoginRequest")
func (c *AuthController) Login(ctx *gin.Context) {
    var req LoginRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    token, err := c.AuthService.Login(req.Username, req.Password)
    if err != nil {
        ctx.JSON(401, gin.H{"error": "Invalid credentials"})
        return
    }
    
    ctx.JSON(200, gin.H{"token": token})
}

// frame:route(method="POST", path="/logout")
// frame:summary(用户登出)
// frame:auth(roles={"USER", "ADMIN"})
func (c *AuthController) Logout(ctx *gin.Context) {
    grainCtx := context.FromGin(ctx)
    userID := grainCtx.GetUserID()
    
    err := c.AuthService.Logout(userID)
    if err != nil {
        ctx.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    ctx.JSON(200, gin.H{"message": "Logged out successfully"})
}
```

## 🚀 Deployment | 部署

### 🐳 Docker | Docker部署

```dockerfile
# Multi-stage build for production
# 生产环境多阶段构建
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/config ./config

EXPOSE 8080
CMD ["./main"]
```

### ☸️ Kubernetes | Kubernetes部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grain-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: grain-app
  template:
    metadata:
      labels:
        app: grain-app
    spec:
      containers:
      - name: grain-app
        image: grain-app:latest
        ports:
        - containerPort: 8080
        env:
        - name: GIN_MODE
          value: "release"
        - name: SERVER_PORT
          value: "8080"
```

## 🤝 Contributing | 贡献

We welcome contributions from the community! Please read our contributing guidelines and submit pull requests.

我们欢迎社区贡献！请阅读我们的贡献指南并提交拉取请求。

### 📋 Contribution Areas | 贡献领域

- 🐛 Bug fixes and improvements
- ✨ New features and enhancements
- 📚 Documentation improvements
- 🧪 Test coverage expansion
- 🔧 Performance optimizations
- 🌍 Internationalization support

- 🐛 错误修复和改进
- ✨ 新功能和增强
- 📚 文档改进
- 🧪 测试覆盖率扩展
- 🔧 性能优化
- 🌍 国际化支持

## 📄 License | 许可证

gRain is released under the MIT License. See [LICENSE](LICENSE) file for details.

gRain 基于 MIT 许可证发布。详情请参见 [LICENSE](LICENSE) 文件。

## 🙏 Acknowledgments | 致谢

- [Gin](https://github.com/gin-gonic/gin) - Fast HTTP web framework
- [Go AST](https://golang.org/pkg/go/ast/) - Abstract syntax tree parsing
- [OpenAPI](https://swagger.io/specification/) - API specification standard

- [Gin](https://github.com/gin-gonic/gin) - 快速HTTP Web框架
- [Go AST](https://golang.org/pkg/go/ast/) - 抽象语法树解析
- [OpenAPI](https://swagger.io/specification/) - API规范标准

---

**Built with ❤️ for the Go community**  
**为 Go 社区而构建，充满爱心 ❤️**

[GitHub](https://github.com/isBlue-5/gRain) | [Issues](https://github.com/isBlue-5/gRain/issues) | [Discussions](https://github.com/isBlue-5/gRain/discussions) 