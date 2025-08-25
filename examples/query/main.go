// Package query 提供自定义查询注解示例
package query

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 演示自定义查询功能
func RunQueryDemo() {
	// 设置GORM日志
	gormLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// 连接数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Fatalf("无法连接数据库: %v", err)
	}

	// 自动迁移
	err = db.AutoMigrate(&User{})
	if err != nil {
		log.Fatalf("自动迁移失败: %v", err)
	}

	// 创建用户仓库
	userRepo := NewUserRepository(db)

	// 创建上下文
	ctx := context.Background()

	// 插入测试数据
	users := []*User{
		{Username: "admin", Email: "admin@example.com", Age: 30, Status: "active"},
		{Username: "user1", Email: "user1@example.com", Age: 25, Status: "active"},
		{Username: "user2", Email: "user2@example.com", Age: 20, Status: "inactive"},
		{Username: "user3", Email: "user3@example.com", Age: 35, Status: "active"},
		{Username: "user4", Email: "user4@example.com", Age: 40, Status: "inactive"},
	}

	for _, user := range users {
		if err := userRepo.Save(ctx, user); err != nil {
			log.Fatalf("插入用户失败: %v", err)
		}
	}

	fmt.Println("=== 自定义查询演示 ===")

	// 1. FindByUsername
	fmt.Println("\n1. 根据用户名查找用户:")
	user, err := userRepo.FindByUsername(ctx, "admin")
	if err != nil {
		fmt.Printf("查找用户失败: %v\n", err)
	} else {
		fmt.Printf("找到用户: ID=%d, 用户名=%s, 邮箱=%s\n", user.ID, user.Username, user.Email)
	}

	// 2. FindByEmail
	fmt.Println("\n2. 根据邮箱查找用户:")
	user, err = userRepo.FindByEmail(ctx, "user1@example.com")
	if err != nil {
		fmt.Printf("查找用户失败: %v\n", err)
	} else if user != nil {
		fmt.Printf("找到用户: ID=%d, 用户名=%s, 邮箱=%s\n", user.ID, user.Username, user.Email)
	} else {
		fmt.Println("未找到用户")
	}

	// 3. FindAllByStatus
	fmt.Println("\n3. 根据状态查找用户:")
	activeUsers, err := userRepo.FindAllByStatus(ctx, "active", 10, 0)
	if err != nil {
		fmt.Printf("查找用户失败: %v\n", err)
	} else {
		fmt.Printf("找到 %d 个活跃用户:\n", len(activeUsers))
		for _, u := range activeUsers {
			fmt.Printf("  - ID=%d, 用户名=%s, 状态=%s\n", u.ID, u.Username, u.Status)
		}
	}

	// 4. FindAllByAgeGreaterThan
	fmt.Println("\n4. 查找年龄大于30的用户:")
	olderUsers, err := userRepo.FindAllByAgeGreaterThan(ctx, 30)
	if err != nil {
		fmt.Printf("查找用户失败: %v\n", err)
	} else {
		fmt.Printf("找到 %d 个年龄大于30的用户:\n", len(olderUsers))
		for _, u := range olderUsers {
			fmt.Printf("  - ID=%d, 用户名=%s, 年龄=%d\n", u.ID, u.Username, u.Age)
		}
	}

	// 5. FindAllByAgeBetween
	fmt.Println("\n5. 查找年龄在20-30之间的用户:")
	middleAgeUsers, err := userRepo.FindAllByAgeBetween(ctx, 20, 30)
	if err != nil {
		fmt.Printf("查找用户失败: %v\n", err)
	} else {
		fmt.Printf("找到 %d 个年龄在20-30之间的用户:\n", len(middleAgeUsers))
		for _, u := range middleAgeUsers {
			fmt.Printf("  - ID=%d, 用户名=%s, 年龄=%d\n", u.ID, u.Username, u.Age)
		}
	}

	// 6. CountByStatus
	fmt.Println("\n6. 统计不同状态的用户数量:")
	activeCount, err := userRepo.CountByStatus(ctx, "active")
	if err != nil {
		fmt.Printf("统计失败: %v\n", err)
	} else {
		fmt.Printf("活跃用户数量: %d\n", activeCount)
	}

	inactiveCount, err := userRepo.CountByStatus(ctx, "inactive")
	if err != nil {
		fmt.Printf("统计失败: %v\n", err)
	} else {
		fmt.Printf("非活跃用户数量: %d\n", inactiveCount)
	}

	// 7. ExistsByUsername
	fmt.Println("\n7. 检查用户名是否存在:")
	exists, err := userRepo.ExistsByUsername(ctx, "admin")
	if err != nil {
		fmt.Printf("检查失败: %v\n", err)
	} else {
		fmt.Printf("用户名 'admin' 存在: %v\n", exists)
	}

	exists, err = userRepo.ExistsByUsername(ctx, "nonexistent")
	if err != nil {
		fmt.Printf("检查失败: %v\n", err)
	} else {
		fmt.Printf("用户名 'nonexistent' 存在: %v\n", exists)
	}

	// 8. DeleteByStatus
	fmt.Println("\n8. 删除非活跃用户:")
	deleted, err := userRepo.DeleteByStatus(ctx, "inactive")
	if err != nil {
		fmt.Printf("删除失败: %v\n", err)
	} else {
		fmt.Printf("已删除 %d 个非活跃用户\n", deleted)
	}

	// 9. 验证删除结果
	fmt.Println("\n9. 验证删除结果:")
	allUsers, err := userRepo.FindAll(ctx)
	if err != nil {
		fmt.Printf("查找用户失败: %v\n", err)
	} else {
		fmt.Printf("剩余用户数量: %d\n", len(allUsers))
		for _, u := range allUsers {
			fmt.Printf("  - ID=%d, 用户名=%s, 状态=%s\n", u.ID, u.Username, u.Status)
		}
	}

	fmt.Println("\n=== 自定义查询演示结束 ===")
}
