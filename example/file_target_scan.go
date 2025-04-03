/*
 * Package example
 * 文件目标扫描示例：演示如何从文件中读取目标列表进行批量扫描
 * @Author: zhizhuo
 * @IDE：GoLand
 * @File: file_target_scan.go
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

	// 设置目标文件路径 - 文件中每行包含一个目标URL
	// 确保文件存在且有读取权限
	targetsFile := "targets.txt"
	if _, err := os.Stat(targetsFile); os.IsNotExist(err) {
		fmt.Printf("目标文件不存在: %s\n", targetsFile)
		os.Exit(1)
	}
	options.TargetsFile = targetsFile
	
	// 启用调试模式以查看详细日志
	options.Debug = true
	
	// 可选：设置并发线程数 - 对于大量目标，可以适当增加
	options.Threads = 20
	
	// 可选：设置输出文件
	options.Output = "scan_results.txt"

	// 执行扫描
	fmt.Println("开始从文件扫描目标:", options.TargetsFile)
	gxx.FingerScan(options)
	fmt.Println("扫描完成，结果保存在:", options.Output)
}
