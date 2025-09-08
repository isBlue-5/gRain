# 事务注解使用指南与最佳实践

## 1. 事务注解基础

### 1.1 什么是事务注解

事务注解是一种声明式事务管理方式，通过在方法上添加特定格式的注释，实现自动事务管理，无需编写样板代码。

```go
// frame:transaction(timeout="5s", rollbackFor={"*errors.Error", "*ValidationError"})
func CreateUser(ctx context.Context, user User) (User, error) {
    // 方法实现...
}
```

### 1.2 支持的参数

| 参数 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| propagation | string | 事务传播行为 | "REQUIRED" |
| timeout | string | 事务超时时间 | "" (无超时) |
| rollbackFor | []string | 指定回滚的错误类型 | {} (所有错误) |
| readOnly | bool | 是否为只读事务 | false |
| isolation | string | 事务隔离级别 | "" (使用数据库默认) |

## 2. 事务传播行为

### 2.1 支持的传播行为类型

- **REQUIRED**：如存在事务则加入，否则新建事务（默认）
- **REQUIRES_NEW**：总是创建新事务，如存在事务则挂起
- **SUPPORTS**：如存在事务则加入，否则无事务执行
- **NOT_SUPPORTED**：无事务执行，如存在事务则挂起
- **MANDATORY**：必须在已有事务中执行，否则抛出异常
- **NEVER**：必须在无事务环境下执行，否则抛出异常
- **NESTED**：如存在事务则创建嵌套事务（保存点），否则新建事务

### 2.2 各传播行为使用场景

- **REQUIRED**：适用于大多数业务方法
- **REQUIRES_NEW**：适用于需要独立事务的操作，如日志记录
- **SUPPORTS**：适用于既可在事务中也可在非事务中执行的方法
- **NOT_SUPPORTED**：适用于不需要事务且耗时较长的操作
- **MANDATORY**：适用于必须在调用者事务中执行的方法
- **NEVER**：适用于不应在事务中执行的操作
- **NESTED**：适用于可能需要部分回滚的复杂操作

## 3. 事务状态跟踪与诊断

### 3.1 事务状态信息

每个事务都包含以下状态信息，可用于调试和诊断：

- **ID**: 唯一标识符
- **Level**: 嵌套层级（0为最外层）
- **Status**: 当前状态（active/committed/rolledback）

### 3.2 获取事务信息

```go
// 通过上下文获取当前事务
if tx, ok := data.GetTransaction(ctx); ok {
    fmt.Printf("事务ID: %d, 层级: %d, 状态: %s\n", 
        tx.ID(), tx.Level(), tx.Status())
}
```

## 4. 高级用法

### 4.1 事务模板方法

框架提供两种事务模板方法，用于编程式事务管理：

```go
// 基本事务模板
result, err := data.TransactionTemplate(ctx, dbSession, func(txCtx context.Context) (interface{}, error) {
    // 事务中的业务逻辑
    return nil, nil
})

// 带事件钩子的增强事务模板
result, err := data.TransactionTemplateV2(
    ctx,
    dbSession,
    func(txCtx context.Context) (interface{}, error) {
        // 事务中的业务逻辑
        return nil, nil
    },
    beforeCommit, // 提交前钩子
    afterCommit,  // 提交后钩子
    beforeRollback, // 回滚前钩子
    afterRollback,  // 回滚后钩子
    loggerFunc, // 日志函数
)
```

### 4.2 事务事件钩子

事务事件钩子可用于在事务生命周期的关键点执行自定义逻辑：

```go
// 事务提交前钩子
beforeCommit := func(ctx context.Context, tx data.Transaction) {
    log.Printf("事务 %d 即将提交", tx.ID())
    // 可执行提交前验证、审计等逻辑
}

// 事务提交后钩子
afterCommit := func(ctx context.Context, tx data.Transaction) {
    log.Printf("事务 %d 已提交", tx.ID())
    // 可执行缓存清理、通知等后续处理
}
```

## 5. 最佳实践

### 5.1 事务边界设置

- 将事务边界设置在服务层（Service），而不是数据访问层（DAO/Repository）
- 确保事务边界覆盖所有需要原子操作的数据库访问
- 避免在单个事务中执行过多操作，可能导致锁竞争

### 5.2 避免事务中的阻塞操作

- 不要在事务中执行 HTTP 请求等远程调用
- 避免在事务中进行长时间计算
- 对于耗时操作，考虑使用 NOT_SUPPORTED 传播行为

### 5.3 异常处理

- 正确设置 rollbackFor 参数，指定需要回滚的异常类型
- 区分业务异常和技术异常，针对不同异常采取不同的回滚策略
- 确保事务回滚后的状态一致性

### 5.4 性能考虑

- 使用 readOnly=true 标记只读事务，优化数据库资源使用
- 合理设置 timeout 参数，避免事务长时间持有锁
- 对于查询密集型操作，考虑使用 SUPPORTS 传播行为

### 5.5 事务与日志结合

- 在事务钩子中添加审计日志记录
- 关键操作增加事务状态日志
- 事务失败时记录详细的错误信息和状态

## 6. 故障排除

### 6.1 常见问题

- **事务不生效**: 检查是否通过注入的上下文对象访问数据库
- **嵌套事务异常**: 检查传播行为设置，确认是否支持嵌套
- **事务超时**: 检查操作耗时，可能需要优化或拆分事务

### 6.2 调试技巧

- 开启事务日志输出，跟踪事务生命周期
- 使用 TransactionTemplateV2 的自定义日志功能
- 在事务钩子中添加调试信息 