/*
 * Package main
 * 基本扫描示例：演示如何使用GXX进行单个目标的指纹识别
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
	fmt.Println("--------------------------------------------")

	// 初始化指纹规则库
	if err := gxx.InitFingerRules(options); err != nil {
		fmt.Printf("初始化指纹规则库失败: %v\n", err)
		os.Exit(1)
	}

	// 对每个目标单独扫描
	for _, target := range options.Target {
		fmt.Printf("扫描目标: %s\n", target)

		// 使用API接口方式扫描单个目标
		result, err := gxx.FingerScan(target, options.Proxy, options.Timeout, options.Threads)
		if err != nil {
			fmt.Printf("扫描失败: %v\n", err)
			continue
		}

		// 输出基本信息
		fmt.Printf("URL: %s, 状态码: %d, 标题: %s\n",
			result.URL, result.StatusCode, result.Title)

		// 输出匹配的指纹
		matches := gxx.GetFingerMatches(result)
		if len(matches) > 0 {
			fmt.Printf("匹配到 %d 个指纹:\n", len(matches))
			for i, match := range matches {
				fmt.Printf("  %d. %s\n", i+1, match.Finger.Info.Name)
			}
		} else {
			fmt.Println("未匹配到任何指纹")
		}

		fmt.Println()
	}

	fmt.Println("--------------------------------------------")
	fmt.Printf("总耗时: %s\n", time.Since(startTime))
	fmt.Println("扫描完成")
}

// CLI 命令行接口包装
type CLI struct {
	options *gxx.CmdOptions
}

// NewCLI 创建新的命令行接口
func NewCLI(options *gxx.CmdOptions) *CLI {
	return &CLI{options: options}
}

// Run 执行扫描
func (c *CLI) Run() {
	// 调用gxx库的FingerScan函数
	// 该函数会自动加载指纹规则库并执行扫描
	fmt.Println("正在扫描...")

	// 对于每个目标，我们可以单独调用API
	for _, target := range c.options.Target {
		// 初始化指纹规则库（如果尚未初始化）
		if err := gxx.InitFingerRules(c.options); err != nil {
			fmt.Printf("初始化指纹规则库失败: %v\n", err)
			continue
		}

		// 使用API接口方式扫描单个目标
		fmt.Printf("扫描目标: %s\n", target)
		result, err := gxx.FingerScan(target, c.options.Proxy, c.options.Timeout, c.options.Threads)
		if err != nil {
			fmt.Printf("扫描失败: %v\n", err)
			continue
		}

		// 输出基本信息
		fmt.Printf("URL: %s, 状态码: %d, 标题: %s\n",
			result.URL, result.StatusCode, result.Title)

		// 输出匹配的指纹
		matches := gxx.GetFingerMatches(result)
		if len(matches) > 0 {
			fmt.Printf("匹配到 %d 个指纹:\n", len(matches))
			for i, match := range matches {
				fmt.Printf("  %d. %s\n", i+1, match.Finger.Info.Name)
			}
		} else {
			fmt.Println("未匹配到任何指纹")
		}

		fmt.Println()
	}
}
