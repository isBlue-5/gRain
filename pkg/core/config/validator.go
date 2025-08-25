package config

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ConfigValidatorImpl 配置验证器实现
type ConfigValidatorImpl struct {
	// 验证规则
	rules map[string]ValidationRule

	// 验证统计
	validationCount int

	// 自定义验证器
	customValidators map[string]CustomValidator
}

// ValidationRule 验证规则
type ValidationRule struct {
	// 字段名称
	Field string

	// 验证类型
	Type string

	// 验证参数
	Params map[string]interface{}

	// 错误消息
	Message string
}

// CustomValidator 自定义验证器
type CustomValidator func(value interface{}, params map[string]interface{}) error

// NewConfigValidator 创建新的配置验证器
func NewConfigValidator() *ConfigValidatorImpl {
	v := &ConfigValidatorImpl{
		rules:            make(map[string]ValidationRule),
		customValidators: make(map[string]CustomValidator),
	}

	// 注册内置验证器
	v.registerBuiltinValidators()

	return v
}

// registerBuiltinValidators 注册内置验证器
func (v *ConfigValidatorImpl) registerBuiltinValidators() {
	// 字符串验证器
	v.customValidators["required"] = v.validateRequired
	v.customValidators["min_len"] = v.validateMinLength
	v.customValidators["max_len"] = v.validateMaxLength
	v.customValidators["pattern"] = v.validatePattern
	v.customValidators["email"] = v.validateEmail
	v.customValidators["url"] = v.validateURL

	// 数值验证器
	v.customValidators["min"] = v.validateMin
	v.customValidators["max"] = v.validateMax
	v.customValidators["range"] = v.validateRange

	// 时间验证器
	v.customValidators["future"] = v.validateFuture
	v.customValidators["past"] = v.validatePast

	// 文件验证器
	v.customValidators["file_exists"] = v.validateFileExists
	v.customValidators["dir_exists"] = v.validateDirExists
}

// Validate 验证配置
func (v *ConfigValidatorImpl) Validate(config interface{}) error {
	v.validationCount++

	val := reflect.ValueOf(config)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("config must be a struct, got %s", val.Kind())
	}

	return v.validateStruct(val)
}

// validateStruct 验证结构体
func (v *ConfigValidatorImpl) validateStruct(val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过未导出的字段
		if !field.CanInterface() {
			continue
		}

		// 获取验证标签
		validateTag := fieldType.Tag.Get("validate")
		if validateTag == "" {
			continue
		}

		// 验证字段
		if err := v.validateField(field, fieldType, validateTag); err != nil {
			return fmt.Errorf("field %s validation failed: %w", fieldType.Name, err)
		}
	}

	return nil
}

// validateField 验证字段
func (v *ConfigValidatorImpl) validateField(field reflect.Value, fieldType reflect.StructField, validateTag string) error {
	// 解析验证标签
	rules := v.parseValidationRules(validateTag)

	// 应用验证规则
	for _, rule := range rules {
		if err := v.applyValidationRule(field, fieldType, rule); err != nil {
			return err
		}
	}

	return nil
}

// parseValidationRules 解析验证规则
func (v *ConfigValidatorImpl) parseValidationRules(validateTag string) []string {
	rules := strings.Split(validateTag, ",")
	var result []string

	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule != "" {
			result = append(result, rule)
		}
	}

	return result
}

// applyValidationRule 应用验证规则
func (v *ConfigValidatorImpl) applyValidationRule(field reflect.Value, fieldType reflect.StructField, rule string) error {
	// 解析规则名称和参数
	ruleName, params := v.parseRuleWithParams(rule)

	// 检查是否有自定义验证器
	if validator, exists := v.customValidators[ruleName]; exists {
		return validator(field.Interface(), params)
	}

	// 应用内置验证规则
	return v.applyBuiltinRule(field, fieldType, ruleName, params)
}

// parseRuleWithParams 解析规则和参数
func (v *ConfigValidatorImpl) parseRuleWithParams(rule string) (string, map[string]interface{}) {
	parts := strings.Split(rule, ":")
	ruleName := parts[0]
	params := make(map[string]interface{})

	if len(parts) > 1 {
		paramStr := parts[1]
		paramPairs := strings.Split(paramStr, ",")

		for _, pair := range paramPairs {
			kv := strings.Split(pair, "=")
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])

				// 尝试转换为数值
				if intVal, err := strconv.Atoi(value); err == nil {
					params[key] = intVal
				} else if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
					params[key] = floatVal
				} else {
					params[key] = value
				}
			}
		}
	}

	return ruleName, params
}

