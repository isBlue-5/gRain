// Package main 提供GORM示例的入口点
package main

import (
	"log"

	"github.com/grain-framework/grain/examples/gorm"
)

func main() {
	log.Println("开始运行GORM示例...")
	gorm.RunExample()
}
