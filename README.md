# gRain - Go企业级Web框架
go/(gin) Registration annotation-driven injection framework
gRain是基于Gin的企业级Go Web框架，提供丰富的功能和优秀的开发体验，特别适合构建大型企业应用。

## 核心特性

- **函数选项模式**：类型安全的API配置方式
- **统一上下文管理**：增强的`context.Context`，支持请求全链路追踪
- **应用生命周期管理**：优雅启动与关闭，资源管理
- **统一错误处理**：结构化错误，错误链，HTTP状态码映射
- **类型安全配置**：多源配置，类型绑定，校验
- **注解系统**：基于注释和结构体标签的编译期"注解"，无反射开销git add README.md
- **代码生成**：通过`ginframe-gen`工具自动生成依赖注入和路由注册代码
- **依赖注入**：编译期依赖注入，类型安全，无反射
- **路由注册**：声明式路由定义，自动注册

## 快速开始

### 安装

```bash
go get -u github.com/yourusername/gRain
```

### 创建应用

```go
package main

import (
    "github.com/yourusername/gRain/pkg/core/app"
    "github.com/yourusername/gRain/pkg/core/config"
    "github.com/gin-gonic/gin"
)

type AppConfig struct {
    Server struct {
        Port string `env:"SERVER_PORT" default:"8080"`
    }
}

func main() {
    // 创建配置
    cfg := &AppConfig{}
    loader := config.DefaultLoader("", "APP")
    if err := loader.Load(cfg); err != nil {
        panic(err)
    }

    // 创建应用
    application := app.New(
        app.WithPort(cfg.Server.Port),
    )

    // 添加路由
    router := gin.Default()
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })

    // 运行应用
    application.Run()
}
```

### 使用注解系统

示例控制器:

```go
// frame:controller(path="/api/users")
type UserController struct {
    // 通过inject注解标记依赖注入
    UserService UserService `inject:""`
}

// GetUser 获取单个用户
// frame:route(method="GET", path="/:id")
// frame:auth(roles={"USER", "ADMIN"})
// frame:log(level="INFO")
func (c *UserController) GetUser(ctx *gin.Context) {
    // ...
}
```

使用代码生成工具:

```bash
go generate ./...  # 调用ginframe-gen生成依赖注入和路由注册代码
```

## 框架结构

```
gRain/
├─ cmd/                  # 命令行工具
│  └─ ginframe-gen/      # 代码生成工具
├─ pkg/                  # 公共包
│  ├─ core/              # 核心组件
│  │  ├─ app/            # 应用管理
│  │  ├─ context/        # 上下文管理
│  │  ├─ errors/         # 错误处理
│  │  └─ config/         # 配置管理
│  ├─ annotation/        # 注解系统
│  │  ├─ processor/      # 注解处理器
│  │  ├─ registry/       # 注解注册表
│  │  └─ types/          # 注解类型
│  └─ util/              # 工具函数
│     └─ options/        # 函数选项工具
├─ examples/             # 示例应用
│  └─ annotation/        # 注解系统示例
└─ README.md             # 项目说明文档
```

## 详细文档

### 应用管理

```go
// 创建应用
app := app.New(
    app.WithHost("127.0.0.1"),
    app.WithPort("8080"),
    app.WithShutdownTimeout(30),
)

// 添加服务
app.AddService(myService)

// 运行应用
err := app.Run()
```

### 上下文管理

```go
// 创建上下文
ctx := context.NewContext(c.Request.Context())
ctx.SetUserID("123")
ctx.SetRequestID("req-456")

// 获取上下文数据
userID := ctx.GetUserID()
requestID := ctx.GetRequestID()

// 存储自定义数据
ctx.Set("key", value)
val, ok := ctx.Get("key")
```

### 错误处理

```go
// 创建错误
err := errors.NewError(errors.CodeNotFound, "用户不存在")

// 包装错误
err = errors.Wrap(originalErr, errors.CodeDatabaseError, "查询用户失败")

// 设置HTTP状态码
err = errors.WrapWithStatus(originalErr, errors.CodeForbidden, "没有权限", errors.StatusForbidden)

// 添加详情
err.AddDetail("userId", "123")

// 错误检查
if errors.IsNotFound(err) {
    // 处理未找到错误
}

// 转换为响应
resp := err.ToMap()
```

### 配置管理

```go
// 定义配置结构
type ServerConfig struct {
    Host    string `env:"SERVER_HOST" default:"0.0.0.0"`
    Port    int    `env:"SERVER_PORT" default:"8080"`
    LogPath string `env:"LOG_PATH" required:"true"`
}

// 加载配置
cfg := &ServerConfig{}
loader := config.NewConfigLoader(
    &config.EnvConfigSource{Prefix: "APP"},
    &config.YAMLConfigSource{Path: "config.yaml"},
)
if err := loader.Load(cfg); err != nil {
    // 处理错误
}
```

### 函数选项模式

```go
// 定义选项
type Config struct {
    Timeout int
    Retries int
    Debug   bool
}

type Option func(*Config)

func WithTimeout(timeout int) Option {
    return func(c *Config) {
        c.Timeout = timeout
    }
}

// 应用选项
cfg := &Config{}
options.Apply(cfg, WithTimeout(30))
```

### 注解系统

```go
// 通过注释方式使用注解
// frame:route(method="POST", path="/users")
// frame:auth(roles={"ADMIN"})
func (c *UserController) CreateUser(ctx *gin.Context) {
    // ...
}

// 通过结构体标签使用注解
type UserController struct {
    UserService UserService `inject:""`
}
```

## 当前状态

gRain框架已完成第一阶段和第二阶段开发：

- 第一阶段：基础架构与核心组件（完成）
  - 框架项目结构初始化
  - 函数选项模式实现
  - 核心Context实现
  - 应用生命周期管理
  - 错误处理与配置管理系统

- 第二阶段：注解系统与代码生成（完成）
  - 注解类型系统
  - 注解解析器和注册中心
  - 依赖注入生成
  - 路由注册生成
  - 代码生成工具

- 第三阶段：Web层与请求处理（进行中）
  - 请求绑定与参数验证
  - 内容协商与响应处理
  - 中间件系统完善

## 贡献

我们欢迎任何形式的贡献，包括功能请求、错误报告、文档改进和代码贡献。请通过GitHub提交问题和拉取请求。

## 许可证

gRain框架是在MIT许可下发布的开源软件。 