package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("gen: 开始从数据库生成 model / query 代码...")

	GenDB()

	fmt.Println("gen: 完成")
	os.Exit(0)
}
