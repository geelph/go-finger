/*
  - Package utils
    @Author: zhizhuo
    @IDE：GoLand
    @File: runner.go
    @Date: 2025/3/10 下午2:11*
*/
package pkg

import (
	"context"
	"fmt"
	"gxx/pkg/cel"
	finger2 "gxx/pkg/finger"
	"gxx/pkg/network"
	"gxx/types"
	"gxx/utils"
	"gxx/utils/common"
	"gxx/utils/logger"
	"gxx/utils/output"
	"gxx/utils/proto"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

var AllFinger []*finger2.Finger

// SendRequest获取的最后一个请求响应
var lastResponse *proto.Response
var lastRequest *proto.Request

// TargetResult 存储每个目标的扫描结果
type TargetResult struct {
	URL        string
	StatusCode int32
	Title      string
	Server     *types.ServerInfo
	Matches    []*FingerMatch
}

// FingerMatch 存储每个匹配的指纹信息
type FingerMatch struct {
	Finger *finger2.Finger
	Result bool
}

// BaseInfo 存储目标的基础信息
type BaseInfo struct {
	Title      string
	Server     *types.ServerInfo
	StatusCode int32
}

// initializeCache 初始化请求响应缓存
func initializeCache(httpResp *http.Response, proxy string) *proto.Response {
	if httpResp == nil {
		return nil
	}

	// 读取响应体
	respBody, _ := io.ReadAll(httpResp.Body)
	_ = httpResp.Body.Close()
	utf8RespBody := common.Str2UTF8(string(respBody))

	// 构建响应对象
	initialResponse := finger2.BuildProtoResponse(httpResp, utf8RespBody, 0, proxy)

	// 初始化请求缓存
	reqMethod := "GET"
	reqBody := ""
	reqPath := "/"
	lastRequest = finger2.BuildProtoRequest(httpResp, reqMethod, reqBody, reqPath)
	lastResponse = initialResponse

	return initialResponse
}

// loadFingerprints 加载指纹规则文件
func loadFingerprints(options *types.CmdOptions) error {
	var targetPath string
	// 使用嵌入式指纹库
	if options.PocFile == "" && options.PocYaml == "" {
		logger.Info("使用默认指纹库")
		fin, err := utils.GetFingerYaml()
		if err != nil {
			return err
		}
		AllFinger = fin
	}

	if options.PocFile != "" {
		targetPath = options.PocFile
		logger.Info(fmt.Sprintf("加载yaml文件目录：%s", targetPath))
		// 使用WalkDir递归遍历目录中的所有文件
		return filepath.WalkDir(targetPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && common.IsYamlFile(path) {
				if poc, err := finger2.Read(path); err == nil && poc != nil {
					AllFinger = append(AllFinger, poc)
				}
			}
			return nil
		})

	} else if options.PocYaml != "" {
		targetPath = options.PocYaml
		logger.Info(fmt.Sprintf("加载yaml文件：%s", targetPath))
		// 直接读取单个文件
		if common.IsYamlFile(targetPath) {
			if poc, err := finger2.Read(targetPath); err == nil && poc != nil {
				AllFinger = append(AllFinger, poc)
			} else if err != nil {
				return fmt.Errorf("读取yaml文件出错: %v", err)
			}
		} else {
			return fmt.Errorf("%s 不是有效的yaml文件", targetPath)
		}
	}
	return nil
}

// prepareRequest 准备HTTP请求
func prepareRequest(target string) (*http.Request, error) {
	urlWithProtocol := target
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		urlWithProtocol = "https://" + target
	}

	req, err := http.NewRequest("GET", urlWithProtocol, nil)
	if err != nil {
		return nil, fmt.Errorf("创建临时请求失败: %v", err)
	}
	return req, nil
}

