package main

import (
	"log"

	"github.com/isBlue-5/gRain/pkg/core"
)

// 这是一个最简单的gRain应用示例
// 框架会在启动时自动完成所有代码生成和依赖注入工作

func main() {
	log.Println("🚀 启动gRain框架示例应用...")

	// 创建应用实例 - 就是这么简单！
	app := core.New(
		core.WithPort("8080"),
		core.WithDebug(true),
		core.WithAutoGenerate(true), // 启用自动代码生成
	)

	// 运行应用 - 框架会自动处理一切！
	if err := app.Run(); err != nil {
		log.Fatalf("应用运行失败: %v", err)
	}
}
