package runner

import (
	"fmt"
	"gxx/types"
	"gxx/utils/common"
	"gxx/utils/logger"
	"gxx/utils/output"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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
	// 确保目标不为空
	if target == "" {
		return nil, fmt.Errorf("目标URL不能为空")
	}

	// 创建目标结果对象，提前预分配
	targetResult := &TargetResult{
		URL:        target,
		StatusCode: 0,
		Title:      "",
		Server:     types.EmptyServerInfo(),
		Matches:    make([]*FingerMatch, 0, 10), // 预分配容量
		Wappalyzer: nil,
	}

	// 获取目标基础信息
	baseInfoResp, err := GetBaseInfo(target, proxy, timeout)

	// 即使获取基础信息失败，也继续处理
	if err != nil {
		logger.Debug(fmt.Sprintf("获取目标 %s 基础信息失败: %v", target, err))
		return targetResult, nil
	}

	// 更新目标结果对象
	targetResult.StatusCode = baseInfoResp.StatusCode
	targetResult.Title = baseInfoResp.Title
	targetResult.Server = baseInfoResp.Server
	targetResult.Wappalyzer = baseInfoResp.Wappalyzer
	targetResult.URL = baseInfoResp.Url
	logger.Debug(fmt.Sprintf("初始URL：%s", targetResult.URL))

	// 初始化缓存和变量映射
	var variableMap = make(map[string]any, 4) // 预分配map容量
	lastResponse, lastRequest := initializeCache(baseInfoResp.Response, proxy)
	if lastResponse == nil {
		// 如果无法获取响应，直接返回
		return targetResult, nil
	}

	variableMap["request"] = lastRequest
	variableMap["response"] = lastResponse

	targetResult.LastRequest = lastRequest
	targetResult.LastResponse = lastResponse

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
	matches := runFingerDetection(baseInfoResp.Url, baseInfo, proxy, timeout)
	targetResult.Matches = matches

	// 指纹规则运行完成之后立即删除缓存，减少内存压力
	ClearTargetURLCache(target)

	return targetResult, nil
}

// runFingerDetection 执行指纹识别，使用全局规则池高效处理指纹识别任务
func runFingerDetection(target string, baseInfo *BaseInfo, proxy string, timeout int) []*FingerMatch {
	// 确保全局规则池已初始化
	if GlobalRulePool == nil {
		logger.Error("全局规则池未初始化")
		return []*FingerMatch{}
	}

	// 如果没有指纹规则，直接返回
	if len(AllFinger) == 0 {
		return []*FingerMatch{}
	}

	// 创建更大的缓冲通道收集匹配结果，减少阻塞
	resultChan := make(chan *FingerMatch, len(AllFinger)*2)

	// 创建等待组
	var wg sync.WaitGroup

	// 记录开始时间用于性能监控
	startTime := time.Now()

	// 统计实际提交的任务数
	submittedTasks := int64(0)

	// 提交所有指纹任务到全局规则池
	for _, fingerprint := range AllFinger {
		wg.Add(1)

		// 构造规则任务
		task := &RuleTask{
			Target:     target,
			Finger:     fingerprint,
			BaseInfo:   baseInfo,
			Proxy:      proxy,
			Timeout:    timeout,
			ResultChan: resultChan,
			WaitGroup:  &wg,
		}

		// 简化重试机制，只在池满时重试一次
		submitErr := GlobalRulePool.Invoke(task)
		if submitErr != nil {
			// 如果提交失败，简单重试一次
			time.Sleep(1 * time.Millisecond)
			submitErr = GlobalRulePool.Invoke(task)
		}

		if submitErr != nil {
			logger.Debug(fmt.Sprintf("提交指纹任务失败: %s, 错误: %v", fingerprint.Id, submitErr))
			wg.Done()
			continue
		}

		// 统计成功提交的任务
		submittedTasks++
	}

	// 更新全局任务统计
	atomic.AddInt64(&poolStats.TotalTasks, submittedTasks)

	// 启动结果收集协程，避免阻塞主流程
	matches := make([]*FingerMatch, 0, len(AllFinger)/4) // 预分配容量
	var matchesMutex sync.Mutex
	resultDone := make(chan struct{})

	go func() {
		defer close(resultDone)
		for result := range resultChan {
			if result != nil && result.Result {
				matchesMutex.Lock()
				matches = append(matches, result)
				matchesMutex.Unlock()
			}
		}
	}()

	// 等待所有指纹任务完成
	wg.Wait()
	close(resultChan)

	// 等待结果收集完成
	<-resultDone

	// 记录性能信息
	duration := time.Since(startTime)
	logger.Debug(fmt.Sprintf("目标 %s 指纹识别完成，耗时: %v, 匹配数量: %d/%d, 实际任务数: %d",
		target, duration, len(matches), len(AllFinger), submittedTasks))

	return matches
}

// handleMatchResults 处理匹配结果，将结果输出到终端和文件
func handleMatchResults(targetResult *TargetResult, options *types.CmdOptions, printResult func(string), outputFormat string) {
	output.HandleMatchResults(&output.TargetResult{
		URL:        targetResult.URL,
		StatusCode: targetResult.StatusCode,
		Title:      targetResult.Title,
		ServerInfo: targetResult.Server,
		Matches:    convertFingerMatches(targetResult.Matches),
		Wappalyzer: targetResult.Wappalyzer,
	}, options.Output, options.SockOutput, printResult, outputFormat, targetResult.LastResponse)
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