// GetBaseInfo 获取目标的基础信息（标题和Server信息）并返回完整HTTP响应
func GetBaseInfo(target, proxy string, timeout int) (string, *types.ServerInfo, int32, *http.Response, error) {
	// 准备URL
	urlWithProtocol := target
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		urlWithProtocol = "https://" + target
	}

	// 检查并规范化URL协议
	checkedURL, err := network.CheckProtocol(urlWithProtocol)
	if err == nil && checkedURL != "" {
		urlWithProtocol = checkedURL
	}

	// 创建请求选项
	timeoutDuration := time.Duration(timeout) * time.Second
	if timeout <= 0 {
		timeoutDuration = 5 * time.Second // 使用默认3秒作为超时时间
	}

	options := network.OptionsRequest{
		Proxy:              proxy,
		Timeout:            timeoutDuration,
		Retries:            2,
		FollowRedirects:    true,
		InsecureSkipVerify: true,
		CustomHeaders: map[string]string{
			"User-Agent":      common.RandomUA(),
			"X-Forwarded-For": common.GetRandomIP(),
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
			"Connection":      "close",
		},
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	// 发送请求
	resp, err := network.SendRequestHttp(ctx, "GET", urlWithProtocol, "", options)
	if err != nil {
		return "", nil, 0, nil, fmt.Errorf("发送请求失败: %v", err)
	}

	// 注意：不要关闭resp.Body，让调用方负责关闭

	// 获取状态码
	statusCode := int32(resp.StatusCode)

	// 使用finger2包的GetTitle方法提取标题
	title := finger2.GetTitle(urlWithProtocol, resp)

	// 使用finger2包的GetServerInfoFromResponse方法提取Server信息
	serverInfo := finger2.GetServerInfoFromResponse(resp)

	return title, serverInfo, statusCode, resp, nil
}

// NewFingerRunner 创建并运行指纹识别器
func NewFingerRunner(options *types.CmdOptions) {
	// 处理目标URL列表
	targets := getTargets(options)
	if len(targets) == 0 {
		logger.Error("未找到有效的目标URL")
		return
	}

	// 加载指纹规则
	if err := loadFingerprints(options); err != nil {
		logger.Error(fmt.Sprintf("加载指纹规则出错: %v", err))
		return
	}
	logger.Info(fmt.Sprintf("加载指纹数量：%v个", len(AllFinger)))
	// 确定输出格式并初始化输出文件
	outputFormat := getOutputFormat(options)
	if options.Output != "" {
		if err := output.InitOutput(options.Output, outputFormat); err != nil {
			logger.Error(fmt.Sprintf("初始化输出文件失败: %v", err))
			return
		}
		defer func() {
			_ = output.Close()
		}()
	}

	// 设置并发参数
	urlWorkerCount := options.Threads
	if urlWorkerCount <= 0 {
		urlWorkerCount = 10 // 默认10个线程
	}
	fingerWorkerCount := 5 * urlWorkerCount

	logger.Info(fmt.Sprintf("开始扫描 %d 个目标，使用 %d 个并发线程...", len(targets), urlWorkerCount))

	// 执行扫描
	results := runScan(targets, options, urlWorkerCount, fingerWorkerCount, outputFormat)

	// 输出统计信息
	printSummary(targets, results)
}

// runScan 执行扫描过程
func runScan(targets []string, options *types.CmdOptions, urlWorkerCount, fingerWorkerCount int, outputFormat string) map[string]*TargetResult {
	results := make(map[string]*TargetResult)
	var resultsMutex sync.Mutex
	var outputMutex sync.Mutex

	// 创建URL任务通道和完成通道
	urlChan := make(chan string, len(targets))
	doneChan := make(chan struct{}, len(targets))
	var urlWg sync.WaitGroup

	// 创建进度条
	bar := createProgressBar(len(targets))

	// 启动进度条更新协程
	startTime := time.Now()
	go func() {
		for range doneChan {
			outputMutex.Lock()
			_ = bar.Add(1)
			outputMutex.Unlock()
		}
	}()

	// 存储输出的结果
	saveResult := func(msg string) {
		outputMutex.Lock()
		defer outputMutex.Unlock()

		// 暂时清除进度条并输出结果
		fmt.Print("\033[2K\r")
		fmt.Println(msg)

		// 重新显示进度条
		_ = bar.RenderBlank()
	}

	// 启动URL工作协程
	for i := 0; i < urlWorkerCount; i++ {
		urlWg.Add(1)
		go func() {
			defer urlWg.Done()

			for target := range urlChan {
				// 处理单个URL
				targetResult, _ := processURL(target, options.Proxy, options.Timeout, fingerWorkerCount, options, saveResult, outputFormat)

				// 存储结果
				resultsMutex.Lock()
				results[target] = targetResult
				resultsMutex.Unlock()

				// 通知完成一个任务
				doneChan <- struct{}{}
			}
		}()
	}

	// 发送URL任务
	for _, target := range targets {
		urlChan <- target
	}
	close(urlChan)

	// 等待所有URL处理完成
	urlWg.Wait()
	close(doneChan)

	// 确保最终完成100%进度
	outputMutex.Lock()
	_ = bar.Finish()
	outputMutex.Unlock()

	// 显示扫描耗时信息
	elapsedTime := time.Since(startTime)
	itemsPerSecond := float64(len(targets)) / elapsedTime.Seconds()

	maxProgress := fmt.Sprintf("指纹识别 100%% [==================================================] (%d/%d, %.2f it/s)",
		len(targets), len(targets), itemsPerSecond)
	fmt.Println(maxProgress)

	return results
}

