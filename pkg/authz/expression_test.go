package authz

import (
	"testing"
)

func TestExpressionEngine(t *testing.T) {
	// 创建模拟身份
	identity := &MockIdentity{
		ID:       "user123",
		Username: "admin",
		Roles:    []string{"admin"},
		Claims: map[string]interface{}{
			"department": "IT",
			"level":      5,
			"active":     true,
		},
	}

	// 创建表达式引擎
	engine := NewExpressionEngine(identity)

	// 测试简单表达式
	result, err := engine.Evaluate(`user.role == "admin"`, identity)
	if err != nil {
		t.Errorf("评估表达式失败: %v", err)
	}
	if !result {
		t.Errorf("期望结果为true，实际得到false")
	}

	// 测试复杂表达式
	result, err = engine.Evaluate(`user.role == "admin" && user.department == "IT"`, identity)
	if err != nil {
		t.Errorf("评估表达式失败: %v", err)
	}
	if !result {
		t.Errorf("期望结果为true，实际得到false")
	}

	// 测试角色检查
	result, err = engine.Evaluate(`user.role == "user"`, identity)
	if err != nil {
		t.Errorf("评估表达式失败: %v", err)
	}
	if result {
		t.Errorf("期望结果为false，实际得到true")
	}
}

func TestExpressionEngineWithDifferentIdentities(t *testing.T) {
	tests := []struct {
		name     string
		identity *MockIdentity
		expr     string
		expected bool
	}{
		{
			name: "管理员用户",
			identity: &MockIdentity{
				ID:       "admin1",
				Username: "admin",
				Roles:    []string{"admin"},
				Claims: map[string]interface{}{
					"department": "IT",
					"level":      10,
				},
			},
			expr:     `user.role == "admin"`,
			expected: true,
		},
		{
			name: "普通用户",
			identity: &MockIdentity{
				ID:       "user1",
				Username: "user",
				Roles:    []string{"user"},
				Claims: map[string]interface{}{
					"department": "Sales",
					"level":      2,
				},
			},
			expr:     `user.role == "user"`,
			expected: true,
		},
		{
			name: "经理用户",
			identity: &MockIdentity{
				ID:       "manager1",
				Username: "manager",
				Roles:    []string{"manager"},
				Claims: map[string]interface{}{
					"department": "Engineering",
					"level":      7,
				},
			},
			expr:     `user.role == "manager" && user.level > 5`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewExpressionEngine(tt.identity)
			result, err := engine.Evaluate(tt.expr, tt.identity)
			if err != nil {
				t.Errorf("评估表达式失败: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("期望结果 %v，实际得到 %v", tt.expected, result)
			}
		})
	}
}

func TestExpressionEngineEdgeCases(t *testing.T) {
	identity := &MockIdentity{
		ID:       "user123",
		Username: "admin",
		Roles:    []string{"admin"},
		Claims: map[string]interface{}{
			"department": "IT",
			"level":      5,
			"active":     true,
		},
	}

	engine := NewExpressionEngine(identity)

	// 测试空表达式
	result, err := engine.Evaluate("", identity)
	if err != nil {
		t.Errorf("评估空表达式失败: %v", err)
	}
	if !result {
		t.Errorf("空表达式应该返回true")
	}

	// 测试只有空格的表达式
	result, err = engine.Evaluate("   ", identity)
	if err != nil {
		t.Errorf("评估空白表达式失败: %v", err)
	}
	if !result {
		t.Errorf("空白表达式应该返回true")
	}
}

func TestExpressionEnginePerformance(t *testing.T) {
	identity := &MockIdentity{
		ID:       "user123",
		Username: "admin",
		Roles:    []string{"admin"},
		Claims: map[string]interface{}{
			"department": "IT",
			"level":      8,
			"active":     true,
		},
	}

	engine := NewExpressionEngine(identity)
	complexExpr := `user.role == "admin" && user.department == "IT" && user.level > 5`

	// 运行多次评估以测试性能
	for i := 0; i < 100; i++ {
		result, err := engine.Evaluate(complexExpr, identity)
		if err != nil {
			t.Fatalf("评估表达式失败: %v", err)
		}
		if !result {
			t.Fatalf("期望结果为true，实际得到false")
		}
	}
}
