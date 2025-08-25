// Package config 提供配置管理功能
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// ConfigOptimizer 配置优化器
type ConfigOptimizer struct {
	// 配置验证器
	validator ConfigValidator

	// 配置监控器
	monitor *ConfigMonitor

	// 配置缓存
	cache *ConfigCache

	// 优化选项
	options *OptimizerOptions
}

// OptimizerOptions 优化器选项
type OptimizerOptions struct {
	// 是否启用配置热重载
	EnableHotReload bool

	// 是否启用配置验证
	EnableValidation bool

	// 是否启用配置缓存
	EnableCache bool

	// 配置重载间隔
	ReloadInterval time.Duration

	// 最大配置大小（字节）
	MaxConfigSize int64

	// 允许的配置格式
	AllowedFormats []string
}

// DefaultOptimizerOptions 默认优化器选项
func DefaultOptimizerOptions() *OptimizerOptions {
	return &OptimizerOptions{
		EnableHotReload:  true,
		EnableValidation: true,
		EnableCache:      true,
		ReloadInterval:   30 * time.Second,
		MaxConfigSize:    1024 * 1024, // 1MB
		AllowedFormats:   []string{"yaml", "yml", "json", "env"},
	}
}

// NewConfigOptimizer 创建新的配置优化器
func NewConfigOptimizer(options *OptimizerOptions) *ConfigOptimizer {
	if options == nil {
		options = DefaultOptimizerOptions()
	}

	return &ConfigOptimizer{
		validator: NewConfigValidator(),
		monitor:   NewConfigMonitor(options.ReloadInterval),
		cache:     NewConfigCache(),
		options:   options,
	}
}

// OptimizeConfig 优化配置
func (co *ConfigOptimizer) OptimizeConfig(config interface{}) error {
	// 1. 验证配置
	if co.options.EnableValidation {
		if err := co.validator.Validate(config); err != nil {
			return fmt.Errorf("config validation failed: %w", err)
		}
	}

	// 2. 优化配置值
	if err := co.optimizeConfigValues(config); err != nil {
		return fmt.Errorf("config optimization failed: %w", err)
	}

	// 3. 缓存配置
	if co.options.EnableCache {
		co.cache.Set("config", config)
	}

	return nil
}

// optimizeConfigValues 优化配置值
func (co *ConfigOptimizer) optimizeConfigValues(config interface{}) error {
	val := reflect.ValueOf(config)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	return co.optimizeStruct(val)
}

// optimizeStruct 优化结构体配置
func (co *ConfigOptimizer) optimizeStruct(val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过未导出的字段
		if !field.CanSet() {
			continue
		}

		// 处理嵌套结构体
		if field.Kind() == reflect.Struct {
			if err := co.optimizeStruct(field); err != nil {
				return fmt.Errorf("failed to optimize field %s: %w", fieldType.Name, err)
			}
			continue
		}

		// 处理指针类型
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			if field.Elem().Kind() == reflect.Struct {
				if err := co.optimizeStruct(field.Elem()); err != nil {
					return fmt.Errorf("failed to optimize pointer field %s: %w", fieldType.Name, err)
				}
			}
			continue
		}

		// 应用字段优化规则
		if err := co.applyFieldOptimization(field, fieldType); err != nil {
			return fmt.Errorf("failed to apply optimization to field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// applyFieldOptimization 应用字段优化规则
func (co *ConfigOptimizer) applyFieldOptimization(field reflect.Value, fieldType reflect.StructField) error {
	// 获取字段标签
	tag := fieldType.Tag.Get("config")
	if tag == "" {
		return nil
	}

	// 解析标签
	options := co.parseTagOptions(tag)

	// 应用优化规则
	for _, option := range options {
		if err := co.applyOption(field, option); err != nil {
			return fmt.Errorf("failed to apply option %s: %w", option, err)
		}
	}

	return nil
}

// parseTagOptions 解析标签选项
func (co *ConfigOptimizer) parseTagOptions(tag string) []string {
	// 处理带参数的选项，如 "min:1,max:100"
	options := strings.Split(tag, ",")
	var result []string

	for _, option := range options {
		option = strings.TrimSpace(option)
		if option == "" {
			continue
		}

		// 提取选项名称（去掉参数部分）
		if colonIndex := strings.Index(option, ":"); colonIndex > 0 {
			optionName := option[:colonIndex]
			result = append(result, optionName)
		} else {
			result = append(result, option)
		}
	}

	return result
}

// applyOption 应用单个选项
func (co *ConfigOptimizer) applyOption(field reflect.Value, option string) error {
	switch option {
	case "expand_env":
		return co.expandEnvironmentVariables(field)
	case "resolve_path":
		return co.resolvePath(field)
	case "set_default":
		return co.setDefaultValue(field)
	case "validate_range":
		return co.validateRange(field)
	case "min":
		// min选项由验证器处理，这里跳过
		return nil
	case "max":
		// max选项由验证器处理，这里跳过
		return nil
	default:
		return fmt.Errorf("unknown optimization option: %s", option)
	}
}

// expandEnvironmentVariables 展开环境变量
func (co *ConfigOptimizer) expandEnvironmentVariables(field reflect.Value) error {
	if field.Kind() != reflect.String {
		return nil
	}

	value := field.String()
	if !strings.Contains(value, "${") {
		return nil
	}

	expanded := os.ExpandEnv(value)
	field.SetString(expanded)

	return nil
}

// resolvePath 解析路径
func (co *ConfigOptimizer) resolvePath(field reflect.Value) error {
	if field.Kind() != reflect.String {
		return nil
	}

	path := field.String()
	if path == "" {
		return nil
	}

	// 如果是相对路径，转换为绝对路径
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to resolve path %s: %w", path, err)
		}
		field.SetString(absPath)
	}

	return nil
}