// getTargets 获取所有目标URL
func getTargets(options *types.CmdOptions) []string {
	var targets []string

	// 直接指定的目标
	if len(options.Target) > 0 {
		// 将goflags.StringSlice转换为[]string
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

// createProgressBar 创建进度条
func createProgressBar(total int) *progressbar.ProgressBar {
	return progressbar.NewOptions64(
		int64(total),
		progressbar.OptionSetWidth(50),
		progressbar.OptionEnableColorCodes(false),
		progressbar.OptionShowBytes(false),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionSetDescription("指纹识别"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionClearOnFinish(),
	)
}

// getOutputFormat 确定输出格式
func getOutputFormat(options *types.CmdOptions) string {
	if options.Output == "" {
		return "txt" // 默认为txt格式
	}

	ext := strings.ToLower(filepath.Ext(options.Output))
	if ext == ".csv" {
		return "csv"
	}
	return "txt"
}

// printSummary 打印汇总信息
func printSummary(targets []string, results map[string]*TargetResult) {
	matchCount := 0
	noMatchCount := 0

	// 统计匹配成功和失败的数量
	for _, targetResult := range results {
		if len(targetResult.Matches) > 0 {
			matchCount++
		} else {
			noMatchCount++
			// 只输出未匹配的URL信息
			printNoMatchResult(targetResult)
		}
	}

	// 输出统计信息
	fmt.Println(color.CyanString("─────────────────────────────────────────────────────"))
	fmt.Printf("扫描统计: 目标总数 %d, 匹配成功 %d, 匹配失败 %d\n",
		len(targets), matchCount, noMatchCount)
}

// printNoMatchResult 打印未匹配结果
func printNoMatchResult(targetResult *TargetResult) {
	statusCodeStr := ""
	if targetResult.StatusCode > 0 {
		statusCodeStr = fmt.Sprintf("（%d）", targetResult.StatusCode)
	}

	serverInfo := ""
	if targetResult.Server != nil {
		serverInfo = fmt.Sprintf("%s", targetResult.Server.ServerType)
	}

	baseInfo := fmt.Sprintf("URL：%s %s  标题：%s  Server：%s",
		targetResult.URL, statusCodeStr, targetResult.Title, serverInfo)

	outputMsg := fmt.Sprintf("%s  匹配结果：%s", baseInfo, color.RedString("未匹配"))
	fmt.Println(outputMsg)
}

// processURL 处理单个URL的所有指纹识别
func processURL(target string, proxy string, timeout int, workerCount int, options *types.CmdOptions, printResult func(string), outputFormat string) (*TargetResult, error) {
	// 获取目标基础信息
	title, serverInfo, statusCode, httpResp, err := GetBaseInfo(target, proxy, timeout)

	// 创建目标结果对象
	targetResult := &TargetResult{
		URL:        target,
		StatusCode: statusCode,
		Title:      title,
		Server:     serverInfo,
		Matches:    make([]*FingerMatch, 0),
	}

	// 即使获取基础信息失败，也继续处理
	if err != nil {
		logger.Debug(fmt.Sprintf("获取目标 %s 基础信息失败: %v", target, err))
		targetResult.Title = ""
		targetResult.Server = types.EmptyServerInfo()
		targetResult.StatusCode = 0
	}

	// 初始化缓存
	initialResponse := initializeCache(httpResp, proxy)

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
	matches := runFingerDetection(target, baseInfo, proxy, timeout, workerCount)
	targetResult.Matches = matches

	handleMatchResults(targetResult, options, printResult, outputFormat, initialResponse)
	return targetResult, nil
}

// runFingerDetection 执行指纹识别
func runFingerDetection(target string, baseInfo *BaseInfo, proxy string, timeout int, workerCount int) []*FingerMatch {
	bufferSize := min(len(AllFinger), 1000)

	// 创建通道
	fingerChan := make(chan *finger2.Finger, bufferSize)
	resultChan := make(chan *FingerMatch, bufferSize)
	var fingerWg sync.WaitGroup

	// 预先创建并复用CustomLib实例
	customLibs := make([]*cel.CustomLib, workerCount)
	for i := 0; i < workerCount; i++ {
		customLibs[i] = cel.NewCustomLib()
	}

	// 启动工作协程
	for i := 0; i < workerCount; i++ {
		fingerWg.Add(1)
		go func(workerID int) {
			defer fingerWg.Done()
			customLib := customLibs[workerID]

			for fg := range fingerChan {
				customLib.Reset()

				// 执行指纹识别
				result, err := evaluateFingerprintWithCache(fg, target, baseInfo, proxy, customLib, timeout)
				if err == nil && result {
					// 只存储匹配成功的指纹
					select {
					case resultChan <- &FingerMatch{Finger: fg, Result: true}:
					default:
						// 通道已满，忽略结果
					}
				}
			}
		}(i)
	}

	// 发送指纹任务
	go func() {
		for _, fg := range AllFinger {
			fingerChan <- fg
		}
		close(fingerChan)
	}()

	// 收集结果
	var matches []*FingerMatch
	var matchesMutex sync.Mutex

	go func() {
		for match := range resultChan {
			matchesMutex.Lock()
			matches = append(matches, match)
			matchesMutex.Unlock()
		}
	}()

	// 等待所有指纹识别完成
	fingerWg.Wait()
	close(resultChan)

	// 等待结果收集完成
	time.Sleep(10 * time.Millisecond)

	return matches
}

// handleMatchResults 处理匹配结果
func handleMatchResults(targetResult *TargetResult, options *types.CmdOptions, printResult func(string), outputFormat string, initialResponse *proto.Response) {
	// 构建输出信息
	statusCodeStr := ""
	if targetResult.StatusCode > 0 {
		statusCodeStr = fmt.Sprintf("（%d）", targetResult.StatusCode)
	}

	serverInfo := ""
	if targetResult.Server != nil {
		serverInfo = fmt.Sprintf("%s", targetResult.Server.ServerType)
	}

	// 收集所有匹配的指纹名称
	fingerNames := make([]string, 0, len(targetResult.Matches))
	for _, match := range targetResult.Matches {
		fingerNames = append(fingerNames, match.Finger.Info.Name)
	}

	// 构建输出信息
	baseInfoStr := fmt.Sprintf("URL：%s %s  标题：%s  Server：%s",
		targetResult.URL, statusCodeStr, targetResult.Title, serverInfo)

	if len(targetResult.Matches) > 0 && targetResult.Matches[0].Result {
		outputMsg := fmt.Sprintf("%s  指纹：[%s]  匹配结果：%s",
			baseInfoStr, strings.Join(fingerNames, "，"), color.GreenString("成功"))
		// 输出结果
		printResult(outputMsg)
	}

	// 写入结果文件
	if options.Output != "" {
		writeResultToFile(targetResult, options.Output, outputFormat, initialResponse)
	}
}

// writeResultToFile 将结果写入文件
func writeResultToFile(targetResult *TargetResult, outputs, format string, initialResponse *proto.Response) {
	fingerList := make([]*finger2.Finger, 0, len(targetResult.Matches))
	for _, match := range targetResult.Matches {
		fingerList = append(fingerList, match.Finger)
	}

	// 创建写入选项结构体
	writeOpts := &output.WriteOptions{
		Output:      outputs,
		Format:      format,
		Target:      targetResult.URL,
		Fingers:     fingerList,
		StatusCode:  targetResult.StatusCode,
		Title:       targetResult.Title,
		ServerInfo:  targetResult.Server,
		FinalResult: true,
	}

	// 检查 initialResponse 是否为 nil，再设置 RespHeaders
	if initialResponse != nil {
		writeOpts.RespHeaders = string(initialResponse.RawHeader)
	}

	// 添加响应信息
	if lastResponse != nil {
		writeOpts.Response = lastResponse
	} else if initialResponse != nil {
		writeOpts.Response = initialResponse
	}
	// 写入结果
	if err := output.WriteFingerprints(writeOpts); err != nil {
		logger.Error(fmt.Sprintf("写入结果失败: %v", err))
	}
}

// evaluateFingerprintWithCache 使用缓存的基础信息评估指纹规则
func evaluateFingerprintWithCache(fg *finger2.Finger, target string, baseInfo *BaseInfo, proxy string, customLib *cel.CustomLib, timeout int) (bool, error) {
	// 初始化变量映射
	SetiableMap := make(map[string]any)
	logger.Debug(fmt.Sprintf("获取指纹ID：%s", fg.Id))

	// 准备基础请求
	req, err := prepareRequest(target)
	if err != nil {
		return false, err
	}

	tempReqData, err := network.ParseRequest(req)
	if err != nil {
		return false, fmt.Errorf("解析请求失败: %v", err)
	}

	// 设置基础变量
	SetiableMap["request"] = tempReqData
	// 设置缓存的基础信息
	SetiableMap["title"] = baseInfo.Title
	SetiableMap["server"] = baseInfo.Server

	// 确保SetiableMap中包含response字段，初始化为缓存的响应
	SetiableMap["response"] = &proto.Response{
		Status:      baseInfo.StatusCode,
		Headers:     map[string]string{},
		ContentType: "",
		Body:        []byte{},
		Raw:         []byte{},
		RawHeader:   []byte{},
		Url:         &proto.UrlType{},
		Latency:     0,
	}

	// 处理set规则
	if len(fg.Set) > 0 {
		finger2.IsFuzzSet(fg.Set, SetiableMap, customLib)
	}
	// 处理payload
	if len(fg.Payloads.Payloads) > 0 {
		finger2.IsFuzzSet(fg.Payloads.Payloads, SetiableMap, customLib)
	}

	// 评估规则
	for _, rule := range fg.Rules {
		// 检查是否可以使用缓存
		if shouldUseCache(rule) {
			SetiableMap["request"] = lastRequest
			SetiableMap["response"] = lastResponse
		} else {
			// 发送新请求
			newVarMap, err := finger2.SendRequest(target, rule.Value.Request, rule.Value, SetiableMap, proxy, timeout)
			if err != nil {
				customLib.WriteRuleFunctionsROptions(rule.Key, false)
				continue
			}

			if len(newVarMap) > 0 {
				SetiableMap = newVarMap
				updateCache(SetiableMap)
			}
		}
		logger.Debug(fmt.Sprintf("请求数据包：\n%s", SetiableMap["request"].(*proto.Request).Raw))
		logger.Debug(fmt.Sprintf("响应数据包：\n%s", SetiableMap["response"].(*proto.Response).Raw))
		logger.Debug("开始cel匹配处理")
		result, err := customLib.Evaluate(rule.Value.Expression, SetiableMap)
		if err != nil {
			logger.Debug(fmt.Sprintf("规则 %s CEL解析错误：%s", rule.Key, err.Error()))
			customLib.WriteRuleFunctionsROptions(rule.Key, false)
		} else {
			ruleBool := result.Value().(bool)
			logger.Debug(fmt.Sprintf("规则 %s 评估结果: %v", rule.Value.Expression, ruleBool))
			customLib.WriteRuleFunctionsROptions(rule.Key, ruleBool)
		}

		// 更新output输出
		if len(rule.Value.Output) > 0 {
			finger2.IsFuzzSet(rule.Value.Output, SetiableMap, customLib)
		}
	}

	// 最终评估
	result, err := customLib.Evaluate(fg.Expression, SetiableMap)
	if err != nil {
		return false, fmt.Errorf("最终表达式解析错误：%v", err)
	}
	finalResult := result.Value().(bool)
	logger.Debug(fmt.Sprintf("最终规则 %s 评估结果: %v", fg.Expression, finalResult))
	return finalResult, nil
}

// shouldUseCache 判断是否应该使用缓存
func shouldUseCache(rule finger2.RuleMap) bool {
	if lastRequest == nil || lastResponse == nil {
		return false
	}

	reqType := strings.ToLower(rule.Value.Request.Type)
	method := strings.ToUpper(rule.Value.Request.Method)

	return rule.Value.Request.Path == "/" &&
		(reqType == "" || reqType == common.HTTP_Type) &&
		(method == "GET" || rule.Value.Request.Method == "")
}

// updateCache 更新请求响应缓存
func updateCache(variableMap map[string]any) {
	if resp, ok := variableMap["response"].(*proto.Response); ok {
		lastResponse = resp
	}
	if req, ok := variableMap["request"].(*proto.Request); ok {
		lastRequest = req
	}
}
