/*
  - Package gxx
    @Author: zhizhuo
    @IDE：GoLand
    @File: main.go
    @Date: 2025/3/10 下午2:11*
*/
package gxx

import (
	"fmt"
	"gxx/pkg/runner"
	"gxx/pkg/wappalyzer"
	"gxx/types"
	"net/http"
	"os"
)

type BaseInfoType struct {
	Target     string
	Title      string
	ServerInfo *types.ServerInfo
	StatusCode int32
	Response   *http.Response
	Wappalyzer *wappalyzer.TypeWappalyzer
}
type CmdOptions = types.CmdOptions
type TargetResult = runner.TargetResult
type FingerMatch = runner.FingerMatch

// NewFingerOptions 创建新的指纹扫描选项
// 返回:
//   - types.YamlFingerType: 指纹配置选项
//   - error: 创建过程中的错误信息
func NewFingerOptions() (types.YamlFingerType, error) {
	// 默认在当前目录下查找finger_demo.yml文件
	defaultPath := "finger_demo.yml"
	
	// 检查文件是否存在
	_, err := os.Stat(defaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在则返回空对象，由调用者决定如何处理
			return types.YamlFingerType{}, nil
		}
		return types.YamlFingerType{}, fmt.Errorf("检查默认指纹文件失败: %w", err)
	}
	
	// 文件存在则使用默认文件
	return types.YamlFingerType{
		PocFile: defaultPath,
		PocYaml: "",
	}, nil
}

// InitFingerRules 初始化指纹规则，必须在调用ProcessURL前执行
// 参数:
//   - options: 指纹配置选项，包含指纹文件路径
//
// 返回:
//   - error: 初始化过程中的错误信息
//
// 注意: 该函数必须在调用FingerScan前执行一次
func InitFingerRules(options types.YamlFingerType) error {
	if options.PocFile == "" && options.PocYaml == "" {
		return fmt.Errorf("指纹文件路径(PocFile)和指纹内容(PocYaml)不能同时为空")
	}
	
	return runner.LoadFingerprints(options)
}

// FingerScan 处理单个URL的指纹识别，返回目标结果
// 参数:
//   - target: 目标URL
//   - proxy: HTTP代理地址 (可为空)
//   - timeout: 超时时间(秒)
//   - workerCount: 指纹规则并发线程数，用于控制指纹匹配速度
//
// 返回:
//   - *pkg.TargetResult: 识别结果
//   - error: 错误信息
func FingerScan(target string, proxy string, timeout int, workerCount int) (*runner.TargetResult, error) {
	if target == "" {
		return nil, fmt.Errorf("目标URL不能为空")
	}
	
	if timeout <= 0 {
		timeout = 5 // 设置默认超时时间
	}
	
	if workerCount <= 0 {
		workerCount = 1000 // 设置默认并发数
	}
	
	result, err := runner.ProcessURL(target, proxy, timeout, workerCount)
	if err != nil {
		return nil, fmt.Errorf("处理URL %s 时发生错误: %w", target, err)
	}
	
	if result == nil {
		return nil, fmt.Errorf("扫描目标 %s 返回空结果", target)
	}
	
	return result, nil
}

// GetFingerMatches 获取目标URL的所有匹配的指纹
// 参数:
//   - targetResult: 指纹扫描结果
//
// 返回:
//   - []*runner.FingerMatch: 指纹匹配结果数组，包含指纹信息和匹配结果
//   - 如果传入的targetResult为nil，则返回nil
func GetFingerMatches(targetResult *runner.TargetResult) []*runner.FingerMatch {
	if targetResult == nil {
		return nil
	}
	
	if targetResult.Matches == nil {
		return make([]*runner.FingerMatch, 0) // 返回空数组而不是nil
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
//   - *BaseInfoType: 包含基本信息的结构体
//   - error: 错误信息
func GetBaseInfo(target, proxy string, timeout int) (*BaseInfoType, error) {
	var BaseInfo BaseInfoType
	Bas, err := runner.GetBaseInfo(target, proxy, timeout)
	if err != nil {
		return nil, err
	}
	
	if Bas == nil {
		return nil, fmt.Errorf("获取目标 %s 的基础信息失败", target)
	}
	
	BaseInfo.Target = Bas.Url
	BaseInfo.Title = Bas.Title
	BaseInfo.ServerInfo = Bas.Server
	BaseInfo.StatusCode = Bas.StatusCode
	BaseInfo.Response = Bas.Response
	BaseInfo.Wappalyzer = Bas.Wappalyzer

	return &BaseInfo, nil
}

// WappalyzerScan 单独使用Wappalyzer分析目标URL的技术栈
// 参数:
//   - target: 目标URL
//   - proxy: HTTP代理地址 (可为空)
//   - timeout: 超时时间(秒)
//
// 返回:
//   - *wappalyzer.TypeWappalyzer: 技术栈分析结果
//   - error: 错误信息
func WappalyzerScan(target, proxy string, timeout int) (*wappalyzer.TypeWappalyzer, error) {
	// 调用GetBaseInfo获取Wappalyzer分析结果
	baseInfo, err := GetBaseInfo(target, proxy, timeout)
	if err != nil {
		return nil, err
	}

	if baseInfo.Wappalyzer == nil {
		return nil, fmt.Errorf("无法获取目标 %s 的技术栈信息", target)
	}

	return baseInfo.Wappalyzer, nil
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
	workerCount := 10000 // 规则并发线程数，可设置较高的值提高识别速度

	result, err := gxx.FingerScan(target, proxy, timeout, workerCount)
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

	// 6. 获取技术栈信息
	baseInfo, err := gxx.GetBaseInfo(target, proxy, timeout)
	if err != nil {
		log.Printf("获取基本信息错误: %v", err)
		return
	}

	if baseInfo.Wappalyzer != nil {
		fmt.Println("\n技术栈信息:")
		if len(baseInfo.Wappalyzer.WebServers) > 0 {
			fmt.Printf("  Web服务器: %v\n", baseInfo.Wappalyzer.WebServers)
		}
		if len(baseInfo.Wappalyzer.ProgrammingLanguages) > 0 {
			fmt.Printf("  编程语言: %v\n", baseInfo.Wappalyzer.ProgrammingLanguages)
		}
		if len(baseInfo.Wappalyzer.WebFrameworks) > 0 {
			fmt.Printf("  Web框架: %v\n", baseInfo.Wappalyzer.WebFrameworks)
		}
	}

	// 7. 单独进行技术栈分析
	wappResult, err := gxx.WappalyzerScan(target, proxy, timeout)
	if err != nil {
		log.Printf("技术栈分析错误: %v", err)
		return
	}

	fmt.Println("\n单独技术栈分析结果:")
	if len(wappResult.WebServers) > 0 {
		fmt.Printf("  Web服务器: %v\n", wappResult.WebServers)
	}
}
*/
