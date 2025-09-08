package main

import (
	"fmt"
	"log"

	"github.com/isBlue-5/grain/pkg/annotation/processor"
	"github.com/isBlue-5/grain/pkg/annotation/registry"
)

func main() {
	fmt.Println("开始测试代码生成器...")

	// 创建注解解析器
	parser := processor.NewAnnotationParser("frame:")

	// 创建注册中心
	reg := registry.NewRegistry()

	// 创建生成器
	generator := processor.NewGenerator(parser, reg, "./test-output", "frame:")

	// 测试解析器
	fmt.Println("✓ 注解解析器创建成功")
	fmt.Println("✓ 注册中心创建成功")
	fmt.Println("✓ 代码生成器创建成功")

	// 测试解析示例目录
	paths := []string{"./examples/annotation"}

	fmt.Println("\n开始解析注解...")
	if err := generator.Generate(paths); err != nil {
		log.Printf("代码生成失败: %v", err)
	} else {
		fmt.Println("✓ 代码生成成功完成")
	}

	fmt.Println("\n测试完成!")
}
