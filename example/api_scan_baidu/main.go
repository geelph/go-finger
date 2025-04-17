/*
 * Package main
 * API扫描示例：演示如何使用GXX的API接口直接进行指纹识别，目标为百度
 * @Author: zhizhuo
 * @IDE：GoLand
 * @File: main.go
 * @Date: 2025/3/10 下午2:11
 */
package main

import (
	"encoding/json"
	"fmt"
	"gxx"
	"gxx/utils/logger"
	"log"
	"os"
	"time"
)

func main() {
	// 1. 创建配置选项
	logLevel := 1     // 日志级别
	NoFileLog := true //是否关闭log文件的存储
	logger.InitLogger("logs", 5, logLevel, NoFileLog)
	options, err := gxx.NewFingerOptions()
	if err != nil {
		log.Fatalf("创建选项错误: %v", err)
	}
	options.PocYaml = "/Users/zhizhuo/Desktop/开发目录/Dev/gxx/finger_demo.yml"
	// 2. 初始化指纹规则库（仅需执行一次）
	fmt.Println("初始化指纹规则库...")
	startTime := time.Now()
	if err := gxx.InitFingerRules(options); err != nil {
		log.Fatalf("初始化指纹规则错误: %v", err)
	}
	fmt.Printf("初始化完成，耗时: %s\n", time.Since(startTime))

	// 3. 设置扫描参数
	target := "https://www.baidu.com"
	proxy := ""       // 如果需要代理，可以设置为 "http://127.0.0.1:8080" 或 "socks5://127.0.0.1:1080"
	timeout := 5      // 超时时间，单位：秒
	workerCount := 10 // 并发工作线程数

	// 4. 获取基础信息
	fmt.Printf("\n开始获取目标基础信息: %s\n", target)
	result, err := gxx.GetBaseInfo(target, proxy, timeout)
	if err != nil {
		fmt.Printf("获取基础信息失败: %v\n", err)
	} else {
		fmt.Printf("状态码: %d\n", result.StatusCode)
		fmt.Printf("标题: %s\n", result.Title)
		if result.ServerInfo != nil {
			fmt.Printf("服务器: %s\n", result.ServerInfo.ServerType)
		}
		fmt.Printf("站点技术: %s\n", result.Wappalyzer)
	}

	// 5. 执行指纹识别
	fmt.Printf("\n开始进行指纹识别: %s\n", target)
	startTime = time.Now()
	res, err := gxx.FingerScan(target, proxy, timeout, workerCount)
	if err != nil {
		fmt.Printf("指纹识别失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("指纹识别完成，耗时: %s\n", time.Since(startTime))

	// 6. 处理识别结果
	fmt.Printf("\n识别结果:\n")
	fmt.Printf("URL: %s\n", res.URL)
	fmt.Printf("状态码: %d\n", res.StatusCode)
	fmt.Printf("标题: %s\n", res.Title)
	if res.Server != nil {
		fmt.Printf("服务器: %s\n", res.Server.ServerType)
	}

	// 7. 输出匹配详情（JSON格式）
	fmt.Println("\n匹配详情(JSON格式):")
	jsonData, err := json.MarshalIndent(result.Wappalyzer, "", "  ")
	if err != nil {
		fmt.Printf("JSON转换错误: %v\n", err)

	}
	fmt.Printf("指纹 \n%s\n\n", string(jsonData))

	// 8. 处理匹配的指纹（友好格式）
	matches := gxx.GetFingerMatches(res)
	if len(matches) > 0 {
		fmt.Printf("\n匹配的指纹 (%d个):\n", len(matches))
		for i, match := range matches {
			fmt.Printf("  %d. %s\n", i+1, match.Finger.Info.Name)
			fmt.Printf("     - 匹配结果: %v\n", match.Result)
			fmt.Printf("     - 指纹ID: %s\n", match.Finger.Id)
			if match.Finger.Info.Description != "" {
				fmt.Printf("     - 描述: %s\n", match.Finger.Info.Description)
			}
			fmt.Println()
		}
	} else {
		fmt.Println("\n未匹配到任何指纹")
	}

	fmt.Println("\n扫描完成")
}
