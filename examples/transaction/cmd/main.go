// Package main 提供事务示例的入口点
package main

import (
	"log"

	"github.com/isBlue-5/grain/examples/transaction"
)

func main() {
	log.Println("开始运行事务示例...")
	transaction.RunExample()
}
