package data

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTransaction 模拟事务实现，用于测试
type MockTransaction struct {
	mock.Mock
	id     uint64
	level  int
	status string
}

func (m *MockTransaction) ID() uint64 {
	return m.id
}

func (m *MockTransaction) Level() int {
	return m.level
}

func (m *MockTransaction) Status() string {
	return m.status
}

func (m *MockTransaction) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	arguments := m.Called(ctx, dest, query, args)
	return arguments.Error(0)
}

func (m *MockTransaction) QueryRow(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	arguments := m.Called(ctx, dest, query, args)
	return arguments.Error(0)
}

func (m *MockTransaction) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	arguments := m.Called(ctx, query, args)
	result, _ := arguments.Get(0).(sql.Result)
	return result, arguments.Error(1)
}

func (m *MockTransaction) BeginTx(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	arguments := m.Called(ctx, opts)
	tx, _ := arguments.Get(0).(Transaction)
	return tx, arguments.Error(1)
}

func (m *MockTransaction) Commit() error {
	args := m.Called()
	if err := args.Error(0); err == nil {
		m.status = "committed"
	}
	return args.Error(0)
}

func (m *MockTransaction) Rollback() error {
	args := m.Called()
	if err := args.Error(0); err == nil {
		m.status = "rolledback"
	}
	return args.Error(0)
}

// MockDBSession 模拟数据库会话实现，用于测试
type MockDBSession struct {
	mock.Mock
}

func (m *MockDBSession) Query(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	arguments := m.Called(ctx, dest, query, args)
	return arguments.Error(0)
}

func (m *MockDBSession) QueryRow(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	arguments := m.Called(ctx, dest, query, args)
	return arguments.Error(0)
}

func (m *MockDBSession) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	arguments := m.Called(ctx, query, args)
	result, _ := arguments.Get(0).(sql.Result)
	return result, arguments.Error(1)
}

func (m *MockDBSession) BeginTx(ctx context.Context, opts *sql.TxOptions) (Transaction, error) {
	arguments := m.Called(ctx, opts)
	return arguments.Get(0).(Transaction), arguments.Error(1)
}

// 测试用错误类型
type BusinessError struct {
	Message string
}

func (e *BusinessError) Error() string {
	return e.Message
}

// TestTransactionContext 测试事务上下文功能
func TestTransactionContext(t *testing.T) {
	t.Run("WithTransaction 和 GetTransaction", func(t *testing.T) {
		// 创建模拟事务
		mockTx := &MockTransaction{id: 1, level: 0, status: "active"}

		// 测试 WithTransaction 和 GetTransaction
		ctx := context.Background()
		txCtx := WithTransaction(ctx, mockTx)

		// 验证可以获取事务
		tx, ok := GetTransaction(txCtx)
		assert.True(t, ok)
		assert.Equal(t, uint64(1), tx.ID())
		assert.Equal(t, 0, tx.Level())
		assert.Equal(t, "active", tx.Status())

		// 验证原始上下文没有事务
		_, ok = GetTransaction(ctx)
		assert.False(t, ok)
	})

	t.Run("RequireTransaction 成功", func(t *testing.T) {
		mockTx := &MockTransaction{id: 1, level: 0, status: "active"}
		ctx := WithTransaction(context.Background(), mockTx)

		tx, err := RequireTransaction(ctx)
		assert.NoError(t, err)
		assert.Equal(t, uint64(1), tx.ID())
	})

	t.Run("RequireTransaction 失败", func(t *testing.T) {
		ctx := context.Background()

		tx, err := RequireTransaction(ctx)
		assert.Error(t, err)
		assert.Nil(t, tx)
		assert.Contains(t, err.Error(), "事务未在当前上下文中找到")
	})

	t.Run("IsTransactionActive", func(t *testing.T) {
		mockTx := &MockTransaction{id: 1, level: 0, status: "active"}
		ctx := WithTransaction(context.Background(), mockTx)

		assert.True(t, IsTransactionActive(ctx))
		assert.False(t, IsTransactionActive(context.Background()))
	})
}

