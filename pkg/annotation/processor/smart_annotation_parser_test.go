package processor

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/your-org/gRain/pkg/annotation/types"
)

func TestSmartAnnotationParser_ParseJSONFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []types.AnnotationAttribute
		hasError bool
	}{
		{
			name:  "简单JSON对象",
			input: `{"method":"GET", "path":"/users"}`,
			expected: []types.AnnotationAttribute{
				{Name: "method", Value: "GET"},
				{Name: "path", Value: "/users"},
			},
			hasError: false,
		},
		{
			name:  "包含数组的JSON",
			input: `{"method":"POST", "middleware":["auth", "log"]}`,
			expected: []types.AnnotationAttribute{
				{Name: "method", Value: "POST"},
				{Name: "middleware", Value: []interface{}{"auth", "log"}},
			},
			hasError: false,
		},
		{
			name:  "包含布尔值和数字",
			input: `{"auth":true, "timeout":30, "rate":1.5}`,
			expected: []types.AnnotationAttribute{
				{Name: "auth", Value: true},
				{Name: "timeout", Value: float64(30)}, // JSON解析数字为float64
				{Name: "rate", Value: 1.5},
			},
			hasError: false,
		},
		{
			name:     "无效JSON格式",
			input:    `{method:"GET", path:"/users"}`, // 缺少引号
			expected: nil,
			hasError: true,
		},
		{
			name:     "非JSON对象",
			input:    `method="GET"`,
			expected: nil,
			hasError: true,
		},
	}

	parser := NewSmartAnnotationParser(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.parseJSONFormat(tt.input)

			if tt.hasError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, len(tt.expected))

				// 转换为map以便比较（顺序可能不同）
				resultMap := make(map[string]interface{})
				expectedMap := make(map[string]interface{})

				for _, attr := range result {
					resultMap[attr.Name] = attr.Value
				}
				for _, attr := range tt.expected {
					expectedMap[attr.Name] = attr.Value
				}

				assert.Equal(t, expectedMap, resultMap)
			}
		})
	}
}

func TestSmartAnnotationParser_ParseLegacyFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []types.AnnotationAttribute
	}{
		{
			name:  "简单键值对",
			input: `method="GET", path="/users"`,
			expected: []types.AnnotationAttribute{
				{Name: "method", Value: "GET"},
				{Name: "path", Value: "/users"},
			},
		},
		{
			name:  "包含布尔值",
			input: `method="POST", auth=true`,
			expected: []types.AnnotationAttribute{
				{Name: "method", Value: "POST"},
				{Name: "auth", Value: true},
			},
		},
		{
			name:  "包含数字",
			input: `timeout=30, rate=1.5`,
			expected: []types.AnnotationAttribute{
				{Name: "timeout", Value: int64(30)},
				{Name: "rate", Value: 1.5},
			},
		},
		{
			name:  "包含数组",
			input: `middleware=["auth", "log"]`,
			expected: []types.AnnotationAttribute{
				{Name: "middleware", Value: []interface{}{"auth", "log"}},
			},
		},
		{
			name:  "标志格式",
			input: `GET, auth`,
			expected: []types.AnnotationAttribute{
				{Name: "GET", Value: true},
				{Name: "auth", Value: true},
			},
		},
		{
			name:  "混合格式",
			input: `method="GET", auth, timeout=30`,
			expected: []types.AnnotationAttribute{
				{Name: "method", Value: "GET"},
				{Name: "auth", Value: true},
				{Name: "timeout", Value: int64(30)},
			},
		},
	}

	parser := NewSmartAnnotationParser(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.parseLegacyFormat(tt.input)

			assert.NoError(t, err)
			assert.Len(t, result, len(tt.expected))

			// 转换为map以便比较
			resultMap := make(map[string]interface{})
			expectedMap := make(map[string]interface{})

			for _, attr := range result {
				resultMap[attr.Name] = attr.Value
			}
			for _, attr := range tt.expected {
				expectedMap[attr.Name] = attr.Value
			}

			assert.Equal(t, expectedMap, resultMap)
		})
	}
}

func TestSmartAnnotationParser_ParseAnnotationAttributes(t *testing.T) {
	tests := []struct {
		name         string
		config       *SmartAnnotationParserConfig
		input        string
		expectFormat string // "JSON" 或 "Legacy"
		hasError     bool
	}{
		{
			name:         "JSON格式优先",
			config:       nil, // 默认配置
			input:        `{"method":"GET", "path":"/users"}`,
			expectFormat: "JSON",
			hasError:     false,
		},
		{
			name:         "遗留格式回退",
			config:       nil, // 默认配置
			input:        `method="GET", path="/users"`,
			expectFormat: "Legacy",
			hasError:     false,
		},
		{
			name: "严格模式拒绝遗留格式",
			config: &SmartAnnotationParserConfig{
				StrictMode:    true,
				LegacySupport: false,
			},
			input:        `method="GET", path="/users"`,
			expectFormat: "",
			hasError:     true,
		},
		{
			name: "关闭遗留支持",
			config: &SmartAnnotationParserConfig{
				StrictMode:    false,
				LegacySupport: false,
			},
			input:        `method="GET", path="/users"`,
			expectFormat: "",
			hasError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewSmartAnnotationParser(tt.config)
			result, err := parser.ParseAnnotationAttributes(tt.input)

			if tt.hasError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, len(result) > 0)
			}
		})
	}
}

