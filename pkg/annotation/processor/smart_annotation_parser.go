// Package processor 智能注解属性解析器
// 支持JSON标准化格式和向下兼容模式
package processor

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/your-org/gRain/pkg/annotation/types"
)

// SmartAnnotationParser 智能注解属性解析器
type SmartAnnotationParser struct {
	strictMode      bool                       // 严格模式：只支持JSON格式
	legacySupport   bool                       // 遗留支持：兼容旧格式
	customParsers   map[string]AttributeParser // 自定义解析器
	validationRules map[string]ValidationRule  // 验证规则
}

// AttributeParser 属性解析器接口
type AttributeParser interface {
	Parse(value string) (interface{}, error)
	Validate(value interface{}) error
}

// ValidationRule 验证规则接口
type ValidationRule interface {
	Validate(key string, value interface{}) error
}

// SmartAnnotationParserConfig 配置
type SmartAnnotationParserConfig struct {
	StrictMode    bool                       `json:"strict_mode"`
	LegacySupport bool                       `json:"legacy_support"`
	CustomParsers map[string]AttributeParser `json:"-"`
	Validators    map[string]ValidationRule  `json:"-"`
}

// NewSmartAnnotationParser 创建智能注解解析器
func NewSmartAnnotationParser(config *SmartAnnotationParserConfig) *SmartAnnotationParser {
	if config == nil {
		config = &SmartAnnotationParserConfig{
			StrictMode:    false, // 默认非严格模式，支持向下兼容
			LegacySupport: true,  // 默认支持遗留格式
		}
	}

	parser := &SmartAnnotationParser{
		strictMode:      config.StrictMode,
		legacySupport:   config.LegacySupport,
		customParsers:   make(map[string]AttributeParser),
		validationRules: make(map[string]ValidationRule),
	}

	// 注册默认解析器
	parser.registerDefaultParsers()

	// 注册自定义解析器
	if config.CustomParsers != nil {
		for name, customParser := range config.CustomParsers {
			parser.customParsers[name] = customParser
		}
	}

	// 注册验证规则
	if config.Validators != nil {
		for name, validator := range config.Validators {
			parser.validationRules[name] = validator
		}
	}

	return parser
}

// ParseAnnotationAttributes 解析注解属性（主入口）
func (p *SmartAnnotationParser) ParseAnnotationAttributes(attrStr string) ([]types.AnnotationAttribute, error) {
	if strings.TrimSpace(attrStr) == "" {
		return nil, nil
	}

	// 尝试JSON格式解析
	if jsonAttrs, err := p.parseJSONFormat(attrStr); err == nil {
		return jsonAttrs, nil
	}

	// 如果严格模式，不支持遗留格式
	if p.strictMode {
		return nil, fmt.Errorf("严格模式下只支持JSON格式，解析失败: %s", attrStr)
	}

	// 如果支持遗留格式，尝试遗留格式解析
	if p.legacySupport {
		return p.parseLegacyFormat(attrStr)
	}

	return nil, fmt.Errorf("无法解析注解属性: %s", attrStr)
}

// parseJSONFormat 解析JSON格式的注解属性
// 格式: {"method":"GET", "path":"/users", "middleware":["auth", "log"]}
func (p *SmartAnnotationParser) parseJSONFormat(attrStr string) ([]types.AnnotationAttribute, error) {
	// 清理字符串，确保是有效的JSON
	attrStr = strings.TrimSpace(attrStr)

	// 检查是否是JSON对象格式
	if !strings.HasPrefix(attrStr, "{") || !strings.HasSuffix(attrStr, "}") {
		return nil, fmt.Errorf("不是有效的JSON对象格式")
	}

	// 解析JSON
	var jsonObj map[string]interface{}
	if err := json.Unmarshal([]byte(attrStr), &jsonObj); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	// 转换为注解属性
	var attributes []types.AnnotationAttribute
	for key, value := range jsonObj {
		// 应用自定义解析器
		if parser, exists := p.customParsers[key]; exists {
			if parsedValue, err := parser.Parse(fmt.Sprintf("%v", value)); err == nil {
				value = parsedValue
			}
		}

		// 验证属性值
		if validator, exists := p.validationRules[key]; exists {
			if err := validator.Validate(key, value); err != nil {
				return nil, fmt.Errorf("属性 %s 验证失败: %w", key, err)
			}
		}

		attributes = append(attributes, types.AnnotationAttribute{
			Name:  key,
			Value: value,
		})
	}

	return attributes, nil
}

// parseLegacyFormat 解析遗留格式的注解属性（向下兼容）
// 格式: method="GET", path="/users", auth=true
func (p *SmartAnnotationParser) parseLegacyFormat(attrStr string) ([]types.AnnotationAttribute, error) {
	// 使用改进的复杂属性解析器
	return p.parseComplexAttributesImproved(attrStr), nil
}

