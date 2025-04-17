/*
  - Package gxx
    @Author: zhizhuo
    @IDE：GoLand
    @File: main.go
    @Date: 2025/3/10 下午2:11*
*/
package gxx

import (
	"gxx/pkg"
	"gxx/pkg/wappalyzer"
	"gxx/types"
	"net/http"
)

// NewFingerOptions 创建新的指纹扫描选项
func NewFingerOptions() (types.YamlFingerType, error) {
	return types.YamlFingerType{}, nil
}

// InitFingerRules 初始化指纹规则，必须在调用ProcessURL前执行
func InitFingerRules(options types.YamlFingerType) error {
	return pkg.LoadFingerprints(options)
}

// FingerScan 处理单个URL的指纹识别，返回目标结果
// 参数:
//   - target: 目标URL
//   - proxy: HTTP代理地址 (可为空)
//   - timeout: 超时时间(秒)
//   - workerCount: 工作协程数量
//
// 返回:
//   - *pkg.TargetResult: 识别结果
//   - error: 错误信息
func FingerScan(target string, proxy string, timeout int, workerCount int) (*pkg.TargetResult, error) {
	return pkg.ProcessURL(target, proxy, timeout, workerCount)
}

// GetFingerMatches 获取目标URL的所有匹配的指纹
// 返回FingerMatch数组，包含指纹信息和匹配结果
func GetFingerMatches(targetResult *pkg.TargetResult) []*pkg.FingerMatch {
	if targetResult == nil {
		return nil
	}
	return targetResult.Matches
}

// GetBaseInfo 获取目标URL的基础信息（标题、服务器信息和状态码）
// 参数:
//   - target: 目标URL
//   - proxy: HTTP代理地址 (可为空)
//   - timeout: 超时时间(秒)
//
// 返回:
//   - title: 站点标题
//   - serverInfo: 服务器信息
//   - statusCode: HTTP状态码
//   - response: HTTP原始响应对象
//   - error: 错误信息
func GetBaseInfo(target, proxy string, timeout int) (string, *types.ServerInfo, int32, *http.Response, *wappalyzer.TypeWappalyzer, error) {
	return pkg.GetBaseInfo(target, proxy, timeout)
}

// 以下是API使用示例

/*
示例: 如何使用gxx库进行指纹识别

package main

import (
	"fmt"
	"gxx"
	"log"
)

func main() {
	// 1. 创建配置选项
	options, err := gxx.NewFingerOptions()
	if err != nil {
		log.Fatalf("创建选项错误: %v", err)
	}

	// 2. 初始化指纹规则库（仅需执行一次）
	if err := gxx.InitFingerRules(options); err != nil {
		log.Fatalf("初始化指纹规则错误: %v", err)
	}

	// 3. 处理单个URL
	target := "https://example.com"
	proxy := "" // 如果不需要代理，设为空字符串
	timeout := 5 // 超时时间，单位：秒
	workerCount := 10 // 并发工作线程数

	result, err := gxx.ProcessURL(target, proxy, timeout, workerCount)
	if err != nil {
		log.Printf("处理URL错误: %v", err)
		return
	}

	// 4. 输出基本信息
	fmt.Printf("URL: %s\n", result.URL)
	fmt.Printf("状态码: %d\n", result.StatusCode)
	fmt.Printf("标题: %s\n", result.Title)
	if result.Server != nil {
		fmt.Printf("服务器: %s\n", result.Server.ServerType)
	}

	// 5. 处理匹配结果
	matches := gxx.GetFingerMatches(result)
	if len(matches) > 0 {
		fmt.Println("\n匹配的指纹:")
		for i, match := range matches {
			fmt.Printf("  %d. %s (匹配结果: %v)\n", i+1, match.Finger.Info.Name, match.Result)
		}
	} else {
		fmt.Println("\n未匹配到任何指纹")
	}
}
*/
