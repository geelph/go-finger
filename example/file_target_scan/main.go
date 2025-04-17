/*
 * Package main
 * 文件目标扫描示例：演示如何从文件中读取目标列表进行批量扫描
 * @Author: zhizhuo
 * @IDE：GoLand
 * @File: main.go
 * @Date: 2025/3/10 下午2:11
 */
package main

import (
	"bufio"
	"fmt"
	"gxx"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

	// 示例文件路径 - 自动创建示例文件
	exampleDir, _ := os.Getwd()
	targetsFile := filepath.Join(exampleDir, "targets.txt")

	// 创建示例目标文件
	createExampleTargetFile(targetsFile)

	// 设置超时和线程
	timeout := 5
	workerCount := 10
	proxy := ""

	// 执行扫描
	fmt.Printf("开始从文件扫描目标: %s\n", targetsFile)
	fmt.Println("--------------------------------------------")

	// 初始化指纹规则库
	if err := gxx.InitFingerRules(options); err != nil {
		fmt.Printf("初始化指纹规则库失败: %v\n", err)
		os.Exit(1)
	}

	// 读取目标文件
	targets, err := readTargetsFromFile(targetsFile)
	if err != nil {
		fmt.Printf("读取目标文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("从文件中读取到 %d 个目标\n", len(targets))
	fmt.Println("开始并发扫描...")

	// 使用并发执行扫描
	results := scanTargetsWithConcurrency(targets, proxy, timeout, workerCount)

	// 输出结果
	fmt.Println("\n扫描结果:")
	fmt.Printf("总共扫描: %d 个目标, 成功: %d 个\n",
		len(targets), len(results))

	// 输出每个目标的扫描结果
	fmt.Println("\n详细结果:")
	for target, result := range results {
		fmt.Printf("\n目标: %s\n", target)
		fmt.Printf("  - 状态码: %d\n", result.StatusCode)
		fmt.Printf("  - 标题: %s\n", result.Title)
		if result.Server != nil {
			fmt.Printf("  - 服务器: %s\n", result.Server.ServerType)
		}

		// 输出匹配的指纹
		matches := gxx.GetFingerMatches(result)
		if len(matches) > 0 {
			fmt.Printf("  - 匹配指纹: %d 个\n", len(matches))
			for i, match := range matches[:min(3, len(matches))] {
				fmt.Printf("    %d. %s\n", i+1, match.Finger.Info.Name)
			}
			if len(matches) > 3 {
				fmt.Printf("    ...（还有 %d 个）\n", len(matches)-3)
			}
		} else {
			fmt.Printf("  - 匹配指纹: 无\n")
		}
	}

	// 将结果保存到文件
	outputFile := filepath.Join(exampleDir, "scan_results.txt")
	if err := saveResultsToFile(outputFile, results); err != nil {
		fmt.Printf("保存结果失败: %v\n", err)
	} else {
		fmt.Printf("\n结果已保存到: %s\n", outputFile)
	}

	fmt.Println("--------------------------------------------")
	fmt.Printf("总耗时: %s\n", time.Since(startTime))
	fmt.Println("扫描完成")
}

// createExampleTargetFile 创建示例目标文件
func createExampleTargetFile(filePath string) {
	// 检查文件是否已存在
	if _, err := os.Stat(filePath); err == nil {
		fmt.Println("目标文件已存在:", filePath)
		return
	}

	// 创建示例目标
	targets := []string{
		"example.com",
		"github.com",
		"httpbin.org",
	}

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("创建目标文件失败: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// 写入目标
	for _, target := range targets {
		file.WriteString(target + "\n")
	}

	fmt.Println("已创建示例目标文件:", filePath)
	fmt.Println("包含以下目标:")
	for _, target := range targets {
		fmt.Println("  -", target)
	}
}

// readTargetsFromFile 从文件中读取目标列表
func readTargetsFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var targets []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		target := strings.TrimSpace(scanner.Text())
		if target != "" {
			targets = append(targets, target)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return targets, nil
}

// scanTargetsWithConcurrency 使用并发扫描多个目标
func scanTargetsWithConcurrency(targets []string, proxy string, timeout, workerCount int) map[string]*gxx.TargetResult {
	results := make(map[string]*gxx.TargetResult)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	// 创建工作通道
	targetCh := make(chan string, len(targets))
	for _, target := range targets {
		targetCh <- target
	}
	close(targetCh)

	// 启动工作协程
	for i := 0; i < min(workerCount, len(targets)); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for target := range targetCh {
				// 扫描目标
				fmt.Printf("正在扫描: %s\n", target)
				result, err := gxx.FingerScan(target, proxy, timeout, 1)
				if err != nil {
					fmt.Printf("扫描失败 [%s]: %v\n", target, err)
					continue
				}

				// 存储结果
				mutex.Lock()
				results[target] = result
				mutex.Unlock()
			}
		}()
	}

	// 等待所有任务完成
	wg.Wait()
	return results
}

// saveResultsToFile 将结果保存到文件
func saveResultsToFile(filePath string, results map[string]*gxx.TargetResult) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入标题
	file.WriteString("目标,状态码,标题,服务器,匹配指纹数量,指纹列表\n")

	// 写入每个目标的结果
	for target, result := range results {
		// 获取匹配的指纹
		matches := gxx.GetFingerMatches(result)
		fingerprints := make([]string, 0, len(matches))
		for _, match := range matches {
			fingerprints = append(fingerprints, match.Finger.Info.Name)
		}

		// 组装服务器信息
		serverInfo := ""
		if result.Server != nil {
			serverInfo = result.Server.ServerType
		}

		// 写入一行数据
		line := fmt.Sprintf("%s,%d,%s,%s,%d,%s\n",
			target,
			result.StatusCode,
			escapeCSV(result.Title),
			escapeCSV(serverInfo),
			len(matches),
			escapeCSV(strings.Join(fingerprints, "; ")))

		file.WriteString(line)
	}

	return nil
}

// escapeCSV 转义CSV字段中的特殊字符
func escapeCSV(s string) string {
	if strings.Contains(s, ",") || strings.Contains(s, "\"") || strings.Contains(s, "\n") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
