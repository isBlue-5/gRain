package database

import (
	"fmt"
	"log"
	"time"

	"complete_demo/config"
	"complete_demo/models"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database 数据库管理器
type Database struct {
	DB *gorm.DB
}

// NewDatabase 创建数据库连接
func NewDatabase(config *config.Config) (*Database, error) {
	var db *gorm.DB
	var err error

	// 根据配置选择数据库驱动
	switch config.Database.Type {
	case "sqlite":
		db, err = connectSQLite(config.Database)
	case "mysql":
		db, err = connectMySQL(config.Database)
	case "postgres":
		db, err = connectPostgreSQL(config.Database)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", config.Database.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 配置数据库连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 配置GORM日志
	db.Logger = db.Logger.LogMode(logger.Info)

	return &Database{DB: db}, nil
}

// connectSQLite 连接SQLite数据库
func connectSQLite(config config.DatabaseConfig) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(config.Path), &gorm.Config{})
}

// connectMySQL 连接MySQL数据库
func connectMySQL(config config.DatabaseConfig) (*gorm.DB, error) {
	dsn := config.GetDSN()
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

// connectPostgreSQL 连接PostgreSQL数据库
func connectPostgreSQL(config config.DatabaseConfig) (*gorm.DB, error) {
	dsn := config.GetDSN()
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// AutoMigrate 自动迁移数据库表
func (d *Database) AutoMigrate() error {
	log.Println("开始数据库迁移...")

	// 迁移用户表
	if err := d.DB.AutoMigrate(&models.User{}); err != nil {
		return fmt.Errorf("迁移用户表失败: %w", err)
	}

	// 迁移产品表
	if err := d.DB.AutoMigrate(&models.Product{}); err != nil {
		return fmt.Errorf("迁移产品表失败: %w", err)
	}

	log.Println("数据库迁移完成")
	return nil
}

// CreateTables 创建数据库表
func (d *Database) CreateTables() error {
	log.Println("开始创建数据库表...")

	// 创建用户表
	if err := d.DB.Migrator().CreateTable(&models.User{}); err != nil {
		return fmt.Errorf("创建用户表失败: %w", err)
	}

	// 创建产品表
	if err := d.DB.Migrator().CreateTable(&models.Product{}); err != nil {
		return fmt.Errorf("创建产品表失败: %w", err)
	}

	log.Println("数据库表创建完成")
	return nil
}

// DropTables 删除数据库表
func (d *Database) DropTables() error {
	log.Println("开始删除数据库表...")

	// 删除产品表
	if err := d.DB.Migrator().DropTable(&models.Product{}); err != nil {
		return fmt.Errorf("删除产品表失败: %w", err)
	}

	// 删除用户表
	if err := d.DB.Migrator().DropTable(&models.User{}); err != nil {
		return fmt.Errorf("删除用户表失败: %w", err)
	}

	log.Println("数据库表删除完成")
	return nil
}

// SeedData 填充测试数据
func (d *Database) SeedData() error {
	log.Println("开始填充测试数据...")

	// 检查是否已有数据
	var userCount int64
	d.DB.Model(&models.User{}).Count(&userCount)
	if userCount > 0 {
		log.Println("数据库中已有数据，跳过填充")
		return nil
	}

	// 创建管理员用户
	adminUser := &models.User{
		Username:  "admin",
		Email:     "admin@example.com",
		Password:  "admin123", // 实际应用中应该使用哈希密码
		FirstName: "Admin",
		LastName:  "User",
		Role:      "ADMIN",
		Status:    "ACTIVE",
	}

	if err := d.DB.Create(adminUser).Error; err != nil {
		return fmt.Errorf("创建管理员用户失败: %w", err)
	}

	// 创建测试用户
	testUser := &models.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "test123",
		FirstName: "Test",
		LastName:  "User",
		Role:      "USER",
		Status:    "ACTIVE",
	}

	if err := d.DB.Create(testUser).Error; err != nil {
		return fmt.Errorf("创建测试用户失败: %w", err)
	}

	// 创建测试产品
	testProduct := &models.Product{
		Name:        "测试产品",
		Description: "这是一个测试产品",
		Price:       99.99,
		Stock:       100,
		Category:    "电子产品",
		Brand:       "测试品牌",
		SKU:         "TEST001",
		Status:      "ACTIVE",
		CreatedBy:   adminUser.ID,
	}

	if err := d.DB.Create(testProduct).Error; err != nil {
		return fmt.Errorf("创建测试产品失败: %w", err)
	}

	log.Println("测试数据填充完成")
	return nil
}

// GetDB 获取数据库实例
func (d *Database) GetDB() *gorm.DB {
	return d.DB
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// HealthCheck 健康检查
func (d *Database) HealthCheck() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	return sqlDB.Ping()
}

// GetStats 获取数据库统计信息
func (d *Database) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 获取连接池统计
	sqlDB, err := d.DB.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	stats["max_open_connections"] = sqlDB.Stats().MaxOpenConnections
	stats["open_connections"] = sqlDB.Stats().OpenConnections
	stats["in_use"] = sqlDB.Stats().InUse
	stats["idle"] = sqlDB.Stats().Idle

	// 获取表统计
	var userCount int64
	var productCount int64

	d.DB.Model(&models.User{}).Count(&userCount)
	d.DB.Model(&models.Product{}).Count(&productCount)

	stats["user_count"] = userCount
	stats["product_count"] = productCount

	return stats, nil
}
