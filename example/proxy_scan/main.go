/*
 * Package main
 * 代理扫描示例：演示如何使用GXX通过代理服务器进行指纹识别
 * @Author: zhizhuo
 * @IDE: GoLand
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

	// 设置代理地址 - 支持http/https/socks5代理
	// 例如: http://127.0.0.1:8080 或 socks5://127.0.0.1:1080
	proxy := "http://127.0.0.1:8080"

	// 设置超时时间（秒）
	timeout := 10

	// 设置线程数
	workerCount := 500 // 规则并发线程数，使用默认值

	fmt.Printf("开始通过代理 %s 扫描目标: %s\n", proxy, target)
	fmt.Println("--------------------------------------------")

	// 初始化指纹规则库
	if err := gxx.InitFingerRules(options); err != nil {
		fmt.Printf("初始化指纹规则库失败: %v\n", err)
		os.Exit(1)
	}

	// 获取目标基本信息
	baseInfo, err := gxx.GetBaseInfo(target, proxy, timeout)
	if err != nil {
		fmt.Printf("获取目标基本信息失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("目标基本信息:\n")
	fmt.Printf("URL: %s\n", baseInfo.Target)
	fmt.Printf("状态码: %d\n", baseInfo.StatusCode)
	fmt.Printf("标题: %s\n", baseInfo.Title)
	if baseInfo.ServerInfo != nil {
		fmt.Printf("服务器: %s\n", baseInfo.ServerInfo.ServerType)
	}

	// 执行指纹识别
	result, err := gxx.FingerScan(target, proxy, timeout, workerCount)
	if err != nil {
		fmt.Printf("扫描失败: %v\n", err)
		os.Exit(1)
	}

	// 获取匹配的指纹
	matches := gxx.GetFingerMatches(result)

	// 输出匹配结果
	if len(matches) > 0 {
		fmt.Printf("\n匹配到 %d 个指纹:\n", len(matches))
		for i, match := range matches {
			fmt.Printf("  %d. %s\n", i+1, match.Finger.Info.Name)
		}
	} else {
		fmt.Println("\n未匹配到任何指纹")
	}

	// 输出技术栈信息
	if result.Wappalyzer != nil {
		fmt.Println("\n技术栈信息:")
		if len(result.Wappalyzer.WebServers) > 0 {
			fmt.Printf("  Web服务器: %v\n", result.Wappalyzer.WebServers)
		}
		if len(result.Wappalyzer.ProgrammingLanguages) > 0 {
			fmt.Printf("  编程语言: %v\n", result.Wappalyzer.ProgrammingLanguages)
		}
		if len(result.Wappalyzer.WebFrameworks) > 0 {
			fmt.Printf("  Web框架: %v\n", result.Wappalyzer.WebFrameworks)
		}
	}

	fmt.Println("--------------------------------------------")
	fmt.Printf("总耗时: %s\n", time.Since(startTime))
	fmt.Println("扫描完成")
}
