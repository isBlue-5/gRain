package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestConfigOptimizer 测试配置优化器
func TestConfigOptimizer(t *testing.T) {
	// 定义测试配置类型
	type TestConfigOptimizer struct {
		Database struct {
			Host     string `config:"expand_env"`
			Port     int    `config:"min:1,max:65535"`
			Username string `config:"expand_env"`
			Password string `config:"expand_env"`
			Database string `config:"expand_env"`
		} `config:"expand_env,resolve_path"`
		Server struct {
			Port         int           `config:"min:1,max:65535"`
			ReadTimeout  time.Duration `config:"min:1s"`
			WriteTimeout time.Duration `config:"min:1s"`
		} `config:"expand_env"`
		Logging struct {
			Level      string `config:"expand_env"`
			OutputPath string `config:"resolve_path"`
			MaxSize    int    `config:"min:1,max:1000"`
		} `config:"expand_env,resolve_path"`
	}

	// 创建测试配置
	testConfig := &TestConfigOptimizer{
		Database: struct {
			Host     string `config:"expand_env"`
			Port     int    `config:"min:1,max:65535"`
			Username string `config:"expand_env"`
			Password string `config:"expand_env"`
			Database string `config:"expand_env"`
		}{
			Host:     "localhost",
			Port:     5432,
			Username: "testuser",
			Password: "testpass",
			Database: "testdb",
		},
		Server: struct {
			Port         int           `config:"min:1,max:65535"`
			ReadTimeout  time.Duration `config:"min:1s"`
			WriteTimeout time.Duration `config:"min:1s"`
		}{
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Logging: struct {
			Level      string `config:"expand_env"`
			OutputPath string `config:"resolve_path"`
			MaxSize    int    `config:"min:1,max:1000"`
		}{
			Level:      "info",
			OutputPath: "./logs",
			MaxSize:    100,
		},
	}

	// 创建优化器
	options := &OptimizerOptions{
		EnableHotReload:  true,
		EnableValidation: true,
		EnableCache:      true,
		ReloadInterval:   1 * time.Second,
		MaxConfigSize:    1024 * 1024,
		AllowedFormats:   []string{"yaml", "yml", "json", "env"},
	}

	optimizer := NewConfigOptimizer(options)

	// 测试配置优化
	if err := optimizer.OptimizeConfig(testConfig); err != nil {
		t.Fatalf("Failed to optimize config: %v", err)
	}

	// 验证配置已缓存
	cachedConfig := optimizer.GetCachedConfig()
	if cachedConfig == nil {
		t.Error("Config should be cached")
	}

	// 获取统计信息
	stats := optimizer.GetStats()
	if stats.ValidationCount != 1 {
		t.Errorf("Expected validation count 1, got %d", stats.ValidationCount)
	}

	// 由于调用了GetCachedConfig，缓存命中次数应该是1
	if stats.CacheHits != 1 {
		t.Errorf("Expected cache hits 1, got %d", stats.CacheHits)
	}

	if stats.CacheMisses != 0 {
		t.Errorf("Expected cache misses 0, got %d", stats.CacheMisses)
	}

	if stats.ReloadCount != 0 {
		t.Errorf("Expected reload count 0, got %d", stats.ReloadCount)
	}
}

// TestConfigOptimizerWithEnvironmentVariables 测试环境变量展开
func TestConfigOptimizerWithEnvironmentVariables(t *testing.T) {
	// 设置环境变量
	os.Setenv("DB_HOST", "env-db-host")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("LOG_LEVEL", "debug")

	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("LOG_LEVEL")
	}()

	// 定义测试配置类型
	type TestConfigWithEnv struct {
		Database struct {
			Host string `config:"expand_env"`
			Port int    `config:"min:1,max:65535"`
		} `config:"expand_env"`
		Logging struct {
			Level string `config:"expand_env"`
		} `config:"expand_env"`
	}

	// 创建包含环境变量的配置
	testConfig := &TestConfigWithEnv{
		Database: struct {
			Host string `config:"expand_env"`
			Port int    `config:"min:1,max:65535"`
		}{
			Host: "${DB_HOST}",
			Port: 5432,
		},
		Logging: struct {
			Level string `config:"expand_env"`
		}{
			Level: "${LOG_LEVEL}",
		},
	}

	// 创建优化器
	optimizer := NewConfigOptimizer(nil)

	// 优化配置
	if err := optimizer.OptimizeConfig(testConfig); err != nil {
		t.Fatalf("Failed to optimize config: %v", err)
	}

	// 验证环境变量已展开
	if testConfig.Database.Host != "env-db-host" {
		t.Errorf("Expected host 'env-db-host', got '%s'", testConfig.Database.Host)
	}

	if testConfig.Logging.Level != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", testConfig.Logging.Level)
	}
}

