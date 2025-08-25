// Package validate 提供请求验证功能
package validate

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	// ErrValidationFailed 验证失败错误
	ErrValidationFailed = errors.New("validation failed")

	// ErrInvalidTarget 无效验证目标错误
	ErrInvalidTarget = errors.New("invalid validation target")
)

// ValidateError 验证错误
type ValidateError struct {
	Field   string
	Message string
	Tag     string
	Value   interface{}
}

// Error 错误信息
func (e *ValidateError) Error() string {
	return fmt.Sprintf("validation failed on field '%s': %s", e.Field, e.Message)
}

// FieldError 字段验证错误
type FieldError struct {
	Field   string
	Message string
	Tag     string
	Value   interface{}
}

// ValidationErrors 验证错误集合
type ValidationErrors []*FieldError

// Error 错误信息
func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("validation failed: ")

	for i, err := range v {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(err.Field)
		sb.WriteString(": ")
		sb.WriteString(err.Message)
	}

	return sb.String()
}

// Add 添加错误
func (v *ValidationErrors) Add(field, message, tag string, value interface{}) {
	*v = append(*v, &FieldError{
		Field:   field,
		Message: message,
		Tag:     tag,
		Value:   value,
	})
}

// Validate 验证器接口
type Validator interface {
	Validate(interface{}) ValidationErrors
}

// DefaultValidator 默认验证器
type DefaultValidator struct{}

// Validate 验证对象
func (v *DefaultValidator) Validate(obj interface{}) ValidationErrors {
	var errors ValidationErrors

	// 检查对象类型
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	// 只能验证结构体
	if value.Kind() != reflect.Struct {
		errors.Add("", "validation target must be a struct", "invalid", obj)
		return errors
	}

	// 获取类型信息
	t := value.Type()

	// 遍历字段
	for i := 0; i < value.NumField(); i++ {
		field := t.Field(i)
		fieldValue := value.Field(i)

		// 跳过非导出字段
		if field.PkgPath != "" {
			continue
		}

		// 获取validate标签
		validateTag := field.Tag.Get("validate")
		if validateTag == "" {
			continue
		}

		// 获取字段名（优先使用json标签，其次是字段名）
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				fieldName = parts[0]
			}
		}

		// 验证字段
		if fieldErrs := validateField(fieldName, fieldValue, validateTag); len(fieldErrs) > 0 {
			errors = append(errors, fieldErrs...)
		}
	}

	return errors
}

// validateField 验证字段
func validateField(fieldName string, fieldValue reflect.Value, validateTag string) ValidationErrors {
	var errors ValidationErrors

	// 解析验证标签
	validations := parseValidateTag(validateTag)

	// 检查每个验证规则
	for _, validation := range validations {
		switch validation.tag {
		case "required":
			if isZeroValue(fieldValue) {
				errors.Add(fieldName, "field is required", "required", fieldValue.Interface())
			}

		case "email":
			if fieldValue.Kind() == reflect.String && fieldValue.String() != "" {
				if !isValidEmail(fieldValue.String()) {
					errors.Add(fieldName, "invalid email format", "email", fieldValue.String())
				}
			}

		case "min":
			param, err := strconv.Atoi(validation.param)
			if err != nil {
				continue
			}

			if fieldValue.Kind() == reflect.String {
				if utf8.RuneCountInString(fieldValue.String()) < param {
					errors.Add(fieldName, fmt.Sprintf("minimum length is %d", param), "min", fieldValue.String())
				}
			} else if isNumberKind(fieldValue.Kind()) {
				if fieldValue.Convert(reflect.TypeOf(float64(0))).Float() < float64(param) {
					errors.Add(fieldName, fmt.Sprintf("minimum value is %d", param), "min", fieldValue.Interface())
				}
			}

		case "max":
			param, err := strconv.Atoi(validation.param)
			if err != nil {
				continue
			}

			if fieldValue.Kind() == reflect.String {
				if utf8.RuneCountInString(fieldValue.String()) > param {
					errors.Add(fieldName, fmt.Sprintf("maximum length is %d", param), "max", fieldValue.String())
				}
			} else if isNumberKind(fieldValue.Kind()) {
				if fieldValue.Convert(reflect.TypeOf(float64(0))).Float() > float64(param) {
					errors.Add(fieldName, fmt.Sprintf("maximum value is %d", param), "max", fieldValue.Interface())
				}
			}

		case "len":
			param, err := strconv.Atoi(validation.param)
			if err != nil {
				continue
			}

			if fieldValue.Kind() == reflect.String {
				if utf8.RuneCountInString(fieldValue.String()) != param {
					errors.Add(fieldName, fmt.Sprintf("length must be %d", param), "len", fieldValue.String())
				}
			} else if fieldValue.Kind() == reflect.Slice || fieldValue.Kind() == reflect.Array || fieldValue.Kind() == reflect.Map {
				if fieldValue.Len() != param {
					errors.Add(fieldName, fmt.Sprintf("length must be %d", param), "len", fieldValue.Interface())
				}
			}

		case "oneof":
			values := strings.Split(validation.param, " ")
			valid := false

			if fieldValue.Kind() == reflect.String {
				str := fieldValue.String()
				for _, val := range values {
					if str == val {
						valid = true
						break
					}
				}
			} else if isNumberKind(fieldValue.Kind()) {
				num := fieldValue.Convert(reflect.TypeOf(float64(0))).Float()
				for _, val := range values {
					if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
						if num == floatVal {
							valid = true
							break
						}
					}
				}
			}

			if !valid {
				errors.Add(fieldName, fmt.Sprintf("must be one of: %s", validation.param), "oneof", fieldValue.Interface())
			}

		case "regex":
			if fieldValue.Kind() == reflect.String {
				re, err := regexp.Compile(validation.param)
				if err != nil {
					continue
				}

				if !re.MatchString(fieldValue.String()) {
					errors.Add(fieldName, "does not match pattern", "regex", fieldValue.String())
				}
			}
		}
	}

	return errors
}

// validation 验证规则
type validation struct {
	tag   string
	param string
}

// parseValidateTag 解析验证标签
func parseValidateTag(tag string) []validation {
	var validations []validation

	// 分割验证规则
	for _, v := range strings.Split(tag, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}

		// 分离标签和参数
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 1 {
			// 没有参数的标签
			validations = append(validations, validation{
				tag:   parts[0],
				param: "",
			})
		} else {
			// 有参数的标签
			validations = append(validations, validation{
				tag:   parts[0],
				param: parts[1],
			})
		}
	}

	return validations
}

// isZeroValue 检查是否为零值
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan:
		return v.IsNil()
	case reflect.Struct:
		return false // 结构体需要深度检查，这里简化处理
	default:
		return false
	}
}

// isNumberKind 检查是否为数字类型
func isNumberKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// isValidEmail 检查是否为有效的邮箱格式
func isValidEmail(email string) bool {
	// 简单的邮箱格式验证
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// New 创建一个新的验证器
func New() Validator {
	return &DefaultValidator{}
}

// ValidateStruct 验证结构体
func ValidateStruct(obj interface{}) error {
	validator := New()
	if errs := validator.Validate(obj); len(errs) > 0 {
		return errs
	}
	return nil
}
