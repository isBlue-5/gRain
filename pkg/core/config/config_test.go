package config

import (
	"os"
	"testing"
)

// TestConfig 用于测试的配置结构体
type TestConfig struct {
	Server struct {
		Host string `json:"host" yaml:"host" env:"SERVER_HOST" default:"0.0.0.0"`
		Port int    `json:"port" yaml:"port" env:"SERVER_PORT" default:"8080"`
	} `json:"server" yaml:"server"`

	Database struct {
		URL      string `json:"url" yaml:"url" env:"DB_URL" required:"true"`
		MaxConns int    `json:"maxConns" yaml:"maxConns" env:"DB_MAX_CONNS" default:"10"`
	} `json:"database" yaml:"database"`

	Features struct {
		EnableMetrics bool `json:"enableMetrics" yaml:"enableMetrics" env:"ENABLE_METRICS" default:"false"`
		EnableTracing bool `json:"enableTracing" yaml:"enableTracing" env:"ENABLE_TRACING" default:"false"`
	} `json:"features" yaml:"features"`
}

// TestEnvConfigSource 测试环境变量配置源
func TestEnvConfigSource(t *testing.T) {
	// 设置环境变量
	os.Setenv("TEST_SERVER_HOST", "127.0.0.1")
	os.Setenv("TEST_SERVER_PORT", "9090")
	os.Setenv("TEST_DB_URL", "mysql://localhost:3306/testdb")
	defer func() {
		os.Unsetenv("TEST_SERVER_HOST")
		os.Unsetenv("TEST_SERVER_PORT")
		os.Unsetenv("TEST_DB_URL")
	}()

	// 创建配置对象 - 注意：移除 required:"true" 用于测试
	type TestEnvConfig struct {
		Server struct {
			Host string `env:"SERVER_HOST" default:"0.0.0.0"`
			Port int    `env:"SERVER_PORT" default:"8080"`
		}
		Database struct {
			URL      string `env:"DB_URL"`
			MaxConns int    `env:"DB_MAX_CONNS" default:"10"`
		}
	}

	cfg := &TestEnvConfig{}

	// 创建环境变量配置源
	envSource := &EnvConfigSource{Prefix: "TEST"}

	// 加载配置
	err := envSource.Load(cfg)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 验证配置
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected Server.Host to be '127.0.0.1', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected Server.Port to be 9090, got %d", cfg.Server.Port)
	}
	if cfg.Database.URL != "mysql://localhost:3306/testdb" {
		t.Errorf("Expected Database.URL to be 'mysql://localhost:3306/testdb', got '%s'", cfg.Database.URL)
	}
	if cfg.Database.MaxConns != 10 {
		t.Errorf("Expected Database.MaxConns to be 10, got %d", cfg.Database.MaxConns)
	}
}

