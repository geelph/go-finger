/*
 * Package example
 * 代理扫描示例：演示如何使用代理进行指纹识别扫描
 * @Author: zhizhuo
 * @IDE：GoLand
 * @File: proxy_scan.go
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

	// 设置目标URL
	options.Target = []string{"example.com"}
	
	// 设置代理 - 支持HTTP或SOCKS5代理
	// 格式: http://127.0.0.1:8080 或 socks5://127.0.0.1:1080
	options.Proxy = "http://127.0.0.1:7890"
	
	// 启用调试模式以查看详细日志
	options.Debug = true
	
	// 可选：设置超时时间（秒）- 使用代理时可能需要更长的超时时间
	options.Timeout = 10
	
	// 可选：设置重试次数
	options.Retries = 2

	// 执行扫描
	fmt.Println("开始通过代理扫描目标:", options.Target)
	fmt.Println("使用代理:", options.Proxy)
	gxx.FingerScan(options)
	fmt.Println("扫描完成")
}
