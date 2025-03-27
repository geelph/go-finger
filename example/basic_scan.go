/*
  - Package example
    @Author: zhizhuo
    @IDE：GoLand
    @File: basic_scan.go
    @Date: 2025/3/10 下午2:11*
*/
package main

import (
	"fmt"
	"gxx"
)

func main() {
	// 创建新的扫描选项
	options, err := gxx.NewFingerOptions()
	if err != nil {
		fmt.Printf("创建选项失败: %v\n", err)
		return
	}

	// 设置目标
	options.Target = []string{"example.com"}
	options.Debug = true

	// 执行扫描
	gxx.FingerScan(options)
}
