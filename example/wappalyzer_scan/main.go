/*
 * Package main
 * Wappalyzer扫描示例：演示如何使用GXX的Wappalyzer功能进行技术栈识别
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

	// 设置目标URL
	target := "https://github.com"

	// 设置超时时间（秒）
	timeout := 10

	fmt.Printf("开始分析目标: %s\n", target)
	fmt.Println("--------------------------------------------")

	// 获取目标基本信息
	baseInfo, err := gxx.GetBaseInfo(target, "", timeout)
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

	// 获取Wappalyzer分析结果
	wappalyzerResult, err := gxx.WappalyzerScan(target, "", timeout)
	if err != nil {
		fmt.Printf("Wappalyzer分析失败: %v\n", err)
		os.Exit(1)
	}

	// 输出技术栈详细信息
	fmt.Println("\n技术栈详细信息:")

	if len(wappalyzerResult.WebServers) > 0 {
		fmt.Printf("Web服务器:\n")
		for _, server := range wappalyzerResult.WebServers {
			fmt.Printf("  - %s\n", server)
		}
	}

	if len(wappalyzerResult.ProgrammingLanguages) > 0 {
		fmt.Printf("\n编程语言:\n")
		for _, lang := range wappalyzerResult.ProgrammingLanguages {
			fmt.Printf("  - %s\n", lang)
		}
	}

	if len(wappalyzerResult.WebFrameworks) > 0 {
		fmt.Printf("\nWeb框架:\n")
		for _, framework := range wappalyzerResult.WebFrameworks {
			fmt.Printf("  - %s\n", framework)
		}
	}

	if len(wappalyzerResult.JavaScriptFrameworks) > 0 {
		fmt.Printf("\nJavaScript框架:\n")
		for _, framework := range wappalyzerResult.JavaScriptFrameworks {
			fmt.Printf("  - %s\n", framework)
		}
	}

	fmt.Println("--------------------------------------------")
	fmt.Printf("总耗时: %s\n", time.Since(startTime))
	fmt.Println("分析完成")
}
