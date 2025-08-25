// Package main 提供GORM示例的入口点
package main

import (
	"log"

	"github.com/isBlue-5/grain/examples/gorm"
)

func main() {
	log.Println("开始运行GORM示例...")
	gorm.RunExample()
}
