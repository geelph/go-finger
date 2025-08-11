/*
 * Package main
 * 基本扫描示例：演示如何使用GXX进行单个或多个目标的指纹识别
 * @Author: zhizhuo
 * @IDE：GoLand
 * @File: main.go
 * @Date: 2025/3/10 下午2:11
 */
package main

import (
	"fmt"
	"gxx"
	"gxx/pkg/wappalyzer"
	_ "gxx/types"
	"os"
	"time"
)

func main() {
	startTime := time.Now()

	// 1. 创建新的扫描选项，用于初始化指纹规则库
	options, err := gxx.NewFingerOptions()
	if err != nil {
		fmt.Printf("创建选项失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 设置目标URL列表，可以添加多个目标
	targets := []string{"example.com", "github.com"}

	// 3. 配置基本扫描参数
	timeout := 5       // 超时时间（秒）
	workerCount := 5000 // 规则并发线程数，更高的值可提高识别速度

	// 4. 初始化打印
	fmt.Println("开始扫描目标:", targets)
	fmt.Println("--------------------------------------------")

	// 5. 初始化指纹规则库（只需执行一次）
	if err := gxx.InitFingerRules(options); err != nil {
		fmt.Printf("初始化指纹规则库失败: %v\n", err)
		os.Exit(1)
	}

	// 6. 逐个扫描目标
	for _, target := range targets {
		// 打印当前目标
		fmt.Printf("扫描目标: %s\n", target)

		// 使用API接口扫描单个目标
		// 第一个参数：目标URL
		// 第二个参数：代理地址，为空表示不使用代理
		// 第三个参数：超时时间（秒）
		// 第四个参数：规则并发线程数，影响识别速度
		result, err := gxx.FingerScan(target, "", timeout, workerCount)
		if err != nil {
			fmt.Printf("扫描失败: %v\n", err)
			continue
		}

		// 7. 输出基础信息
		fmt.Printf("URL: %s, 状态码: %d, 标题: %s\n",
			result.URL, result.StatusCode, result.Title)

		if result.Server != nil {
			fmt.Printf("服务器: %s\n", result.Server.ServerType)
		}

		// 8. 输出匹配的指纹
		matches := gxx.GetFingerMatches(result)
		if len(matches) > 0 {
			fmt.Printf("\n匹配到 %d 个指纹:\n", len(matches))
			for i, match := range matches {
				fmt.Printf("  %d. %s\n", i+1, match.Finger.Info.Name)
				// 如果需要更详细的信息，可以取消下面的注释
				// fmt.Printf("     ID: %s, 匹配结果: %v\n", match.Finger.Id, match.Result)
			}
		} else {
			fmt.Println("\n未匹配到任何指纹")
		}

		// 9. 输出技术栈信息（如果有）
		if result.Wappalyzer != nil && hasWappalyzerData(result.Wappalyzer) {
			fmt.Println("\n技术栈信息:")
			printWappalyzerInfo(result.Wappalyzer)
		}

		fmt.Println()
	}

	// 10. 输出总结信息
	fmt.Println("--------------------------------------------")
	fmt.Printf("总耗时: %s\n", time.Since(startTime))
	fmt.Println("扫描完成")
}

// 检查Wappalyzer结果是否有数据
func hasWappalyzerData(wappalyzer *wappalyzer.TypeWappalyzer) bool {
	if wappalyzer == nil {
		return false
	}

	return len(wappalyzer.WebServers) > 0 ||
		len(wappalyzer.ProgrammingLanguages) > 0 ||
		len(wappalyzer.WebFrameworks) > 0 ||
		len(wappalyzer.JavaScriptFrameworks) > 0 ||
		len(wappalyzer.JavaScriptLibraries) > 0 ||
		len(wappalyzer.Security) > 0
}

// 打印Wappalyzer技术栈信息
func printWappalyzerInfo(wappalyzer *wappalyzer.TypeWappalyzer) {
	if len(wappalyzer.WebServers) > 0 {
		fmt.Printf("  Web服务器: %v\n", wappalyzer.WebServers)
	}
	if len(wappalyzer.ProgrammingLanguages) > 0 {
		fmt.Printf("  编程语言: %v\n", wappalyzer.ProgrammingLanguages)
	}
	if len(wappalyzer.WebFrameworks) > 0 {
		fmt.Printf("  Web框架: %v\n", wappalyzer.WebFrameworks)
	}
	if len(wappalyzer.JavaScriptFrameworks) > 0 {
		fmt.Printf("  JS框架: %v\n", wappalyzer.JavaScriptFrameworks)
	}
	if len(wappalyzer.JavaScriptLibraries) > 0 {
		fmt.Printf("  JS库: %v\n", wappalyzer.JavaScriptLibraries)
	}
	if len(wappalyzer.Security) > 0 {
		fmt.Printf("  安全组件: %v\n", wappalyzer.Security)
	}
}
