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
	logger.Info(fmt.Sprintf("加载指纹数量：%v个", len(AllFinger)))
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
		timeoutDuration = 3 * time.Second // 使用默认3秒作为超时时间
	}

	options := network.OptionsRequest{
		Proxy:              proxy,
		Timeout:            timeoutDuration,
		Retries:            2,
		FollowRedirects:    true,
		InsecureSkipVerify: true,
		CustomHeaders: map[string]string{
			"User-Agent":      common.GetRandomIP(),
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
	var targets []string
	if len(options.Target) > 0 {
		targets = options.Target
	} else if options.TargetsFile != "" {
		// 从文件读取目标列表
		content, err := os.ReadFile(options.TargetsFile)
		if err != nil {
			logger.Error(fmt.Sprintf("读取目标文件失败: %v", err))
			return
		}
		// 按行分割，去除空行和空白
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				targets = append(targets, line)
			}
		}
	}

	if len(targets) == 0 {
		logger.Error("未找到有效的目标URL")
		return
	}

	// 记录原始URL数量
	originalCount := len(targets)
	logger.Info(fmt.Sprintf("原始目标数量：%v个", originalCount))

	// 进行URL去重
	targets = common.RemoveDuplicateURLs(targets)

	// 记录去重后的URL数量和重复URL数量
	duplicateCount := originalCount - len(targets)
	logger.Info(fmt.Sprintf("重复目标数量：%v个，去重后目标数量：%v个", duplicateCount, len(targets)))

	proxy := options.Proxy
	logger.Debug(fmt.Sprintf("使用代理 Proxy: %s", proxy))

	// 加载指纹规则
	if err := loadFingerprints(options); err != nil {
		logger.Error("加载指纹规则出错")
		return
	}

	// 确定输出格式
	outputFormat := "txt" // 默认为txt格式
	if options.Output != "" {
		ext := strings.ToLower(filepath.Ext(options.Output))
		if ext == ".csv" {
			outputFormat = "csv"
		}
	}

	// 初始化输出文件
	if options.Output != "" {
		if err := output.InitOutput(options.Output, outputFormat); err != nil {
			logger.Error(fmt.Sprintf("初始化输出文件失败: %v", err))
			return
		}
		// 确保在函数结束时关闭输出文件
		defer func() {
			_ = output.Close()
		}()
	}

	logger.Info("开始目标扫描...")

	// 创建结果存储
	results := make(map[string]*TargetResult)
	var resultsMutex sync.Mutex

	// 创建互斥锁用于控制输出，避免输出混乱
	var outputMutex sync.Mutex

	// 创建URL处理协程池
	urlWorkerCount := 10
	if options.Threads > 0 {
		urlWorkerCount = options.Threads
	}
	logger.Info(fmt.Sprintf("使用URL处理线程：%v个", urlWorkerCount))

	// 创建指纹识别线程池大小
	fingerWorkerCount := 5 * urlWorkerCount
	logger.Info(fmt.Sprintf("每个URL使用指纹识别线程：%v个", fingerWorkerCount))

	// 创建URL任务通道和完成通道
	urlChan := make(chan string, len(targets))
	doneChan := make(chan struct{}, len(targets))

	// 创建一个WaitGroup来等待所有URL处理完成
	var urlWg sync.WaitGroup

	// 创建错误通道
	errorChan := make(chan error, len(targets))

	// 创建自定义进度条
	bar := progressbar.NewOptions64(
		int64(len(targets)),
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

	// 初始显示进度条
	_ = bar.RenderBlank()

	// 存储输出的结果
	saveResult := func(msg string) {
		outputMutex.Lock()
		defer outputMutex.Unlock()

		// 暂时清除进度条
		fmt.Print("\033[2K\r")

		// 输出结果
		fmt.Println(msg)

		// 重新显示进度条
		_ = bar.RenderBlank()
	}

	// 启动一个协程来更新进度条
	startTime := time.Now()
	go func() {
		for range doneChan {
			outputMutex.Lock()
			_ = bar.Add(1)
			outputMutex.Unlock()
		}
	}()

	// 启动URL工作协程
	for i := 0; i < urlWorkerCount; i++ {
		urlWg.Add(1)
		go func() {
			defer urlWg.Done()

			for target := range urlChan {
				// 处理单个URL
				targetResult, err := processURL(target, proxy, options.Timeout, fingerWorkerCount, options, saveResult, outputFormat)
				if err != nil {
					errorChan <- fmt.Errorf("处理URL %s 失败: %v", target, err)
				}

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

	// 计算执行时间
	elapsedTime := time.Since(startTime)
	itemsPerSecond := float64(len(targets)) / elapsedTime.Seconds()

	// 显示一行完整的100%进度条
	maxProgress := fmt.Sprintf("指纹识别 100%% [==================================================] (%d/%d, %.2f it/s)",
		len(targets), len(targets), itemsPerSecond)
	fmt.Println(maxProgress)

	close(errorChan)

	// 收集所有错误
	var errors []string
	for err := range errorChan {
		errors = append(errors, err.Error())
	}

	logger.Info("指纹识别完成，开始生成结果汇总...")

	// 输出最终统计信息
	matchCount := 0
	noMatchCount := 0

	// 统计匹配成功和失败的数量
	for _, targetResult := range results {
		if len(targetResult.Matches) > 0 {
			matchCount++
		} else {
			noMatchCount++
			// 只输出未匹配的URL信息，因为匹配成功的在处理完成时已经输出
			statusCodeStr := ""
			if targetResult.StatusCode > 0 {
				statusCodeStr = fmt.Sprintf("（%d）", targetResult.StatusCode)
			}

			serverInfo := ""
			if targetResult.Server != nil {
				serverInfo = fmt.Sprintf("%s", targetResult.Server.ServerType)
			}

			baseInfo := fmt.Sprintf("URL：%s %s  标题：%s  Server：%s",
				targetResult.URL,
				statusCodeStr,
				targetResult.Title,
				serverInfo)

			outputMsg := fmt.Sprintf("%s  匹配结果：%s",
				baseInfo,
				color.RedString("未匹配"))
			fmt.Println(outputMsg)
		}
	}

	// 输出统计信息
	fmt.Println(color.CyanString("─────────────────────────────────────────────────────"))
	fmt.Printf("扫描统计: 目标总数 %d, 匹配成功 %d, 匹配失败 %d, 总耗时 %.2fs\n",
		len(targets),
		matchCount,
		noMatchCount,
		elapsedTime.Seconds())

	// 输出收集的错误信息
	if len(errors) > 0 {
		logger.Info(fmt.Sprintf("共有 %d 个错误发生", len(errors)))
		for _, err := range errors {
			logger.Error(err)
		}
	}
}

// processURL 处理单个URL的所有指纹识别
func processURL(target string, proxy string, timeout int, workerCount int, options *types.CmdOptions, printResult func(string), outputFormat string) (*TargetResult, error) {
	// 获取目标基础信息
	logger.Debug(fmt.Sprintf("获取目标 %s 的基础信息", target))
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
		// 使用空的基础信息
		targetResult.Title = ""
		targetResult.Server = types.EmptyServerInfo()
		targetResult.StatusCode = 0
	}

	// 从httpResp中提取响应头用于保存
	var initialResponse *proto.Response
	if httpResp != nil {
		// 读取响应体并转换为UTF-8
		respBody, _ := io.ReadAll(httpResp.Body)
		_ = httpResp.Body.Close()
		// 使用Str2UTF8替代ConvertToUTF8
		utf8RespBody := common.Str2UTF8(string(respBody))
		initialResponse = finger2.BuildProtoResponse(httpResp, utf8RespBody, 0, proxy)
		lastResponse = initialResponse // 保存初始响应
	}

	// 创建基础信息对象供指纹识别使用
	baseInfo := &BaseInfo{
		Title:      targetResult.Title,
		Server:     targetResult.Server,
		StatusCode: targetResult.StatusCode,
	}

	// 如果没有指纹规则，直接返回结果
	if len(AllFinger) == 0 {
		return targetResult, nil
	}

	// 使用有缓冲的通道避免阻塞
	bufferSize := min(len(AllFinger), 1000) // 避免过大的缓冲区

	// 创建指纹识别任务通道
	fingerChan := make(chan *finger2.Finger, bufferSize)
	// 创建结果通道
	resultChan := make(chan *FingerMatch, bufferSize)
	// 创建错误通道
	fingerErrorChan := make(chan error, bufferSize)
	// 创建等待组
	var fingerWg sync.WaitGroup

	// 预先创建并复用CustomLib实例
	customLibs := make([]*cel.CustomLib, workerCount)
	for i := 0; i < workerCount; i++ {
		customLibs[i] = cel.NewCustomLib()
	}

	// 启动指纹工作协程
	for i := 0; i < workerCount; i++ {
		fingerWg.Add(1)
		go func(workerID int) {
			defer fingerWg.Done()
			// 使用预先创建的CustomLib实例
			customLib := customLibs[workerID]

			for fg := range fingerChan {
				// 重置CustomLib避免上下文污染
				customLib.Reset()

				// 执行指纹识别
				result, err := evaluateFingerprintWithCache(fg, target, baseInfo, proxy, customLib, timeout)
				if err != nil {
					select {
					case fingerErrorChan <- fmt.Errorf("URL %s 的指纹 %s 评估失败: %v", target, fg.Id, err):
					default:
						// 如果通道已满，丢弃错误
					}
				} else if result {
					// 只存储匹配成功的指纹
					select {
					case resultChan <- &FingerMatch{
						Finger: fg,
						Result: true,
					}:
					default:
						// 如果通道已满，记录日志
						logger.Debug(fmt.Sprintf("结果通道已满，丢弃指纹 %s 的结果", fg.Id))
					}
				}
			}
		}(i)
	}

	// 使用另一个goroutine发送指纹任务，避免阻塞
	go func() {
		for _, fg := range AllFinger {
			fingerChan <- fg
		}
		close(fingerChan)
	}()

	// 启动一个协程收集结果
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

	// 关闭结果通道
	close(resultChan)
	close(fingerErrorChan)

	// 等待结果收集完成并赋值
	time.Sleep(10 * time.Millisecond)
	matchesMutex.Lock()
	targetResult.Matches = matches
	matchesMutex.Unlock()

	// 收集错误
	var fingerErrors []error
	for err := range fingerErrorChan {
		// 只收集前100个错误，避免内存占用过多
		if len(fingerErrors) < 100 {
			fingerErrors = append(fingerErrors, err)
		} else {
			break
		}
	}

	// 如果有太多错误，记录一些统计信息
	if len(fingerErrors) > 0 {
		logger.Debug(fmt.Sprintf("URL %s 指纹识别过程中发生 %d 个错误", target, len(fingerErrors)))
	}

	// URL处理完成后处理结果
	if len(targetResult.Matches) > 0 {
		// 构建基础信息输出
		statusCodeStr := ""
		if targetResult.StatusCode > 0 {
			statusCodeStr = fmt.Sprintf("（%d）", targetResult.StatusCode)
		}

		serverInfo := ""
		if targetResult.Server != nil {
			serverInfo = fmt.Sprintf("%s", targetResult.Server.ServerType)
		}

		// 构建统一格式的输出信息
		baseInfoStr := fmt.Sprintf("URL：%s %s  标题：%s  Server：%s",
			targetResult.URL,
			statusCodeStr,
			targetResult.Title,
			serverInfo)

		// 收集所有匹配的指纹名称
		fingerNames := make([]string, 0, len(targetResult.Matches))
		for _, match := range targetResult.Matches {
			fingerNames = append(fingerNames, match.Finger.Info.Name)
		}

		// 输出匹配成功的信息
		outputMsg := fmt.Sprintf("%s  指纹：[%s]  匹配结果：%s",
			baseInfoStr,
			strings.Join(fingerNames, "，"),
			color.GreenString("成功"))

		// 通过回调函数添加结果
		printResult(outputMsg)

		// 收集指纹信息写入结果文件
		if options.Output != "" {
			fingerList := make([]*finger2.Finger, 0, len(targetResult.Matches))
			for _, match := range targetResult.Matches {
				fingerList = append(fingerList, match.Finger)
			}

			// 创建写入选项结构体
			writeOpts := &output.WriteOptions{
				Output:      options.Output,
				Format:      outputFormat,
				Target:      target,
				Fingers:     fingerList,
				StatusCode:  targetResult.StatusCode,
				Title:       targetResult.Title,
				ServerInfo:  targetResult.Server,
				FinalResult: true,
			}

			// 添加响应头信息
			if lastResponse != nil {
				writeOpts.Response = lastResponse
			} else if initialResponse != nil {
				// 如果没有完整的lastResponse，使用初始响应
				writeOpts.Response = initialResponse
			}
			// 使用指定格式写入结果
			if err := output.WriteFingerprints(writeOpts); err != nil {
				logger.Error(fmt.Sprintf("写入结果失败: %v", err))
			}
		}
	}

	return targetResult, nil
}

// SendRequest获取的最后一个请求响应
var lastResponse *proto.Response

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
		// 缓存标志，判断是否使用缓存的响应
		useCache := false

		// 只有当满足以下条件时才使用缓存:
		// 1. 请求路径是根路径 "/"
		// 2. 请求类型是HTTP或为空（默认为HTTP）
		// 3. 请求方法是GET
		reqType := strings.ToLower(rule.Value.Request.Type)
		if rule.Value.Request.Path == "/" &&
			(reqType == "" || reqType == common.HTTP_Type) &&
			(strings.ToUpper(rule.Value.Request.Method) == "GET" || rule.Value.Request.Method == "") {
			useCache = true
			logger.Debug(fmt.Sprintf("规则 %s 使用缓存的根路径HTTP响应", rule.Key))
		}
		// 如果不使用缓存，则发送新请求
		if !useCache {
			logger.Debug(fmt.Sprintf("发送指纹探测请求，路径：%s，类型：%s", rule.Value.Request.Path, rule.Value.Request.Type))
			SetiableMaps, err := finger2.SendRequest(target, rule.Value.Request, rule.Value, SetiableMap, proxy, timeout)
			if err != nil {
				logger.Debug(fmt.Sprintf("规则 %s 请求出错：%s", rule.Key, err.Error()))
				customLib.WriteRuleFunctionsROptions(rule.Key, false)
				continue
			}
			logger.Debug("请求发送完成，开始请求处理")
			if len(SetiableMaps) > 0 {
				// 完全替换为新的map
				SetiableMap = SetiableMaps

				// 保存最后一个响应用于输出
				if resp, ok := SetiableMap["response"].(*proto.Response); ok {
					lastResponse = resp
				}
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
