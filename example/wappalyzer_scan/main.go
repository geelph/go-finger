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
	"gxx/pkg/wappalyzer"
	"strings"
	"time"
)

func main() {
	startTime := time.Now()

	// 设置目标URL
	targets := []string{
		"https://github.com",
		"https://wordpress.org",
	}

	// 设置超时时间（秒）和代理（如需）
	timeout := 10
	proxy := "" // 如果需要代理，可以设置为 "http://127.0.0.1:8080"

	// 遍历所有目标
	for i, target := range targets {
		if i > 0 {
			fmt.Println("\n" + strings.Repeat("=", 50) + "\n")
		}

		fmt.Printf("开始分析目标: %s\n", target)
		fmt.Println("--------------------------------------------")

		// 方法1: 获取目标基本信息，包含技术栈信息
		analyzeSiteInfo(target, proxy, timeout)

		// 方法2: 单独进行技术栈分析
		analyzeWappalyzer(target, proxy, timeout)
	}

	fmt.Println("--------------------------------------------")
	fmt.Printf("总耗时: %s\n", time.Since(startTime))
	fmt.Printf("分析完成，共处理 %d 个目标\n", len(targets))
}

// analyzeSiteInfo 方法1：通过GetBaseInfo获取基本信息和技术栈
func analyzeSiteInfo(target, proxy string, timeout int) {
	// 获取目标基本信息
	baseInfo, err := gxx.GetBaseInfo(target, proxy, timeout)
	if err != nil {
		fmt.Printf("获取目标基本信息失败: %v\n", err)
		return
	}

	// 输出基本信息
	fmt.Printf("目标基本信息:\n")
	fmt.Printf("URL: %s\n", baseInfo.Target)
	fmt.Printf("状态码: %d\n", baseInfo.StatusCode)
	fmt.Printf("标题: %s\n", baseInfo.Title)
	if baseInfo.ServerInfo != nil {
		fmt.Printf("服务器: %s\n", baseInfo.ServerInfo.ServerType)
		if baseInfo.ServerInfo.Version != "" {
			fmt.Printf("服务器版本: %s\n", baseInfo.ServerInfo.Version)
		}
	}

	// 输出技术栈信息
	if baseInfo.Wappalyzer != nil {
		fmt.Println("\n通过GetBaseInfo获取的技术栈信息:")
		printWappalyzerInfo(baseInfo.Wappalyzer)
	}
}

// analyzeWappalyzer 方法2：使用专用WappalyzerScan API
func analyzeWappalyzer(target, proxy string, timeout int) {
	// 获取Wappalyzer分析结果
	wappalyzerResult, err := gxx.WappalyzerScan(target, proxy, timeout)
	if err != nil {
		fmt.Printf("Wappalyzer分析失败: %v\n", err)
		return
	}

	// 输出技术栈详细信息
	fmt.Println("\n通过WappalyzerScan获取的技术栈信息:")
	printWappalyzerInfo(wappalyzerResult)
}

// printWappalyzerInfo 打印技术栈信息
func printWappalyzerInfo(wapp *wappalyzer.TypeWappalyzer) {
	if wapp == nil {
		fmt.Println("未检测到技术栈信息")
		return
	}

	// 创建一个结构化的输出函数
	printCategory := func(name string, items []string) {
		if len(items) > 0 {
			fmt.Printf("%s:\n", name)
			for _, item := range items {
				fmt.Printf("  - %s\n", item)
			}
			fmt.Println()
		}
	}

	// 输出所有可能的技术类别
	printCategory("Web服务器", wapp.WebServers)
	printCategory("编程语言", wapp.ProgrammingLanguages)
	printCategory("Web框架", wapp.WebFrameworks)
	printCategory("JavaScript框架", wapp.JavaScriptFrameworks)
	printCategory("JavaScript库", wapp.JavaScriptLibraries)
	printCategory("安全组件", wapp.Security)
	printCategory("缓存系统", wapp.Caching)
	printCategory("反向代理", wapp.ReverseProxies)
	printCategory("静态站点生成器", wapp.StaticSiteGenerator)
	printCategory("主机面板", wapp.HostingPanels)
	printCategory("其他组件", wapp.Other)

	// 检查是否没有任何技术栈信息
	if !hasAnyWappalyzerData(wapp) {
		fmt.Println("未检测到任何技术栈信息")
	}
}

// hasAnyWappalyzerData 判断是否有任何技术栈数据
func hasAnyWappalyzerData(wapp *wappalyzer.TypeWappalyzer) bool {
	if wapp == nil {
		return false
	}

	// 检查所有可能的字段
	return len(wapp.WebServers) > 0 ||
		len(wapp.ProgrammingLanguages) > 0 ||
		len(wapp.WebFrameworks) > 0 ||
		len(wapp.JavaScriptFrameworks) > 0 ||
		len(wapp.JavaScriptLibraries) > 0 ||
		len(wapp.Security) > 0 ||
		len(wapp.Caching) > 0 ||
		len(wapp.ReverseProxies) > 0 ||
		len(wapp.StaticSiteGenerator) > 0 ||
		len(wapp.HostingPanels) > 0 ||
		len(wapp.Other) > 0
}
