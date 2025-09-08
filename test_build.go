//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("开始构建测试...")

	// 测试编译各个包
	packages := []string{
		"./pkg/annotation/types",
		"./pkg/annotation/registry",
		"./pkg/annotation/processor",
		"./cmd/ginframe-gen",
	}

	for _, pkg := range packages {
		fmt.Printf("编译包: %s\n", pkg)

		cmd := exec.Command("go", "build", pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Printf("✗ 包 %s 编译失败: %v\n", pkg, err)
			os.Exit(1)
		} else {
			fmt.Printf("✓ 包 %s 编译成功\n", pkg)
		}
	}

	fmt.Println("✓ 所有包编译成功!")
}
