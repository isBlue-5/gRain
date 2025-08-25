// Package main 提供事务示例的入口点
package main

import (
	"log"

	"github.com/grain-framework/grain/examples/transaction"
)

func main() {
	log.Println("开始运行事务示例...")
	transaction.RunExample()
}