// parseComplexAttributesImproved 改进的复杂属性解析器
func (p *SmartAnnotationParser) parseComplexAttributesImproved(attrStr string) []types.AnnotationAttribute {
	if strings.TrimSpace(attrStr) == "" {
		return nil
	}

	var attributes []types.AnnotationAttribute

	// 使用正则表达式进行更准确的解析
	// 支持: key="value", key='value', key=value, key={...}, key=[...]
	re := regexp.MustCompile(`(\w+)\s*=\s*("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|\{[^}]*\}|\[[^\]]*\]|[^,]+)|(\w+)(?:\s*,\s*|$)`)

	matches := re.FindAllStringSubmatch(attrStr, -1)

	for _, match := range matches {
		if len(match) >= 3 && match[1] != "" && match[2] != "" {
			// 键值对格式: key=value
			key := strings.TrimSpace(match[1])
			value := strings.TrimSpace(match[2])

			// 去除引号
			if len(value) >= 2 {
				if (value[0] == '"' && value[len(value)-1] == '"') ||
					(value[0] == '\'' && value[len(value)-1] == '\'') {
					value = value[1 : len(value)-1]
				}
			}

			// 解析值
			parsedValue := p.parseAttributeValueImproved(value)

			attributes = append(attributes, types.AnnotationAttribute{
				Name:  key,
				Value: parsedValue,
			})
		} else if len(match) >= 4 && match[3] != "" {
			// 标志格式: flag
			flag := strings.TrimSpace(match[3])
			attributes = append(attributes, types.AnnotationAttribute{
				Name:  flag,
				Value: true,
			})
		}
	}

	return attributes
}

// parseAttributeValueImproved 改进的属性值解析
func (p *SmartAnnotationParser) parseAttributeValueImproved(value string) interface{} {
	value = strings.TrimSpace(value)

	// 布尔值
	if value == "true" {
		return true
	} else if value == "false" {
		return false
	}

	// 数字
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}

	// JSON数组或对象
	if (strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}")) ||
		(strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]")) {
		var result interface{}
		if err := json.Unmarshal([]byte(value), &result); err == nil {
			return result
		}
	}

	// 字符串数组（简单格式）
	if strings.Contains(value, ",") && !strings.HasPrefix(value, "{") && !strings.HasPrefix(value, "[") {
		parts := strings.Split(value, ",")
		var strArray []string
		for _, part := range parts {
			strArray = append(strArray, strings.TrimSpace(part))
		}
		return strArray
	}

	// 默认字符串
	return value
}

// registerDefaultParsers 注册默认解析器
func (p *SmartAnnotationParser) registerDefaultParsers() {
	// HTTP方法解析器
	p.customParsers["method"] = &HTTPMethodParser{}

	// 路径解析器
	p.customParsers["path"] = &PathParser{}

	// 中间件解析器
	p.customParsers["middleware"] = &MiddlewareParser{}

	// 权限解析器
	p.customParsers["roles"] = &RolesParser{}
	p.customParsers["permissions"] = &PermissionsParser{}
}

// HTTPMethodParser HTTP方法解析器
type HTTPMethodParser struct{}

func (p *HTTPMethodParser) Parse(value string) (interface{}, error) {
	method := strings.ToUpper(strings.TrimSpace(value))
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, validMethod := range validMethods {
		if method == validMethod {
			return method, nil
		}
	}

	return nil, fmt.Errorf("无效的HTTP方法: %s", value)
}

func (p *HTTPMethodParser) Validate(value interface{}) error {
	if method, ok := value.(string); ok {
		_, err := p.Parse(method)
		return err
	}
	return fmt.Errorf("HTTP方法必须是字符串")
}

// PathParser 路径解析器
type PathParser struct{}

func (p *PathParser) Parse(value string) (interface{}, error) {
	path := strings.TrimSpace(value)
	if path == "" {
		return nil, fmt.Errorf("路径不能为空")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path, nil
}

func (p *PathParser) Validate(value interface{}) error {
	if path, ok := value.(string); ok {
		_, err := p.Parse(path)
		return err
	}
	return fmt.Errorf("路径必须是字符串")
}

// MiddlewareParser 中间件解析器
type MiddlewareParser struct{}

func (p *MiddlewareParser) Parse(value string) (interface{}, error) {
	// 支持字符串和数组格式
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		var middlewares []string
		if err := json.Unmarshal([]byte(value), &middlewares); err != nil {
			return nil, fmt.Errorf("中间件数组解析失败: %w", err)
		}
		return middlewares, nil
	}

	// 逗号分隔的字符串
	if strings.Contains(value, ",") {
		parts := strings.Split(value, ",")
		var middlewares []string
		for _, part := range parts {
			middlewares = append(middlewares, strings.TrimSpace(part))
		}
		return middlewares, nil
	}

	// 单个中间件
	return []string{strings.TrimSpace(value)}, nil
}

