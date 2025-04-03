/*
 * Package example
 * 基本扫描示例：演示如何使用GXX进行单个目标的指纹识别
 * @Author: zhizhuo
 * @IDE：GoLand
 * @File: basic_scan.go
 * @Date: 2025/3/10 下午2:11
 */
package main

import (
	"fmt"
	"gxx"
	"os"
)

func main() {
	// 创建新的扫描选项
	options, err := gxx.NewFingerOptions()
	if err != nil {
		fmt.Printf("创建选项失败: %v\n", err)
		os.Exit(1)
	}

	// 设置目标URL - 可以设置多个目标
	options.Target = []string{"example.com"}
	
	// 启用调试模式以查看详细日志
	options.Debug = true
	
	// 可选：设置超时时间（秒）
	options.Timeout = 5
	
	// 可选：设置线程数
	options.Threads = 5

	// 执行扫描并输出结果
	fmt.Println("开始扫描目标:", options.Target)
	gxx.FingerScan(options)
	fmt.Println("扫描完成")
}
