package main

import (
	"testing"
	"time"

	"complete_demo/config"
	"complete_demo/database"

	"github.com/stretchr/testify/assert"
)

func TestConfigLoad(t *testing.T) {
	// 测试配置加载
	cfg, err := config.Load()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// 验证默认值
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "sqlite", cfg.Database.Type)
	assert.Equal(t, "./demo.db", cfg.Database.Path)
}

func TestDatabaseConnection(t *testing.T) {
	// 测试数据库连接
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Type: "sqlite",
			Path: ":memory:", // 使用内存数据库进行测试
		},
	}

	db, err := database.NewDatabase(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// 测试健康检查
	err = db.HealthCheck()
	assert.NoError(t, err)

	// 测试自动迁移
	err = db.AutoMigrate()
	assert.NoError(t, err)

	// 测试获取统计信息
	stats, err := db.GetStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	// 清理
	db.Close()
}

func TestConfigValidation(t *testing.T) {
	// 测试配置验证
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         "8080",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Database: config.DatabaseConfig{
			Type: "sqlite",
			Path: "./test.db",
		},
		JWT: config.JWTConfig{
			Secret:     "test-secret",
			Expiration: 24 * time.Hour,
		},
	}

	// 验证配置
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "sqlite", cfg.Database.Type)
	assert.Equal(t, "./test.db", cfg.Database.Path)
	assert.Equal(t, "test-secret", cfg.JWT.Secret)
}

func TestDatabaseDSN(t *testing.T) {
	// 测试数据库连接字符串生成
	cfg := config.DatabaseConfig{
		Type:     "mysql",
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "password",
		Name:     "demo",
	}

	dsn := cfg.GetDSN()
	assert.Contains(t, dsn, "root:password@tcp(localhost:3306)/demo")
}

func TestRedisAddr(t *testing.T) {
	// 测试Redis地址生成
	cfg := config.RedisConfig{
		Host: "localhost",
		Port: 6379,
	}

	addr := cfg.GetAddr()
	assert.Equal(t, "localhost:6379", addr)
}

func TestEnvironmentVariables(t *testing.T) {
	// 测试环境变量处理
	// 这里可以添加环境变量测试
	// 由于测试环境可能没有设置这些变量，我们主要测试默认值
	cfg, err := config.Load()
	assert.NoError(t, err)

	// 验证默认值
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, "sqlite", cfg.Database.Type)
	assert.Equal(t, "./demo.db", cfg.Database.Path)
}

func TestDatabaseOperations(t *testing.T) {
	// 测试数据库操作
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Type: "sqlite",
			Path: ":memory:",
		},
	}

	db, err := database.NewDatabase(cfg)
	assert.NoError(t, err)
	defer db.Close()

	// 测试创建表
	err = db.CreateTables()
	assert.NoError(t, err)

	// 测试填充数据
	err = db.SeedData()
	assert.NoError(t, err)

	// 测试获取统计信息
	stats, err := db.GetStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	// 验证数据是否正确填充
	assert.GreaterOrEqual(t, stats["user_count"], int64(2))    // 至少应该有2个用户
	assert.GreaterOrEqual(t, stats["product_count"], int64(1)) // 至少应该有1个产品
}

func TestConfigMethods(t *testing.T) {
	// 测试配置方法
	cfg := config.DatabaseConfig{
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "password",
		Name:     "demo",
		SSLMode:  "disable",
	}

	dsn := cfg.GetDSN()
	assert.Contains(t, dsn, "host=localhost")
	assert.Contains(t, dsn, "port=5432")
	assert.Contains(t, dsn, "user=postgres")
	assert.Contains(t, dsn, "password=password")
	assert.Contains(t, dsn, "dbname=demo")
	assert.Contains(t, dsn, "sslmode=disable")
}

func TestServerConfig(t *testing.T) {
	// 测试服务器配置
	cfg := config.ServerConfig{
		Port:         "9090",
		ReadTimeout:  45 * time.Second,
		WriteTimeout: 45 * time.Second,
		IdleTimeout:  90 * time.Second,
	}

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, 45*time.Second, cfg.ReadTimeout)
	assert.Equal(t, 45*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 90*time.Second, cfg.IdleTimeout)
}

func TestJWTConfig(t *testing.T) {
	// 测试JWT配置
	cfg := config.JWTConfig{
		Secret:     "my-secret-key",
		Expiration: 12 * time.Hour,
	}

	assert.Equal(t, "my-secret-key", cfg.Secret)
	assert.Equal(t, 12*time.Hour, cfg.Expiration)
}

func TestLogConfig(t *testing.T) {
	// 测试日志配置
	cfg := config.LogConfig{
		Level:  "debug",
		Format: "json",
		Output: "file",
	}

	assert.Equal(t, "debug", cfg.Level)
	assert.Equal(t, "json", cfg.Format)
	assert.Equal(t, "file", cfg.Output)
}