func (p *MiddlewareParser) Validate(value interface{}) error {
	switch v := value.(type) {
	case string:
		return nil
	case []interface{}:
		for _, item := range v {
			if _, ok := item.(string); !ok {
				return fmt.Errorf("中间件必须是字符串")
			}
		}
		return nil
	case []string:
		return nil
	default:
		return fmt.Errorf("中间件必须是字符串或字符串数组")
	}
}

// RolesParser 角色解析器
type RolesParser struct{}

func (p *RolesParser) Parse(value string) (interface{}, error) {
	return (&MiddlewareParser{}).Parse(value) // 复用中间件解析逻辑
}

func (p *RolesParser) Validate(value interface{}) error {
	return (&MiddlewareParser{}).Validate(value) // 复用中间件验证逻辑
}

// PermissionsParser 权限解析器
type PermissionsParser struct{}

func (p *PermissionsParser) Parse(value string) (interface{}, error) {
	return (&MiddlewareParser{}).Parse(value) // 复用中间件解析逻辑
}

func (p *PermissionsParser) Validate(value interface{}) error {
	return (&MiddlewareParser{}).Validate(value) // 复用中间件验证逻辑
}

// GetSupportedFormats 获取支持的格式说明
func (p *SmartAnnotationParser) GetSupportedFormats() map[string]string {
	formats := make(map[string]string)

	// JSON格式（推荐）
	formats["JSON (推荐)"] = `{"method":"GET", "path":"/users", "middleware":["auth", "log"]}`

	if p.legacySupport {
		// 遗留格式（兼容）
		formats["遗留格式 (兼容)"] = `method="GET", path="/users", middleware=["auth", "log"]`
	}

	return formats
}

// MigrationHelper 迁移助手
type MigrationHelper struct {
	parser *SmartAnnotationParser
}

// NewMigrationHelper 创建迁移助手
func NewMigrationHelper(parser *SmartAnnotationParser) *MigrationHelper {
	return &MigrationHelper{parser: parser}
}

// ConvertLegacyToJSON 将遗留格式转换为JSON格式
func (m *MigrationHelper) ConvertLegacyToJSON(legacyStr string) (string, error) {
	// 解析遗留格式
	attrs := m.parser.parseComplexAttributesImproved(legacyStr)
	if len(attrs) == 0 {
		return "", fmt.Errorf("无法解析遗留格式: %s", legacyStr)
	}

	// 转换为JSON对象
	jsonObj := make(map[string]interface{})
	for _, attr := range attrs {
		jsonObj[attr.Name] = attr.Value
	}

	// 序列化为JSON
	jsonBytes, err := json.Marshal(jsonObj)
	if err != nil {
		return "", fmt.Errorf("JSON序列化失败: %w", err)
	}

	return string(jsonBytes), nil
}

// ValidateAndSuggest 验证并提供建议
func (m *MigrationHelper) ValidateAndSuggest(attrStr string) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:     true,
		Format:      "unknown",
		Suggestions: []string{},
	}

	// 尝试解析
	attrs, err := m.parser.ParseAnnotationAttributes(attrStr)
	if err != nil {
		result.IsValid = false
		result.Error = err.Error()

		// 提供修复建议
		if jsonStr, convertErr := m.ConvertLegacyToJSON(attrStr); convertErr == nil {
			result.Suggestions = append(result.Suggestions,
				fmt.Sprintf("建议使用JSON格式: %s", jsonStr))
		}

		return result, nil
	}

	// 检测格式
	if strings.HasPrefix(strings.TrimSpace(attrStr), "{") {
		result.Format = "JSON"
	} else {
		result.Format = "Legacy"

		// 建议迁移到JSON格式
		if jsonStr, err := m.ConvertLegacyToJSON(attrStr); err == nil {
			result.Suggestions = append(result.Suggestions,
				fmt.Sprintf("建议迁移到JSON格式: %s", jsonStr))
		}
	}

	result.ParsedAttributes = attrs
	return result, nil
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid          bool                        `json:"is_valid"`
	Format           string                      `json:"format"`
	Error            string                      `json:"error,omitempty"`
	Suggestions      []string                    `json:"suggestions,omitempty"`
	ParsedAttributes []types.AnnotationAttribute `json:"parsed_attributes,omitempty"`
}