// TestYAMLConfigSource 测试YAML配置源
func TestYAMLConfigSource(t *testing.T) {
	// 创建临时YAML文件
	yamlContent := `
server:
  host: 192.168.1.1
  port: 8000
database:
  url: postgres://localhost:5432/testdb
  maxConns: 20
features:
  enableMetrics: true
  enableTracing: false
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// 创建配置对象 - 移除 required 字段用于测试
	type TestYamlConfig struct {
		Server struct {
			Host string `yaml:"host" default:"0.0.0.0"`
			Port int    `yaml:"port" default:"8080"`
		} `yaml:"server"`
		Database struct {
			URL      string `yaml:"url"`
			MaxConns int    `yaml:"maxConns" default:"10"`
		} `yaml:"database"`
		Features struct {
			EnableMetrics bool `yaml:"enableMetrics" default:"false"`
			EnableTracing bool `yaml:"enableTracing" default:"false"`
		} `yaml:"features"`
	}

	cfg := &TestYamlConfig{}

	// 创建YAML配置源
	yamlSource := &YAMLConfigSource{Path: tmpFile.Name()}

	// 加载配置
	err = yamlSource.Load(cfg)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 验证配置
	if cfg.Server.Host != "192.168.1.1" {
		t.Errorf("Expected Server.Host to be '192.168.1.1', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 8000 {
		t.Errorf("Expected Server.Port to be 8000, got %d", cfg.Server.Port)
	}
	if cfg.Database.URL != "postgres://localhost:5432/testdb" {
		t.Errorf("Expected Database.URL to be 'postgres://localhost:5432/testdb', got '%s'", cfg.Database.URL)
	}
	if cfg.Database.MaxConns != 20 {
		t.Errorf("Expected Database.MaxConns to be 20, got %d", cfg.Database.MaxConns)
	}
	if cfg.Features.EnableMetrics != true {
		t.Errorf("Expected Features.EnableMetrics to be true, got %v", cfg.Features.EnableMetrics)
	}
}

// TestJSONConfigSource 测试JSON配置源
func TestJSONConfigSource(t *testing.T) {
	// 创建临时JSON文件
	jsonContent := `{
	"server": {
		"host": "10.0.0.1",
		"port": 7000
	},
	"database": {
		"url": "mongodb://localhost:27017/testdb",
		"maxConns": 30
	},
	"features": {
		"enableMetrics": true,
		"enableTracing": true
	}
}`
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(jsonContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// 创建配置对象 - 移除 required 字段用于测试
	type TestJsonConfig struct {
		Server struct {
			Host string `json:"host" default:"0.0.0.0"`
			Port int    `json:"port" default:"8080"`
		} `json:"server"`
		Database struct {
			URL      string `json:"url"`
			MaxConns int    `json:"maxConns" default:"10"`
		} `json:"database"`
		Features struct {
			EnableMetrics bool `json:"enableMetrics" default:"false"`
			EnableTracing bool `json:"enableTracing" default:"false"`
		} `json:"features"`
	}

	cfg := &TestJsonConfig{}

	// 创建JSON配置源
	jsonSource := &JSONConfigSource{Path: tmpFile.Name()}

	// 加载配置
	err = jsonSource.Load(cfg)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 验证配置
	if cfg.Server.Host != "10.0.0.1" {
		t.Errorf("Expected Server.Host to be '10.0.0.1', got '%s'", cfg.Server.Host)
	}
	if cfg.Server.Port != 7000 {
		t.Errorf("Expected Server.Port to be 7000, got %d", cfg.Server.Port)
	}
	if cfg.Database.URL != "mongodb://localhost:27017/testdb" {
		t.Errorf("Expected Database.URL to be 'mongodb://localhost:27017/testdb', got '%s'", cfg.Database.URL)
	}
	if cfg.Database.MaxConns != 30 {
		t.Errorf("Expected Database.MaxConns to be 30, got %d", cfg.Database.MaxConns)
	}
	if cfg.Features.EnableTracing != true {
		t.Errorf("Expected Features.EnableTracing to be true, got %v", cfg.Features.EnableTracing)
	}
}

// TestConfigLoader 测试配置加载器
func TestConfigLoader(t *testing.T) {
	// 设置环境变量（优先级高于文件）
	os.Setenv("TEST_SERVER_PORT", "9999")
	defer os.Unsetenv("TEST_SERVER_PORT")

	// 创建临时YAML文件
	yamlContent := `
server:
  host: 192.168.1.1
  port: 8000
database:
  url: postgres://localhost:5432/testdb
  maxConns: 20
features:
  enableMetrics: true
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// 创建配置对象 - 移除 required 字段用于测试
	type TestLoaderConfig struct {
		Server struct {
			Host string `yaml:"host" json:"host" env:"SERVER_HOST" default:"0.0.0.0"`
			Port int    `yaml:"port" json:"port" env:"SERVER_PORT" default:"8080"`
		} `yaml:"server" json:"server"`
		Database struct {
			URL      string `yaml:"url" json:"url" env:"DB_URL"`
			MaxConns int    `yaml:"maxConns" json:"maxConns" env:"DB_MAX_CONNS" default:"10"`
		} `yaml:"database" json:"database"`
		Features struct {
			EnableMetrics bool `yaml:"enableMetrics" json:"enableMetrics" env:"ENABLE_METRICS" default:"false"`
			EnableTracing bool `yaml:"enableTracing" json:"enableTracing" env:"ENABLE_TRACING" default:"false"`
		} `yaml:"features" json:"features"`
	}

	cfg := &TestLoaderConfig{}

	// 创建配置加载器，先加载文件，再加载环境变量
	loader := NewConfigLoader(
		&YAMLConfigSource{Path: tmpFile.Name()},
		&EnvConfigSource{Prefix: "TEST"},
	)

	// 加载配置
	err = loader.Load(cfg)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 验证配置
	if cfg.Server.Host != "192.168.1.1" {
		t.Errorf("Expected Server.Host to be '192.168.1.1', got '%s'", cfg.Server.Host)
	}
	// 环境变量应该覆盖文件配置
	if cfg.Server.Port != 9999 {
		t.Errorf("Expected Server.Port to be 9999 (from env), got %d", cfg.Server.Port)
	}
	if cfg.Database.URL != "postgres://localhost:5432/testdb" {
		t.Errorf("Expected Database.URL to be 'postgres://localhost:5432/testdb', got '%s'", cfg.Database.URL)
	}
	if cfg.Features.EnableMetrics != true {
		t.Errorf("Expected Features.EnableMetrics to be true, got %v", cfg.Features.EnableMetrics)
	}
}
