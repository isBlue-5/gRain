package options

import (
	"errors"
	"testing"
)

// TestConfig 用于测试的配置结构体
type TestConfig struct {
	Host    string
	Port    int
	Timeout int
	Debug   bool
	Tags    []string
}

// TestOption 用于测试的选项函数
type TestOption = Option[TestConfig]

// WithHost 设置Host选项
func WithHost(host string) TestOption {
	return func(c *TestConfig) {
		c.Host = host
	}
}

// WithPort 设置Port选项
func WithPort(port int) TestOption {
	return func(c *TestConfig) {
		c.Port = port
	}
}

// WithTimeout 设置Timeout选项
func WithTimeout(timeout int) TestOption {
	return func(c *TestConfig) {
		c.Timeout = timeout
	}
}

// WithDebug 设置Debug选项
func WithDebug(debug bool) TestOption {
	return func(c *TestConfig) {
		c.Debug = debug
	}
}

// WithTags 设置Tags选项
func WithTags(tags ...string) TestOption {
	return func(c *TestConfig) {
		c.Tags = tags
	}
}

// TestOptionWithError 用于测试的带错误选项函数
type TestOptionWithError = OptionWithError[TestConfig]

// WithValidatedPort 带验证的端口设置
func WithValidatedPort(port int) TestOptionWithError {
	return func(c *TestConfig) error {
		if port <= 0 || port > 65535 {
			return errors.New("port must be between 1 and 65535")
		}
		c.Port = port
		return nil
	}
}

// TestApply 测试Apply函数
func TestApply(t *testing.T) {
	// 创建默认配置
	cfg := &TestConfig{
		Host:    "localhost",
		Port:    8080,
		Timeout: 30,
		Debug:   false,
	}

	// 应用选项
	Apply(cfg, WithHost("127.0.0.1"), WithPort(9090), WithDebug(true))

	// 验证配置
	if cfg.Host != "127.0.0.1" {
		t.Errorf("Expected Host to be '127.0.0.1', got '%s'", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Errorf("Expected Port to be 9090, got %d", cfg.Port)
	}
	if !cfg.Debug {
		t.Error("Expected Debug to be true, got false")
	}
	if cfg.Timeout != 30 {
		t.Errorf("Expected Timeout to be unchanged (30), got %d", cfg.Timeout)
	}
}

// TestApplyWithError 测试ApplyWithError函数
func TestApplyWithError(t *testing.T) {
	// 测试有效配置
	cfg1 := &TestConfig{}
	err := ApplyWithError(cfg1, WithValidatedPort(8080))
	if err != nil {
		t.Errorf("Expected no error for valid port, got %v", err)
	}
	if cfg1.Port != 8080 {
		t.Errorf("Expected Port to be 8080, got %d", cfg1.Port)
	}

	// 测试无效配置
	cfg2 := &TestConfig{}
	err = ApplyWithError(cfg2, WithValidatedPort(70000))
	if err == nil {
		t.Error("Expected error for invalid port, got nil")
	}
}

// TestConditional 测试Conditional函数
func TestConditional(t *testing.T) {
	// 创建配置
	cfg := &TestConfig{Port: 8080}

	// 条件为true，应该应用选项
	Apply(cfg, Conditional(true, WithPort(9090)))
	if cfg.Port != 9090 {
		t.Errorf("Expected Port to be 9090 when condition is true, got %d", cfg.Port)
	}

	// 条件为false，不应该应用选项
	Apply(cfg, Conditional(false, WithPort(7070)))
	if cfg.Port != 9090 {
		t.Errorf("Expected Port to remain 9090 when condition is false, got %d", cfg.Port)
	}
}

// TestGroup 测试Group函数
func TestGroup(t *testing.T) {
	// 创建配置
	cfg := &TestConfig{}

	// 创建选项组
	developmentGroup := Group[TestConfig](
		WithHost("localhost"),
		WithPort(8080),
		WithDebug(true),
	)

	// 应用选项组
	Apply(cfg, developmentGroup)

	// 验证配置
	if cfg.Host != "localhost" {
		t.Errorf("Expected Host to be 'localhost', got '%s'", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("Expected Port to be 8080, got %d", cfg.Port)
	}
	if !cfg.Debug {
		t.Error("Expected Debug to be true, got false")
	}

	// 测试组合使用
	productionGroup := Group[TestConfig](
		WithHost("0.0.0.0"),
		WithPort(80),
		WithDebug(false),
	)

	// 先应用生产配置，然后根据条件覆盖部分配置
	Apply(cfg,
		productionGroup,
		Conditional(true, WithTags("prod", "release")),
	)

	// 验证配置
	if cfg.Host != "0.0.0.0" {
		t.Errorf("Expected Host to be '0.0.0.0', got '%s'", cfg.Host)
	}
	if cfg.Port != 80 {
		t.Errorf("Expected Port to be 80, got %d", cfg.Port)
	}
	if cfg.Debug {
		t.Error("Expected Debug to be false, got true")
	}
	if len(cfg.Tags) != 2 || cfg.Tags[0] != "prod" || cfg.Tags[1] != "release" {
		t.Errorf("Expected Tags to be ['prod', 'release'], got %v", cfg.Tags)
	}
}
