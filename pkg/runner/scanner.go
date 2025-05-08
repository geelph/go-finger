package runner

import (
	"fmt"
	finger2 "gxx/pkg/finger"
	"gxx/types"
	"gxx/utils/common"
	"gxx/utils/logger"
	"gxx/utils/output"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
)

// getTargets 从命令行参数或文件中读取目标，并进行去重处理
func getTargets(options *types.CmdOptions) []string {
	var targets []string

	// 直接指定的目标
	if len(options.Target) > 0 {
		targets = options.Target
	} else if options.TargetsFile != "" {
		// 从文件读取目标列表
		content, err := os.ReadFile(options.TargetsFile)
		if err != nil {
			logger.Error(fmt.Sprintf("读取目标文件失败: %v", err))
			return nil
		}

		// 按行分割，去除空行和空白
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				targets = append(targets, line)
			}
		}
	}

	// 去重处理
	originalCount := len(targets)
	targets = common.RemoveDuplicateURLs(targets)
	duplicateCount := originalCount - len(targets)

	logger.Info(fmt.Sprintf("原始目标数量：%v个，重复目标数量：%v个，去重后目标数量：%v个",
		originalCount, duplicateCount, len(targets)))

	return targets
}

// ProcessURL 处理单个URL的所有指纹识别，获取目标基础信息并执行指纹识别
func ProcessURL(target string, proxy string, timeout int, workerCount int) (*TargetResult, error) {
	// 获取目标基础信息
	baseInfoResp, err := GetBaseInfo(target, proxy, timeout)

	// 创建目标结果对象
	targetResult := &TargetResult{
		URL:        target,
		StatusCode: 0,
		Title:      "",
		Server:     types.EmptyServerInfo(),
		Matches:    make([]*FingerMatch, 0),
		Wappalyzer: nil,
	}

	// 即使获取基础信息失败，也继续处理
	if err != nil {
		logger.Debug(fmt.Sprintf("获取目标 %s 基础信息失败: %v", target, err))
		return targetResult, nil // 如果无法获取状态码，直接返回
	}

	// 更新目标结果对象
	targetResult.StatusCode = baseInfoResp.StatusCode
	targetResult.Title = baseInfoResp.Title
	targetResult.Server = baseInfoResp.Server
	targetResult.Wappalyzer = baseInfoResp.Wappalyzer
	targetResult.URL = baseInfoResp.Url
	logger.Debug(fmt.Sprintf("初始URL：%s", targetResult.URL))
	// 初始化缓存和变量映射
	var variableMap = make(map[string]any)
	lastResponse, lastRequest := initializeCache(baseInfoResp.Response, proxy)
	if lastResponse == nil {
		// 如果无法获取响应，直接返回
		return targetResult, nil
	}

	variableMap["request"] = lastRequest
	variableMap["response"] = lastResponse

	// 使用互斥锁保护共享资源访问，一次性更新
	targetResult.mutex.Lock()
	targetResult.LastRequest = lastRequest
	targetResult.LastResponse = lastResponse
	targetResult.mutex.Unlock()

	UpdateTargetCache(variableMap, targetResult.URL, false)

	// 创建基础信息对象
	baseInfo := &BaseInfo{
		Title:      targetResult.Title,
		Server:     targetResult.Server,
		StatusCode: targetResult.StatusCode,
	}

	// 如果没有指纹规则，直接返回结果
	if len(AllFinger) == 0 {
		return targetResult, nil
	}

	// 执行指纹识别
	matches := runFingerDetection(baseInfoResp.Url, baseInfo, proxy, timeout, workerCount, targetResult)
	targetResult.Matches = matches

	return targetResult, nil
}

// runFingerDetection 执行指纹识别，使用高性能池模式处理多个指纹的识别
func runFingerDetection(target string, baseInfo *BaseInfo, proxy string, timeout int, workerCount int, targetResult *TargetResult) []*FingerMatch {
	// 线程安全地存储匹配结果
	var matches []*FingerMatch
	var matchesMutex sync.Mutex

	// 创建同步等待组，用于等待所有任务完成
	var wg sync.WaitGroup

	// 定义任务处理函数
	type fingerTask struct {
		fg *finger2.Finger
	}

	// 创建ants线程池，使用NewPoolWithFunc预定义处理函数
	fingerPool, _ := ants.NewPoolWithFunc(workerCount, func(i interface{}) {
		defer wg.Done()
		task := i.(fingerTask)
		fingerFg := task.fg
		tar := targetResult
		// 执行指纹识别
		result, err := evaluateFingerprintWithCache(fingerFg, target, baseInfo, proxy, timeout, tar)

		if err == nil && result.Result {
			// 创建匹配结果对象
			resultMatch := &FingerMatch{
				Finger:   fingerFg,
				Result:   true,
				Request:  result.Request,
				Response: result.Response,
			}

			// 添加到匹配结果列表
			matchesMutex.Lock()
			matches = append(matches, resultMatch)
			matchesMutex.Unlock()
		}
	},
		ants.WithPreAlloc(true),
		ants.WithExpiryDuration(3*time.Minute),
		ants.WithNonblocking(false), // 使用阻塞模式确保任务按需执行
	)
	defer fingerPool.Release()

	// 提交所有指纹任务到线程池
	for _, fg := range AllFinger {
		wg.Add(1)
		// 提交任务到线程池
		_ = fingerPool.Invoke(fingerTask{fg: fg})
	}

	// 等待所有指纹识别完成
	wg.Wait()
	return matches
}

// handleMatchResults 处理匹配结果，将结果输出到终端和文件
func handleMatchResults(targetResult *TargetResult, options *types.CmdOptions, printResult func(string), outputFormat string) {
	// 使用互斥锁保护LastResponse的访问
	targetResult.mutex.Lock()
	lastResponse := targetResult.LastResponse
	targetResult.mutex.Unlock()

	output.HandleMatchResults(&output.TargetResult{
		URL:        targetResult.URL,
		StatusCode: targetResult.StatusCode,
		Title:      targetResult.Title,
		ServerInfo: targetResult.Server,
		Matches:    convertFingerMatches(targetResult.Matches),
		Wappalyzer: targetResult.Wappalyzer,
	}, options.Output, options.SockOutput, printResult, outputFormat, lastResponse)
}

// convertFingerMatches 将pkg.FingerMatch切片转换为output.FingerMatch切片
func convertFingerMatches(matches []*FingerMatch) []*output.FingerMatch {
	result := make([]*output.FingerMatch, len(matches))
	for i, match := range matches {
		result[i] = &output.FingerMatch{
			Finger:   match.Finger,
			Result:   match.Result,
			Request:  match.Request,
			Response: match.Response,
		}
	}
	return result
}

// printSummary 打印汇总信息
func printSummary(targets []string, results map[string]*TargetResult) {
	// 将pkg.TargetResult映射转换为output.TargetResult映射
	outputResults := make(map[string]*output.TargetResult)
	for key, result := range results {
		outputResults[key] = &output.TargetResult{
			URL:        result.URL,
			StatusCode: result.StatusCode,
			Title:      result.Title,
			ServerInfo: result.Server,
			Matches:    convertFingerMatches(result.Matches),
			Wappalyzer: result.Wappalyzer,
		}
	}
	output.PrintSummary(targets, outputResults)
}
