// Package config 提供配置管理功能
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigSource 定义配置源接口
// 不同的配置源可以实现此接口，如文件、环境变量、远程服务等
type ConfigSource interface {
	// Load 加载配置到目标对象
	Load(config interface{}) error
	// Name 返回配置源名称
	Name() string
}

// ConfigLoader 配置加载器，支持多源配置
type ConfigLoader struct {
	sources []ConfigSource
}

// NewConfigLoader 创建新的配置加载器
func NewConfigLoader(sources ...ConfigSource) *ConfigLoader {
	return &ConfigLoader{
		sources: sources,
	}
}

// AddSource 添加配置源
func (l *ConfigLoader) AddSource(source ConfigSource) {
	l.sources = append(l.sources, source)
}

// Load 从所有配置源加载配置
// 按照添加顺序依次加载，后加载的会覆盖先加载的相同配置项
func (l *ConfigLoader) Load(config interface{}) error {
	// 验证配置对象是指针类型
	if reflect.TypeOf(config).Kind() != reflect.Ptr {
		return fmt.Errorf("config must be a pointer to struct")
	}

	// 依次从所有配置源加载
	for _, source := range l.sources {
		if err := source.Load(config); err != nil {
			return fmt.Errorf("failed to load config from %s: %w", source.Name(), err)
		}
	}

	// 验证所有必需字段
	if err := validateRequiredFields(config); err != nil {
		return err
	}

	return nil
}

// FileConfigSource 文件配置源
type FileConfigSource struct {
	Path string
	// 可选的文件格式，自动检测时可为空
	Format string
}

// Load 从文件加载配置
func (s *FileConfigSource) Load(config interface{}) error {
	// 检查文件是否存在
	if _, err := os.Stat(s.Path); os.IsNotExist(err) {
		return fmt.Errorf("配置文件不存在: %s", s.Path)
	}

	// 读取文件内容
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 确定文件格式
	format := s.Format
	if format == "" {
		format = detectFormat(s.Path)
	}

	// 根据格式解析文件
	switch format {
	case "yaml", "yml":
		return parseYAML(data, config)
	case "json":
		return parseJSON(data, config)
	default:
		// 未知格式，尝试YAML然后JSON
		if err := parseYAML(data, config); err == nil {
			return nil
		}
		return parseJSON(data, config)
	}
}

// Name 返回配置源名称
func (s *FileConfigSource) Name() string {
	return fmt.Sprintf("file(%s)", s.Path)
}

// detectFormat 根据文件扩展名检测文件格式
func detectFormat(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	default:
		return ""
	}
}

// parseYAML 解析YAML数据
func parseYAML(data []byte, config interface{}) error {
	err := yaml.Unmarshal(data, config)
	if err != nil {
		return fmt.Errorf("解析YAML失败: %w", err)
	}
	return nil
}

// parseJSON 解析JSON数据
func parseJSON(data []byte, config interface{}) error {
	err := json.Unmarshal(data, config)
	if err != nil {
		return fmt.Errorf("解析JSON失败: %w", err)
	}
	return nil
}

// YAMLConfigSource YAML特定的配置源
// 为语义明确，提供特定文件格式的配置源
type YAMLConfigSource struct {
	Path string
}

// Load 从YAML文件加载配置
func (s *YAMLConfigSource) Load(config interface{}) error {
	fileSource := &FileConfigSource{
		Path:   s.Path,
		Format: "yaml",
	}
	return fileSource.Load(config)
}

// Name 返回配置源名称
func (s *YAMLConfigSource) Name() string {
	return fmt.Sprintf("yaml(%s)", s.Path)
}

// JSONConfigSource JSON特定的配置源
type JSONConfigSource struct {
	Path string
}

// Load 从JSON文件加载配置
func (s *JSONConfigSource) Load(config interface{}) error {
	fileSource := &FileConfigSource{
		Path:   s.Path,
		Format: "json",
	}
	return fileSource.Load(config)
}

// Name 返回配置源名称
func (s *JSONConfigSource) Name() string {
	return fmt.Sprintf("json(%s)", s.Path)
}

// EnvConfigSource 环境变量配置源
type EnvConfigSource struct {
	Prefix string
}

// Load 从环境变量加载配置
func (s *EnvConfigSource) Load(config interface{}) error {
	return processStruct(reflect.ValueOf(config), s.Prefix)
}

// Name 返回配置源名称
func (s *EnvConfigSource) Name() string {
	if s.Prefix != "" {
		return fmt.Sprintf("env(prefix=%s)", s.Prefix)
	}
	return "env"
}