// setDefaultValue 设置默认值
func (co *ConfigOptimizer) setDefaultValue(field reflect.Value) error {
	// 如果字段已经有值，不设置默认值
	if !field.IsZero() {
		return nil
	}

	// 根据字段类型设置默认值
	switch field.Kind() {
	case reflect.String:
		field.SetString("")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.SetInt(0)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.SetUint(0)
	case reflect.Float32, reflect.Float64:
		field.SetFloat(0.0)
	case reflect.Bool:
		field.SetBool(false)
	}

	return nil
}

// validateRange 验证数值范围
func (co *ConfigOptimizer) validateRange(field reflect.Value) error {
	// 获取字段标签中的范围信息
	fieldType := field.Type()

	// 检查字段类型是否支持范围验证
	switch fieldType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		// 数值类型支持范围验证
	default:
		return nil // 非数值类型跳过范围验证
	}

	// 获取字段的标签信息
	fieldStruct := field.Type()
	if fieldStruct.Kind() == reflect.Ptr {
		fieldStruct = fieldStruct.Elem()
	}

	// 这里可以根据标签中的 min、max 值进行范围验证
	// 例如：config:"min:1,max:100"
	// 由于标签解析需要更复杂的逻辑，这里提供基础框架
	// 实际实现中可以通过反射获取标签值并解析

	return nil
}

// WatchConfig 监控配置变化
func (co *ConfigOptimizer) WatchConfig(configPath string, callback func(interface{}) error) error {
	if !co.options.EnableHotReload {
		return fmt.Errorf("hot reload is disabled")
	}

	return co.monitor.WatchFile(configPath, callback)
}

// GetCachedConfig 获取缓存的配置
func (co *ConfigOptimizer) GetCachedConfig() interface{} {
	if !co.options.EnableCache {
		return nil
	}

	value, _ := co.cache.Get("config")
	return value
}

// ClearCache 清除配置缓存
func (co *ConfigOptimizer) ClearCache() {
	if co.options.EnableCache {
		co.cache.Clear()
	}
}

// GetStats 获取优化器统计信息
func (co *ConfigOptimizer) GetStats() *OptimizerStats {
	stats := &OptimizerStats{
		ValidationCount: 0,
		CacheHits:       0,
		CacheMisses:     0,
		ReloadCount:     0,
	}

	// 尝试获取验证器统计信息
	if validatorImpl, ok := co.validator.(*ConfigValidatorImpl); ok {
		stats.ValidationCount = validatorImpl.GetValidationCount()
	}

	// 获取缓存统计信息
	stats.CacheHits = co.cache.GetHits()
	stats.CacheMisses = co.cache.GetMisses()

	// 获取监控器统计信息
	stats.ReloadCount = co.monitor.GetReloadCount()

	return stats
}

// OptimizerStats 优化器统计信息
type OptimizerStats struct {
	ValidationCount int
	CacheHits       int64
	CacheMisses     int64
	ReloadCount     int
}