// TestTransactionTemplate 测试基础事务模板
func TestTransactionTemplate(t *testing.T) {
	t.Run("正常提交", func(t *testing.T) {
		// 创建模拟会话和事务
		mockSession := new(MockDBSession)
		mockTx := &MockTransaction{id: 1, level: 0, status: "active"}

		// 设置期望行为
		mockSession.On("BeginTx", mock.Anything, mock.Anything).Return(mockTx, nil)
		mockTx.On("Commit").Return(nil)

		// 执行事务模板
		result, err := TransactionTemplate(
			context.Background(),
			mockSession,
			func(ctx context.Context) (interface{}, error) {
				tx, ok := GetTransaction(ctx)
				assert.True(t, ok)
				assert.Equal(t, uint64(1), tx.ID())
				return "success", nil
			},
		)

		// 验证结果
		assert.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, "committed", mockTx.status)
		mockSession.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})

	t.Run("业务错误自动回滚", func(t *testing.T) {
		// 创建模拟会话和事务
		mockSession := new(MockDBSession)
		mockTx := &MockTransaction{id: 1, level: 0, status: "active"}

		// 设置期望行为
		mockSession.On("BeginTx", mock.Anything, mock.Anything).Return(mockTx, nil)
		mockTx.On("Rollback").Return(nil)

		// 定义业务错误
		businessErr := &BusinessError{Message: "业务检查失败"}

		// 执行事务模板
		result, err := TransactionTemplate(
			context.Background(),
			mockSession,
			func(ctx context.Context) (interface{}, error) {
				return nil, businessErr
			},
		)

		// 验证结果
		assert.Error(t, err)
		assert.Equal(t, businessErr, err)
		assert.Nil(t, result)
		assert.Equal(t, "rolledback", mockTx.status)
		mockSession.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})

	t.Run("提交失败", func(t *testing.T) {
		// 创建模拟会话和事务
		mockSession := new(MockDBSession)
		mockTx := &MockTransaction{id: 1, level: 0, status: "active"}

		// 设置期望行为
		mockSession.On("BeginTx", mock.Anything, mock.Anything).Return(mockTx, nil)
		mockTx.On("Commit").Return(errors.New("提交事务失败"))

		// 执行事务模板
		result, err := TransactionTemplate(
			context.Background(),
			mockSession,
			func(ctx context.Context) (interface{}, error) {
				return "success", nil
			},
		)

		// 验证结果
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "事务提交失败")
		mockSession.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})

	t.Run("已有事务场景", func(t *testing.T) {
		// 创建模拟事务
		mockTx := &MockTransaction{id: 1, level: 0, status: "active"}

		// 创建带有事务的上下文
		ctx := WithTransaction(context.Background(), mockTx)
		mockSession := new(MockDBSession)

		// 执行事务模板
		result, err := TransactionTemplate(
			ctx,
			mockSession,
			func(txCtx context.Context) (interface{}, error) {
				// 验证上下文中有事务
				tx, ok := GetTransaction(txCtx)
				assert.True(t, ok)
				assert.Equal(t, uint64(1), tx.ID())
				return "reused tx", nil
			},
		)

		// 验证结果 - 不应该调用 BeginTx/Commit/Rollback
		assert.NoError(t, err)
		assert.Equal(t, "reused tx", result)
		mockSession.AssertNotCalled(t, "BeginTx")
	})
}

