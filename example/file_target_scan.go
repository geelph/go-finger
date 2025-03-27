/*
  - Package example
    @Author: zhizhuo
    @IDE：GoLand
    @File: file_target_scan.go
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

	// 设置目标文件
	options.TargetsFile = "targets.txt"
	// 设置调试模式
	options.Debug = true

	// 执行扫描
	gxx.FingerScan(options)
}
