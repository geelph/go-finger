package runner

import (
	"fmt"
	"gxx/pkg/cel"
	"gxx/pkg/finger"
	"gxx/types"
	"gxx/utils/common"
	"gxx/utils/logger"
	"gxx/utils/output"
	"os"
	"strings"
	"sync"

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

	// 初始化缓存
	lastResponse, lastRequest := initializeCache(baseInfoResp.Response, proxy)
	targetResult.LastResponse = lastResponse
	targetResult.LastRequest = lastRequest

	// 如果无法获取响应，直接返回
	if lastResponse == nil {
		return targetResult, nil
	}

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

	// 创建同步等待组
	var fingerWg sync.WaitGroup

	// 预先创建并复用CustomLib实例，使用sync.Pool提高性能
	customLibPool := &sync.Pool{
		New: func() interface{} {
			return cel.NewCustomLib()
		},
	}

	// 创建指纹工作池
	fingerPool, _ := ants.NewPoolWithFunc(workerCount, func(data interface{}) {
		defer fingerWg.Done()

		// 获取数据
		task := data.(struct {
			fg *finger.Finger
		})
		fg := task.fg

		// 执行指纹识别 - 直接传递整个pool而不是单个实例
		result, err := evaluateFingerprintWithCache(fg, target, baseInfo, proxy, customLibPool, timeout, targetResult)
		if err == nil && result.Result {
			// 创建匹配结果对象
			resultMatch := &FingerMatch{
				Finger:   fg,
				Result:   true,
				Request:  result.Request,
				Response: result.Response,
			}

			// 添加到匹配结果列表
			matchesMutex.Lock()
			matches = append(matches, resultMatch)
			matchesMutex.Unlock()
		}
	}, ants.WithPreAlloc(true))
	defer fingerPool.Release()

	// 提交所有指纹任务到工作池
	for _, fg := range AllFinger {
		fingerWg.Add(1)

		// 提交任务
		_ = fingerPool.Invoke(struct {
			fg *finger.Finger
		}{fg})
	}

	// 等待所有指纹识别完成
	fingerWg.Wait()
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