func TestHTTPMethodParser(t *testing.T) {
	parser := &HTTPMethodParser{}

	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{"有效GET方法", "get", "GET", false},
		{"有效POST方法", "POST", "POST", false},
		{"无效方法", "INVALID", "", true},
		{"空方法", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)

				// 测试验证
				assert.NoError(t, parser.Validate(result))
			}
		})
	}
}

func TestPathParser(t *testing.T) {
	parser := &PathParser{}

	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{"绝对路径", "/users", "/users", false},
		{"相对路径", "users", "/users", false},
		{"复杂路径", "/api/v1/users/{id}", "/api/v1/users/{id}", false},
		{"空路径", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)

				// 测试验证
				assert.NoError(t, parser.Validate(result))
			}
		})
	}
}

func TestMiddlewareParser(t *testing.T) {
	parser := &MiddlewareParser{}

	tests := []struct {
		name     string
		input    string
		expected []string
		hasError bool
	}{
		{"单个中间件", "auth", []string{"auth"}, false},
		{"逗号分隔", "auth, log, cors", []string{"auth", "log", "cors"}, false},
		{"JSON数组", `["auth", "log"]`, []string{"auth", "log"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.input)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)

				// 测试验证
				assert.NoError(t, parser.Validate(result))
			}
		})
	}
}

func TestMigrationHelper_ConvertLegacyToJSON(t *testing.T) {
	parser := NewSmartAnnotationParser(nil)
	helper := NewMigrationHelper(parser)

	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
		hasError bool
	}{
		{
			name:  "简单转换",
			input: `method="GET", path="/users"`,
			expected: map[string]interface{}{
				"method": "GET",
				"path":   "/users",
			},
			hasError: false,
		},
		{
			name:  "包含布尔值",
			input: `method="POST", auth=true, timeout=30`,
			expected: map[string]interface{}{
				"method":  "POST",
				"auth":    true,
				"timeout": int64(30),
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := helper.ConvertLegacyToJSON(tt.input)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// 解析JSON以验证
				var jsonObj map[string]interface{}
				require.NoError(t, json.Unmarshal([]byte(result), &jsonObj))
				assert.Equal(t, tt.expected, jsonObj)
			}
		})
	}
}

func TestMigrationHelper_ValidateAndSuggest(t *testing.T) {
	parser := NewSmartAnnotationParser(nil)
	helper := NewMigrationHelper(parser)

	tests := []struct {
		name             string
		input            string
		expectValid      bool
		expectFormat     string
		expectSuggestion bool
	}{
		{
			name:             "有效JSON格式",
			input:            `{"method":"GET", "path":"/users"}`,
			expectValid:      true,
			expectFormat:     "JSON",
			expectSuggestion: false,
		},
		{
			name:             "有效遗留格式",
			input:            `method="GET", path="/users"`,
			expectValid:      true,
			expectFormat:     "Legacy",
			expectSuggestion: true, // 建议迁移到JSON
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := helper.ValidateAndSuggest(tt.input)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectValid, result.IsValid)
			assert.Equal(t, tt.expectFormat, result.Format)

			if tt.expectSuggestion {
				assert.True(t, len(result.Suggestions) > 0)
			}
		})
	}
}

func TestSmartAnnotationParser_GetSupportedFormats(t *testing.T) {
	// 测试默认配置
	parser := NewSmartAnnotationParser(nil)
	formats := parser.GetSupportedFormats()

	assert.Contains(t, formats, "JSON (推荐)")
	assert.Contains(t, formats, "遗留格式 (兼容)")

	// 测试严格模式
	strictParser := NewSmartAnnotationParser(&SmartAnnotationParserConfig{
		StrictMode:    true,
		LegacySupport: false,
	})
	strictFormats := strictParser.GetSupportedFormats()

	assert.Contains(t, strictFormats, "JSON (推荐)")
	assert.NotContains(t, strictFormats, "遗留格式 (兼容)")
}

// 基准测试
func BenchmarkSmartAnnotationParser_JSON(b *testing.B) {
	parser := NewSmartAnnotationParser(nil)
	jsonStr := `{"method":"GET", "path":"/api/v1/users", "middleware":["auth", "log", "cors"], "timeout":30}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseAnnotationAttributes(jsonStr)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSmartAnnotationParser_Legacy(b *testing.B) {
	parser := NewSmartAnnotationParser(nil)
	legacyStr := `method="GET", path="/api/v1/users", middleware=["auth", "log", "cors"], timeout=30`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseAnnotationAttributes(legacyStr)
		if err != nil {
			b.Fatal(err)
		}
	}
}