// processStruct 处理结构体字段
func processStruct(v reflect.Value, prefix string) error {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// 跳过未导出字段
		if !fieldValue.CanSet() {
			continue
		}

		// 处理嵌套结构体
		if fieldValue.Kind() == reflect.Struct {
			// 对于嵌套结构体，不改变前缀，让子字段自己处理
			if err := processStruct(fieldValue, prefix); err != nil {
				return err
			}
			continue
		}

		// 处理指针类型的嵌套结构体
		if fieldValue.Kind() == reflect.Ptr && fieldValue.Type().Elem().Kind() == reflect.Struct {
			if fieldValue.IsNil() {
				fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
			}
			// 对于指针类型的嵌套结构体，不改变前缀，让子字段自己处理
			if err := processStruct(fieldValue.Elem(), prefix); err != nil {
				return err
			}
			continue
		}

		// 处理基本类型字段
		tagValue := field.Tag.Get("env")
		if tagValue == "-" {
			continue
		}

		// 构建环境变量键
		var envKey string
		if tagValue != "" {
			// 如果标签指定了环境变量名，直接使用前缀+标签值
			if prefix != "" {
				envKey = prefix + "_" + tagValue
			} else {
				envKey = tagValue
			}
		} else {
			// 如果没有标签，使用前缀+字段名
			envKey = prefix
			if envKey != "" {
				envKey = envKey + "_"
			}
			envKey = envKey + strings.ToUpper(field.Name)
		}

		// 获取环境变量值
		envValue, exists := os.LookupEnv(envKey)
		if !exists {
			// 如果有默认值标签，且字段为零值，才使用默认值
			defaultValue := field.Tag.Get("default")
			if defaultValue != "" && isZeroValue(fieldValue) {
				if err := setFieldFromString(fieldValue, defaultValue); err != nil {
					return fmt.Errorf("failed to set default value for field %s: %w", field.Name, err)
				}
			}
			continue
		}

		// 设置字段值
		if err := setFieldFromString(fieldValue, envValue); err != nil {
			return fmt.Errorf("failed to set field %s from env %s: %w", field.Name, envKey, err)
		}
	}

	return nil
}

// setFieldFromString 根据字符串设置字段值
func setFieldFromString(v reflect.Value, s string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if s == "" {
			return nil
		}
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if s == "" {
			return nil
		}
		i, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		v.SetUint(i)
	case reflect.Bool:
		if s == "" {
			return nil
		}
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.SetBool(b)
	case reflect.Float32, reflect.Float64:
		if s == "" {
			return nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		v.SetFloat(f)
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return setFieldFromString(v.Elem(), s)
	case reflect.Slice:
		if s == "" {
			return nil
		}
		items := strings.Split(s, ",")
		slice := reflect.MakeSlice(v.Type(), len(items), len(items))
		for i, item := range items {
			if err := setFieldFromString(slice.Index(i), strings.TrimSpace(item)); err != nil {
				return err
			}
		}
		v.Set(slice)
	default:
		return fmt.Errorf("unsupported field type: %v", v.Kind())
	}
	return nil
}

// validateRequiredFields 验证所有必需字段
func validateRequiredFields(config interface{}) error {
	v := reflect.ValueOf(config)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	return validateRequiredFieldsValue(v, "")
}

// validateRequiredFieldsValue 递归验证必需字段
func validateRequiredFieldsValue(v reflect.Value, path string) error {
	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// 跳过未导出字段
		if !fieldValue.CanSet() {
			continue
		}

		currentPath := path
		if currentPath != "" {
			currentPath = currentPath + "."
		}
		currentPath = currentPath + field.Name

		// 检查是否为必需字段
		requiredTag := field.Tag.Get("required")
		if requiredTag == "true" {
			// 检查值是否为零值
			if isZeroValue(fieldValue) {
				return fmt.Errorf("required field %s is not set", currentPath)
			}
		}

		// 递归处理结构体字段
		if fieldValue.Kind() == reflect.Struct {
			if err := validateRequiredFieldsValue(fieldValue, currentPath); err != nil {
				return err
			}
		} else if fieldValue.Kind() == reflect.Ptr && !fieldValue.IsNil() && fieldValue.Elem().Kind() == reflect.Struct {
			if err := validateRequiredFieldsValue(fieldValue.Elem(), currentPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// isZeroValue 检查字段是否为零值
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr:
		return v.IsNil()
	case reflect.Slice:
		return v.Len() == 0
	default:
		return false
	}
}

// ConfigValidator 配置验证器接口
type ConfigValidator interface {
	// Validate 验证配置是否合法
	Validate(config interface{}) error
}

// DefaultValidator 默认配置验证器
type DefaultValidator struct{}

// Validate 实现ConfigValidator接口
func (v *DefaultValidator) Validate(config interface{}) error {
	// 使用反射检查必需字段
	val := reflect.ValueOf(config)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// 检查required标签
		if required, ok := field.Tag.Lookup("config"); ok && strings.Contains(required, "required") {
			if isZeroValue(fieldVal) {
				return fmt.Errorf("required field '%s' is not set", field.Name)
			}
		}
	}

	return nil
}

// ConfigManager 配置管理器，支持验证和热更新
type ConfigManager struct {
	loader    *ConfigLoader
	validator ConfigValidator
	config    interface{}
}

// NewConfigManager 创建配置管理器
func NewConfigManager(loader *ConfigLoader, validator ConfigValidator) *ConfigManager {
	if validator == nil {
		validator = &DefaultValidator{}
	}
	return &ConfigManager{
		loader:    loader,
		validator: validator,
	}
}

// LoadAndValidate 加载并验证配置
func (m *ConfigManager) LoadAndValidate(config interface{}) error {
	// 加载配置
	if err := m.loader.Load(config); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 验证配置
	if err := m.validator.Validate(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	m.config = config
	return nil
}

// GetConfig 获取当前配置
func (m *ConfigManager) GetConfig() interface{} {
	return m.config
}

// DefaultLoader 返回默认的配置加载器，包含文件和环境变量源
func DefaultLoader(configPath string, envPrefix string) *ConfigLoader {
	return NewConfigLoader(
		&FileConfigSource{Path: configPath},
		&EnvConfigSource{Prefix: envPrefix},
	)
}
