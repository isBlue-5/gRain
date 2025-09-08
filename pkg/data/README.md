# gRain 事务管理

gRain 框架提供强大且灵活的事务管理功能，支持声明式（注解）和编程式（模板方法）两种事务处理方式。

## 快速开始

### 声明式事务

使用事务注解快速实现自动事务管理：

```go
// frame:transaction(timeout="5s", rollbackFor={"*errors.Error"})
func CreateUser(ctx context.Context, user User) (User, error) {
    // 事务会自动开启、提交或回滚
    // 所有数据库操作都在同一事务中
    return repo.Save(ctx, user)
}
```

### 编程式事务

使用事务模板方法手动控制事务：

```go
result, err := data.TransactionTemplate(ctx, dbSession, func(txCtx context.Context) (interface{}, error) {
    // 在事务中执行业务逻辑
    user, err := repo.Save(txCtx, newUser)
    if err != nil {
        return nil, err
    }
    
    // 继续其他操作...
    return user, nil
})
```

## 事务特性

### 传播行为

支持完整的事务传播行为：

- `REQUIRED`：默认行为，如存在则使用，否则创建新事务
- `REQUIRES_NEW`：每次都创建新事务，挂起已存在的事务
- `SUPPORTS`：如存在则使用，否则非事务执行
- `NOT_SUPPORTED`：以非事务方式执行，挂起已存在的事务
- `MANDATORY`：要求事务必须存在，否则抛出异常
- `NEVER`：要求不能存在事务，否则抛出异常
- `NESTED`：如存在则嵌套创建（保存点），否则新建事务

### 事务参数

可定制的事务参数：

- `timeout`：事务超时时间（如 "5s"）
- `rollbackFor`：指定哪些错误类型需要回滚
- `readOnly`：标记只读事务，优化性能
- `isolation`：设置事务隔离级别

## 事务状态与诊断

每个事务对象提供状态跟踪：

```go
tx, ok := data.GetTransaction(ctx)
if ok {
    fmt.Printf("事务ID: %d, 层级: %d, 状态: %s\n", tx.ID(), tx.Level(), tx.Status())
}
```

## 高级用法

### 事件钩子

使用增强的事务模板添加事件钩子：

```go
result, err := data.TransactionTemplateV2(
    ctx,
    dbSession,
    businessFunc,
    beforeCommit,  // 提交前钩子
    afterCommit,   // 提交后钩子
    beforeRollback,// 回滚前钩子
    afterRollback, // 回滚后钩子
    logger,        // 日志函数
)
```

### 与日志集成

结合事务和日志记录：

```go
// 事务提交前记录审计日志
beforeCommit := func(ctx context.Context, tx data.Transaction) {
    log.Printf("用户 %s 即将提交事务 %d", getUserID(ctx), tx.ID())
    auditLogger.Info("数据变更即将提交", map[string]interface{}{
        "txID": tx.ID(),
        "user": getUserID(ctx),
    })
}
```

## 扩展与自定义

### 自定义事务管理器

实现自己的事务增强逻辑：

```go
func MyTransactionManager(ctx context.Context, session data.DBSession, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
    // 自定义事务逻辑...
    // 可以添加监控、重试、分布式事务等功能
}
```

## 更多信息

详细文档请参考：

- [事务注解使用指南](../docs/transaction_guide.md)
- [事务示例](../../examples/transaction/)
- [API 参考](https://godoc.org/github.com/your-org/gRain/pkg/data)

## 最佳实践

- 将事务边界放在服务层而非数据访问层
- 不要在事务中执行长时间操作（如远程调用）
- 使用适当的传播行为避免事务嵌套问题
- 在性能关键场景监控事务执行时间
- 正确处理和记录事务失败 