// TestTransactionTemplateV2 测试增强事务模板（带钩子）
func TestTransactionTemplateV2(t *testing.T) {
	t.Run("成功提交带钩子", func(t *testing.T) {
		// 创建模拟会话和事务
		mockSession := new(MockDBSession)
		mockTx := &MockTransaction{id: 2, level: 0, status: "active"}

		// 设置期望行为
		mockSession.On("BeginTx", mock.Anything, mock.Anything).Return(mockTx, nil)
		mockTx.On("Commit").Return(nil)

		// 跟踪钩子调用
		beforeCommitCalled := false
		afterCommitCalled := false

		// 执行事务模板
		result, err := TransactionTemplateV2(
			context.Background(),
			mockSession,
			func(ctx context.Context) (interface{}, error) {
				return "hook success", nil
			},
			func(ctx context.Context, tx Transaction) {
				beforeCommitCalled = true
				assert.Equal(t, uint64(2), tx.ID())
			},
			func(ctx context.Context, tx Transaction) {
				afterCommitCalled = true
				assert.Equal(t, "committed", tx.Status())
			},
			nil, // beforeRollback
			nil, // afterRollback
			func(format string, args ...interface{}) {}, // logger
		)

		// 验证结果
		assert.NoError(t, err)
		assert.Equal(t, "hook success", result)
		assert.True(t, beforeCommitCalled)
		assert.True(t, afterCommitCalled)
		mockSession.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})

	t.Run("回滚带钩子", func(t *testing.T) {
		// 创建模拟会话和事务
		mockSession := new(MockDBSession)
		mockTx := &MockTransaction{id: 3, level: 0, status: "active"}

		// 设置期望行为
		mockSession.On("BeginTx", mock.Anything, mock.Anything).Return(mockTx, nil)
		mockTx.On("Rollback").Return(nil)

		// 跟踪钩子调用
		beforeRollbackCalled := false
		afterRollbackCalled := false

		// 执行事务模板
		result, err := TransactionTemplateV2(
			context.Background(),
			mockSession,
			func(ctx context.Context) (interface{}, error) {
				return nil, errors.New("业务错误")
			},
			nil, // beforeCommit
			nil, // afterCommit
			func(ctx context.Context, tx Transaction) {
				beforeRollbackCalled = true
				assert.Equal(t, uint64(3), tx.ID())
			},
			func(ctx context.Context, tx Transaction) {
				afterRollbackCalled = true
				assert.Equal(t, "rolledback", tx.Status())
			},
			func(format string, args ...interface{}) {}, // logger
		)

		// 验证结果
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, beforeRollbackCalled)
		assert.True(t, afterRollbackCalled)
		mockSession.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})

	t.Run("日志输出", func(t *testing.T) {
		// 创建模拟会话和事务
		mockSession := new(MockDBSession)
		mockTx := &MockTransaction{id: 4, level: 1, status: "active"}

		// 设置期望行为
		mockSession.On("BeginTx", mock.Anything, mock.Anything).Return(mockTx, nil)
		mockTx.On("Commit").Return(nil)

		// 跟踪日志输出
		var logs []string
		logger := func(format string, args ...interface{}) {
			logs = append(logs, format)
		}

		// 执行事务模板
		_, err := TransactionTemplateV2(
			context.Background(),
			mockSession,
			func(ctx context.Context) (interface{}, error) {
				return "logging test", nil
			},
			nil, // beforeCommit
			nil, // afterCommit
			nil, // beforeRollback
			nil, // afterRollback
			logger,
		)

		// 验证结果
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(logs), 2) // 至少应有开启和提交的日志
		assert.Contains(t, logs[0], "开启新事务")
		assert.Contains(t, logs[len(logs)-1], "事务已提交")
		mockSession.AssertExpectations(t)
		mockTx.AssertExpectations(t)
	})
}

// TestIntegrationTransactionPropagation 测试不同传播行为的集成
// 注意：这是一个集成测试，依赖于实际的数据库连接
func TestIntegrationTransactionPropagation(t *testing.T) {
	// 跳过集成测试，除非显式启用
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 实现真实数据库连接的传播行为测试
	// 这里可以根据实际数据库类型和驱动进行补充

	// 示例：测试事务传播行为
	t.Run("事务传播行为测试", func(t *testing.T) {
		// 创建真实的数据库连接
		// 注意：这需要配置真实的数据库连接信息
		// 可以通过环境变量或配置文件来配置

		// 测试嵌套事务的传播行为
		// 1. 外层事务开启
		// 2. 内层事务开启（应该复用外层事务）
		// 3. 内层事务提交（应该不影响外层事务）
		// 4. 外层事务回滚（应该回滚所有操作）

		t.Skip("需要配置真实数据库连接才能运行此测试")
	})
}

// Benchmark 性能测试
func BenchmarkTransactionTemplate(b *testing.B) {
	// 创建模拟会话和事务
	mockSession := new(MockDBSession)
	mockTx := &MockTransaction{id: 1, level: 0, status: "active"}
	mockSession.On("BeginTx", mock.Anything, mock.Anything).Return(mockTx, nil)
	mockTx.On("Commit").Return(nil)

	// 执行基准测试
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = TransactionTemplate(ctx, mockSession, func(ctx context.Context) (interface{}, error) {
			return nil, nil
		})
	}
}
