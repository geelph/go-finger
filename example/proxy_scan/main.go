/*
 * Package main
 * 代理扫描示例：演示如何使用代理进行指纹识别扫描
 * @Author: zhizhuo
 * @IDE：GoLand
 * @File: main.go
 * @Date: 2025/3/10 下午2:11
 */
package main

import (
	"fmt"
	"gxx"
	"os"
	"time"
)

func main() {
	startTime := time.Now()

	// 创建新的扫描选项
	options, err := gxx.NewFingerOptions()
	if err != nil {
		fmt.Printf("创建选项失败: %v\n", err)
		os.Exit(1)
	}

	// 设置目标URL
	target := "example.com"
	
	// 设置代理 - 支持HTTP或SOCKS5代理
	// 格式: http://127.0.0.1:8080 或 socks5://127.0.0.1:1080
	// 注意: 使用前请确保代理地址正确，此处示例使用的代理可能不存在
	proxy := "http://127.0.0.1:7890"
	
	// 启用调试模式以查看详细日志
	options.Debug = true
	
	// 可选：设置超时时间（秒）- 使用代理时可能需要更长的超时时间
	timeout := 10
	
	// 可选：设置重试次数
	// 注意: 该属性需要在指纹实现中支持才能生效
	options.Retries = 2

	// 可选：设置并发工作线程数
	workerCount := 5

	// 执行扫描
	fmt.Printf("开始通过代理扫描目标: %s\n", target)
	fmt.Printf("使用代理: %s\n", proxy)
	fmt.Println("--------------------------------------------")
	
	// 初始化指纹规则库
	if err := gxx.InitFingerRules(options); err != nil {
		fmt.Printf("初始化指纹规则库失败: %v\n", err)
		os.Exit(1)
	}
	
	// 1. 首先获取目标基础信息
	fmt.Println("获取目标基础信息...")
	title, serverInfo, statusCode, _, err := gxx.GetBaseInfo(target, proxy, timeout)
	if err != nil {
		fmt.Printf("获取基础信息失败: %v\n", err)
	} else {
		fmt.Printf("基础信息:\n")
		fmt.Printf("  - 状态码: %d\n", statusCode)
		fmt.Printf("  - 标题: %s\n", title)
		if serverInfo != nil {
			fmt.Printf("  - 服务器: %s\n", serverInfo.ServerType)
		}
		fmt.Println()
	}
	
	// 2. 执行指纹识别
	fmt.Println("开始指纹识别...")
	result, err := gxx.FingerScan(target, proxy, timeout, workerCount)
	if err != nil {
		fmt.Printf("指纹识别失败: %v\n", err)
		os.Exit(1)
	}
	
	// 3. 输出结果
	fmt.Println("指纹识别结果:")
	fmt.Printf("URL: %s, 状态码: %d, 标题: %s\n", 
		result.URL, result.StatusCode, result.Title)
	
	// 输出匹配的指纹
	matches := gxx.GetFingerMatches(result)
	if len(matches) > 0 {
		fmt.Printf("匹配到 %d 个指纹:\n", len(matches))
		for i, match := range matches {
			fmt.Printf("  %d. %s (匹配结果: %v)\n", 
				i+1, match.Finger.Info.Name, match.Result)
		}
	} else {
		fmt.Println("未匹配到任何指纹")
	}

	fmt.Println("--------------------------------------------")
	fmt.Printf("总耗时: %s\n", time.Since(startTime))
	fmt.Println("扫描完成")
} 