// applyBuiltinRule 应用内置验证规则
func (v *ConfigValidatorImpl) applyBuiltinRule(field reflect.Value, fieldType reflect.StructField, ruleName string, params map[string]interface{}) error {
	switch ruleName {
	case "required":
		return v.validateRequired(field.Interface(), params)
	case "min_len":
		return v.validateMinLength(field.Interface(), params)
	case "max_len":
		return v.validateMaxLength(field.Interface(), params)
	case "min":
		return v.validateMin(field.Interface(), params)
	case "max":
		return v.validateMax(field.Interface(), params)
	default:
		return fmt.Errorf("unknown validation rule: %s", ruleName)
	}
}

// validateRequired 验证必填字段
func (v *ConfigValidatorImpl) validateRequired(value interface{}, params map[string]interface{}) error {
	if value == nil {
		return fmt.Errorf("field is required")
	}

	val := reflect.ValueOf(value)
	if val.Kind() == reflect.String && val.String() == "" {
		return fmt.Errorf("field is required")
	}

	if val.Kind() == reflect.Slice && val.Len() == 0 {
		return fmt.Errorf("field is required")
	}

	return nil
}

// validateMinLength 验证最小长度
func (v *ConfigValidatorImpl) validateMinLength(value interface{}, params map[string]interface{}) error {
	minLen, exists := params["value"]
	if !exists {
		return fmt.Errorf("min_len rule requires value parameter")
	}

	minLenInt, ok := minLen.(int)
	if !ok {
		return fmt.Errorf("min_len value must be integer")
	}

	val := reflect.ValueOf(value)
	if val.Kind() == reflect.String && val.Len() < minLenInt {
		return fmt.Errorf("field length must be at least %d", minLenInt)
	}

	if val.Kind() == reflect.Slice && val.Len() < minLenInt {
		return fmt.Errorf("field length must be at least %d", minLenInt)
	}

	return nil
}

// validateMaxLength 验证最大长度
func (v *ConfigValidatorImpl) validateMaxLength(value interface{}, params map[string]interface{}) error {
	maxLen, exists := params["value"]
	if !exists {
		return fmt.Errorf("max_len rule requires value parameter")
	}

	maxLenInt, ok := maxLen.(int)
	if !ok {
		return fmt.Errorf("max_len value must be integer")
	}

	val := reflect.ValueOf(value)
	if val.Kind() == reflect.String && val.Len() > maxLenInt {
		return fmt.Errorf("field length must be at most %d", maxLenInt)
	}

	if val.Kind() == reflect.Slice && val.Len() > maxLenInt {
		return fmt.Errorf("field length must be at most %d", maxLenInt)
	}

	return nil
}

