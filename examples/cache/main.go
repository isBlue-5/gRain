// Package cache 提供缓存注解示例
package cache

import (
	"context"
	"fmt"
	"time"
)

// 演示缓存注解的使用
func RunCacheDemo() {
	fmt.Println("=== 缓存注解演示 ===")

	// 创建用户服务
	userService := NewUserService()

	// 创建上下文
	ctx := context.Background()

	fmt.Println("\n1. 首次获取用户 (应该从数据库获取)")
	user, err := userService.GetUserByID(ctx, 1)
	if err != nil {
		fmt.Printf("获取用户失败: %v\n", err)
	} else {
		fmt.Printf("获取到用户: ID=%d, 用户名=%s\n", user.ID, user.Username)
	}

	fmt.Println("\n2. 再次获取同一用户 (应该从缓存获取，不会显示数据库访问日志)")
	user, err = userService.GetUserByID(ctx, 1)
	if err != nil {
		fmt.Printf("获取用户失败: %v\n", err)
	} else {
		fmt.Printf("获取到用户: ID=%d, 用户名=%s\n", user.ID, user.Username)
	}

	fmt.Println("\n3. 根据用户名获取用户 (应该从数据库获取)")
	user, err = userService.GetUserByUsername(ctx, "admin")
	if err != nil {
		fmt.Printf("获取用户失败: %v\n", err)
	} else {
		fmt.Printf("获取到用户: ID=%d, 用户名=%s\n", user.ID, user.Username)
	}

	fmt.Println("\n4. 再次根据用户名获取用户 (应该从缓存获取)")
	user, err = userService.GetUserByUsername(ctx, "admin")
	if err != nil {
		fmt.Printf("获取用户失败: %v\n", err)
	} else {
		fmt.Printf("获取到用户: ID=%d, 用户名=%s\n", user.ID, user.Username)
	}

	fmt.Println("\n5. 获取所有用户 (应该从数据库获取)")
	users, err := userService.GetAllUsers(ctx)
	if err != nil {
		fmt.Printf("获取所有用户失败: %v\n", err)
	} else {
		fmt.Printf("获取到 %d 个用户\n", len(users))
		for _, u := range users {
			fmt.Printf("  - ID=%d, 用户名=%s\n", u.ID, u.Username)
		}
	}

	fmt.Println("\n6. 再次获取所有用户 (应该从缓存获取)")
	users, err = userService.GetAllUsers(ctx)
	if err != nil {
		fmt.Printf("获取所有用户失败: %v\n", err)
	} else {
		fmt.Printf("获取到 %d 个用户\n", len(users))
		for _, u := range users {
			fmt.Printf("  - ID=%d, 用户名=%s\n", u.ID, u.Username)
		}
	}

	fmt.Println("\n7. 更新用户信息")
	user.Email = "admin-updated@example.com"
	err = userService.UpdateUser(ctx, user)
	if err != nil {
		fmt.Printf("更新用户失败: %v\n", err)
	} else {
		fmt.Println("用户更新成功")
	}

	fmt.Println("\n8. 获取更新后的用户 (应该从数据库获取，因为缓存已被清除)")
	user, err = userService.GetUserByID(ctx, 1)
	if err != nil {
		fmt.Printf("获取用户失败: %v\n", err)
	} else {
		fmt.Printf("获取到用户: ID=%d, 用户名=%s, 邮箱=%s\n", user.ID, user.Username, user.Email)
	}

	fmt.Println("\n9. 等待缓存过期 (模拟)")
	fmt.Println("等待1秒...")
	time.Sleep(1 * time.Second)

	fmt.Println("\n10. 缓存过期后获取用户 (在实际应用中，这应该从数据库获取，但在本示例中我们没有实际等待缓存过期)")
	user, err = userService.GetUserByID(ctx, 1)
	if err != nil {
		fmt.Printf("获取用户失败: %v\n", err)
	} else {
		fmt.Printf("获取到用户: ID=%d, 用户名=%s\n", user.ID, user.Username)
	}

	fmt.Println("\n=== 缓存注解演示结束 ===")
}