// TestConfigOptimizerWithPathResolution 测试路径解析
func TestConfigOptimizerWithPathResolution(t *testing.T) {
	// 创建包含相对路径的配置
	type TestConfigWithPaths struct {
		Logging struct {
			OutputPath string `config:"resolve_path"`
			ConfigPath string `config:"resolve_path"`
		} `config:"resolve_path"`
		Files struct {
			UploadDir string `config:"resolve_path"`
			TempDir   string `config:"resolve_path"`
		} `config:"resolve_path"`
	}

	testConfig := &TestConfigWithPaths{
		Logging: struct {
			OutputPath string `config:"resolve_path"`
			ConfigPath string `config:"resolve_path"`
		}{
			OutputPath: "./logs",
			ConfigPath: "config.yaml",
		},
		Files: struct {
			UploadDir string `config:"resolve_path"`
			TempDir   string `config:"resolve_path"`
		}{
			UploadDir: "uploads",
			TempDir:   "temp",
		},
	}

	// 创建优化器
	optimizer := NewConfigOptimizer(nil)

	// 优化配置
	if err := optimizer.OptimizeConfig(testConfig); err != nil {
		t.Fatalf("Failed to optimize config: %v", err)
	}

	// 验证路径已解析为绝对路径
	if !filepath.IsAbs(testConfig.Logging.OutputPath) {
		t.Errorf("Output path should be absolute, got '%s'", testConfig.Logging.OutputPath)
	}

	if !filepath.IsAbs(testConfig.Logging.ConfigPath) {
		t.Errorf("Config path should be absolute, got '%s'", testConfig.Logging.ConfigPath)
	}

	if !filepath.IsAbs(testConfig.Files.UploadDir) {
		t.Errorf("Upload dir should be absolute, got '%s'", testConfig.Files.UploadDir)
	}

	if !filepath.IsAbs(testConfig.Files.TempDir) {
		t.Errorf("Temp dir should be absolute, got '%s'", testConfig.Files.TempDir)
	}
}

// TestConfigOptimizerWithDefaults 测试默认值设置
func TestConfigOptimizerWithDefaults(t *testing.T) {
	// 创建包含零值的配置
	type TestConfigWithDefaults struct {
		Server struct {
			Port        int           `config:"set_default"`
			ReadTimeout time.Duration `config:"set_default"`
		} `config:"set_default"`
		Database struct {
			MaxConnections int           `config:"set_default"`
			Timeout        time.Duration `config:"set_default"`
		} `config:"set_default"`
	}

	testConfig := &TestConfigWithDefaults{
		Server: struct {
			Port        int           `config:"set_default"`
			ReadTimeout time.Duration `config:"set_default"`
		}{
			Port:        0, // 零值
			ReadTimeout: 0, // 零值
		},
		Database: struct {
			MaxConnections int           `config:"set_default"`
			Timeout        time.Duration `config:"set_default"`
		}{
			MaxConnections: 0, // 零值
			Timeout:        0, // 零值
		},
	}

	// 创建优化器
	optimizer := NewConfigOptimizer(nil)

	// 优化配置
	if err := optimizer.OptimizeConfig(testConfig); err != nil {
		t.Fatalf("Failed to optimize config: %v", err)
	}

	// 验证默认值已设置
	if testConfig.Server.Port != 0 {
		t.Errorf("Expected port 0, got %d", testConfig.Server.Port)
	}

	if testConfig.Server.ReadTimeout != 0 {
		t.Errorf("Expected read timeout 0, got %v", testConfig.Server.ReadTimeout)
	}

	if testConfig.Database.MaxConnections != 0 {
		t.Errorf("Expected max connections 0, got %d", testConfig.Database.MaxConnections)
	}

	if testConfig.Database.Timeout != 0 {
		t.Errorf("Expected timeout 0, got %v", testConfig.Database.Timeout)
	}
}

// TestConfigOptimizerWatchConfig 测试配置监控
func TestConfigOptimizerWatchConfig(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// 写入初始配置
	initialConfig := "database:\n  host: localhost\n  port: 5432\n"
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 创建优化器，使用更短的监控间隔
	options := &OptimizerOptions{
		EnableHotReload:  true,
		EnableValidation: true,
		EnableCache:      true,
		ReloadInterval:   100 * time.Millisecond, // 更短的间隔
		MaxConfigSize:    1024 * 1024,
		AllowedFormats:   []string{"yaml", "yml", "json", "env"},
	}
	optimizer := NewConfigOptimizer(options)

	// 监控配置变化
	callbackCalled := false
	callback := func(config interface{}) error {
		callbackCalled = true
		return nil
	}

	if err := optimizer.WatchConfig(configPath, callback); err != nil {
		t.Fatalf("Failed to watch config: %v", err)
	}

	// 等待一下让监控器启动
	time.Sleep(200 * time.Millisecond)

	// 修改配置文件
	updatedConfig := "database:\n  host: updated-host\n  port: 5433\n"
	if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// 等待监控器检测变化
	time.Sleep(500 * time.Millisecond)

	// 验证回调被调用
	if !callbackCalled {
		t.Error("Callback should have been called")
	}

	// 获取统计信息
	stats := optimizer.GetStats()
	if stats.ReloadCount == 0 {
		t.Error("Reload count should be greater than 0")
	}
}

// 测试配置结构体

// TestConfigWithPaths 包含路径的测试配置
type TestConfigWithPaths struct {
	Logging struct {
		OutputPath string `config:"resolve_path"`
		ConfigPath string `config:"resolve_path"`
	} `config:"resolve_path"`
	Files struct {
		UploadDir string `config:"resolve_path"`
		TempDir   string `config:"resolve_path"`
	} `config:"resolve_path"`
}