// validateMin 验证最小值
func (v *ConfigValidatorImpl) validateMin(value interface{}, params map[string]interface{}) error {
	minVal, exists := params["value"]
	if !exists {
		return fmt.Errorf("min rule requires value parameter")
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Int() < minVal.(int64) {
			return fmt.Errorf("field value must be at least %v", minVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val.Uint() < minVal.(uint64) {
			return fmt.Errorf("field value must be at least %v", minVal)
		}
	case reflect.Float32, reflect.Float64:
		if val.Float() < minVal.(float64) {
			return fmt.Errorf("field value must be at least %v", minVal)
		}
	}

	return nil
}

// validateMax 验证最大值
func (v *ConfigValidatorImpl) validateMax(value interface{}, params map[string]interface{}) error {
	maxVal, exists := params["value"]
	if !exists {
		return fmt.Errorf("max rule requires value parameter")
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Int() > maxVal.(int64) {
			return fmt.Errorf("field value must be at most %v", maxVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val.Uint() > maxVal.(uint64) {
			return fmt.Errorf("field value must be at most %v", maxVal)
		}
	case reflect.Float32, reflect.Float64:
		if val.Float() > maxVal.(float64) {
			return fmt.Errorf("field value must be at most %v", maxVal)
		}
	}

	return nil
}

// validatePattern 验证正则表达式模式
func (v *ConfigValidatorImpl) validatePattern(value interface{}, params map[string]interface{}) error {
	pattern, exists := params["value"]
	if !exists {
		return fmt.Errorf("pattern rule requires value parameter")
	}

	patternStr, ok := pattern.(string)
	if !ok {
		return fmt.Errorf("pattern value must be string")
	}

	val := reflect.ValueOf(value)
	if val.Kind() != reflect.String {
		return fmt.Errorf("pattern validation only applies to string fields")
	}

	matched, err := regexp.MatchString(patternStr, val.String())
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	if !matched {
		return fmt.Errorf("field value does not match pattern: %s", patternStr)
	}

	return nil
}

// validateEmail 验证邮箱格式
func (v *ConfigValidatorImpl) validateEmail(value interface{}, params map[string]interface{}) error {
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.String {
		return fmt.Errorf("email validation only applies to string fields")
	}

	email := val.String()
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format: %s", email)
	}

	return nil
}

// validateURL 验证URL格式
func (v *ConfigValidatorImpl) validateURL(value interface{}, params map[string]interface{}) error {
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.String {
		return fmt.Errorf("url validation only applies to string fields")
	}

	url := val.String()
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)

	if !urlRegex.MatchString(url) {
		return fmt.Errorf("invalid URL format: %s", url)
	}

	return nil
}

// validateRange 验证数值范围
func (v *ConfigValidatorImpl) validateRange(value interface{}, params map[string]interface{}) error {
	minVal, minExists := params["min"]
	maxVal, maxExists := params["max"]

	if !minExists || !maxExists {
		return fmt.Errorf("range rule requires both min and max parameters")
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		valInt := val.Int()
		minInt := minVal.(int64)
		maxInt := maxVal.(int64)

		if valInt < minInt || valInt > maxInt {
			return fmt.Errorf("field value must be between %d and %d", minInt, maxInt)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		valUint := val.Uint()
		minUint := minVal.(uint64)
		maxUint := maxVal.(uint64)

		if valUint < minUint || valUint > maxUint {
			return fmt.Errorf("field value must be between %d and %d", minUint, maxUint)
		}
	case reflect.Float32, reflect.Float64:
		valFloat := val.Float()
		minFloat := minVal.(float64)
		maxFloat := maxVal.(float64)

		if valFloat < minFloat || valFloat > maxFloat {
			return fmt.Errorf("field value must be between %f and %f", minFloat, maxFloat)
		}
	}

	return nil
}

// validateFuture 验证未来时间
func (v *ConfigValidatorImpl) validateFuture(value interface{}, params map[string]interface{}) error {
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Struct || val.Type() != reflect.TypeOf(time.Time{}) {
		return fmt.Errorf("future validation only applies to time.Time fields")
	}

	t := val.Interface().(time.Time)
	if t.Before(time.Now()) {
		return fmt.Errorf("field value must be in the future")
	}

	return nil
}

// validatePast 验证过去时间
func (v *ConfigValidatorImpl) validatePast(value interface{}, params map[string]interface{}) error {
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Struct || val.Type() != reflect.TypeOf(time.Time{}) {
		return fmt.Errorf("past validation only applies to time.Time fields")
	}

	t := val.Interface().(time.Time)
	if t.After(time.Now()) {
		return fmt.Errorf("field value must be in the past")
	}

	return nil
}

// validateFileExists 验证文件存在
func (v *ConfigValidatorImpl) validateFileExists(value interface{}, params map[string]interface{}) error {
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.String {
		return fmt.Errorf("file_exists validation only applies to string fields")
	}

	filePath := val.String()
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	return nil
}

// validateDirExists 验证目录存在
func (v *ConfigValidatorImpl) validateDirExists(value interface{}, params map[string]interface{}) error {
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.String {
		return fmt.Errorf("dir_exists validation only applies to string fields")
	}

	dirPath := val.String()
	if dirPath == "" {
		return fmt.Errorf("directory path cannot be empty")
	}

	// 检查目录是否存在
	fileInfo, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", dirPath)
	}

	// 检查是否为目录
	if !fileInfo.IsDir() {
		return fmt.Errorf("path exists but is not a directory: %s", dirPath)
	}

	return nil
}

// AddCustomValidator 添加自定义验证器
func (v *ConfigValidatorImpl) AddCustomValidator(name string, validator CustomValidator) {
	v.customValidators[name] = validator
}

// GetValidationCount 获取验证次数
func (v *ConfigValidatorImpl) GetValidationCount() int {
	return v.validationCount
}
