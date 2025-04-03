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
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
		// 遍历目录中的所有文件
		return filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return err
			}
			if !info.IsDir() && common.IsYamlFile(path) {
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

// GetBaseInfo 获取目标的基础信息（标题和Server信息）
func GetBaseInfo(target, proxy string, timeout int) (string, *types.ServerInfo, int32, error) {
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
		return "", nil, 0, fmt.Errorf("发送请求失败: %v", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	// 获取状态码
	statusCode := int32(resp.StatusCode)

	// 使用finger2包的GetTitle方法提取标题
	title := finger2.GetTitle(urlWithProtocol, resp)

	// 使用finger2包的GetServerInfoFromResponse方法提取Server信息
	serverInfo := finger2.GetServerInfoFromResponse(resp)

	return title, serverInfo, statusCode, nil
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
	logger.Info(fmt.Sprintf("重复目标数量：%v个", duplicateCount))
	logger.Info(fmt.Sprintf("去重后目标数量：%v个", len(targets)))
	
	proxy := options.Proxy
	logger.Debug(fmt.Sprintf("使用代理 Proxy: %s", proxy))

	// 加载指纹规则
	if err := loadFingerprints(options); err != nil {
		logger.Error("加载指纹规则出错")
		return
	}
	// 创建目标基础信息缓存
	targetBaseInfoCache := make(map[string]*BaseInfo)
	// 预先获取所有目标的基础信息
	logger.Info("开始获取目标基础信息...")
	for _, target := range targets {
		logger.Debug(fmt.Sprintf("获取目标 %s 的基础信息", target))
		title, serverInfo, statusCode, err := GetBaseInfo(target, proxy, options.Timeout)
		if err != nil {
			// 即使获取失败，也创建一个空的基础信息对象
			targetBaseInfoCache[target] = &BaseInfo{
				Title:      "",
				Server:     types.EmptyServerInfo(),
				StatusCode: 0,
			}
		} else {
			targetBaseInfoCache[target] = &BaseInfo{
				Title:      title,
				Server:     serverInfo,
				StatusCode: statusCode,
			}
		}
	}

	logger.Info("目标基础信息获取完成")

	// 创建工作池
	workerCount := 10
	if options.Threads > 0 {
		workerCount = options.Threads
	}
	logger.Info(fmt.Sprintf("使用工作线程：%v个", workerCount))

	// 创建任务通道 - 以URL为单位创建任务
	type Task struct {
		Target string
		Finger *finger2.Finger
	}

	// 计算总任务数
	totalTasks := len(targets) * len(AllFinger)
	logger.Info(fmt.Sprintf("总任务数：%v个", totalTasks))

	// 创建任务通道
	taskChan := make(chan Task, totalTasks)

	// 创建错误通道
	errorChan := make(chan error, totalTasks)

	// 存储所有目标的结果，按目标URL索引
	results := make(map[string]*TargetResult)

	// 创建互斥锁保护结果映射的并发访问
	var resultsMutex sync.Mutex

	// 再次暂停终端日志输出
	logger.PauseTerminalLogging()

	// 创建最简单的进度条，仅使用ASCII字符
	scanBar := progressbar.NewOptions64(
		int64(totalTasks),
		progressbar.OptionSetWidth(50), // 增加宽度，显示更清晰
		progressbar.OptionEnableColorCodes(false),
		progressbar.OptionShowBytes(false),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(), // 显示每秒处理数量
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionSetDescription("指纹识别"), // 更明确的描述
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=", // 标准ASCII字符
			SaucerHead:    ">", // 标准ASCII字符
			SaucerPadding: " ", // 空格作为填充
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionOnCompletion(func() {
			fmt.Println() // 完成后只添加一个换行，不清除进度条
		}),
	)

	// 创建计数器来跟踪已完成的任务
	var completedTasks int64

	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 每个工作协程创建自己的CustomLib实例
			customLib := cel.NewCustomLib()

			for task := range taskChan {
				// 重置CustomLib避免上下文污染
				customLib.Reset()

				// 获取或创建当前目标的结果对象
				resultsMutex.Lock()
				targetResult, exists := results[task.Target]
				if !exists {
					// 首次处理该目标，创建结果对象并使用缓存的基础信息
					baseInfo := targetBaseInfoCache[task.Target]
					targetResult = &TargetResult{
						URL:        task.Target,
						StatusCode: baseInfo.StatusCode,
						Title:      baseInfo.Title,
						Server:     baseInfo.Server,
						Matches:    make([]*FingerMatch, 0),
					}
					results[task.Target] = targetResult
				}
				resultsMutex.Unlock()

				// 执行指纹识别，传入已缓存的基础信息，避免重复请求
				baseInfo := targetBaseInfoCache[task.Target]
				result, err := evaluateFingerprintWithCache(task.Finger, task.Target, baseInfo, proxy, customLib, options.Timeout)
				if err != nil {
					errorChan <- fmt.Errorf("URL %s 的指纹 %s 评估失败: %v", task.Target, task.Finger.Id, err)
				} else {
					// 存储匹配结果
					fingerMatch := &FingerMatch{
						Finger: task.Finger,
						Result: result,
					}

					// 更新目标结果（需要加锁）
					resultsMutex.Lock()
					// 只记录匹配成功的指纹
					if result {
						targetResult.Matches = append(targetResult.Matches, fingerMatch)
					}
					resultsMutex.Unlock()
				}

				// 更新进度条和计数器
				_ = scanBar.Add(1)
				atomic.AddInt64(&completedTasks, 1)
			}
		}()
	}

	// 发送任务 - 双层循环，先按URL再按指纹
	for _, target := range targets {
		for _, fg := range AllFinger {
			taskChan <- Task{
				Target: target,
				Finger: fg,
			}
		}
	}
	close(taskChan)

	// 等待所有工作协程完成
	wg.Wait()

	// 确保进度条完成，但不清除显示
	_ = scanBar.Finish()

	// 恢复终端日志输出
	logger.ResumeTerminalLogging()

	close(errorChan)

	// 收集所有错误
	var errors []string
	for err := range errorChan {
		errors = append(errors, err.Error())
	}

	logger.Info("指纹识别完成，开始生成结果报告...")

	// 处理并输出所有结果
	matchCount := 0
	fmt.Println(color.CyanString("─────────────────────────────────────────────────────"))
	fmt.Println(color.GreenString("结果报告："))
	for _, target := range targets {
		targetResult, exists := results[target]
		if !exists {
			// 目标没有结果，可能是因为处理过程中出错了
			continue
		}

		// 构建基础信息输出
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

		// 如果有匹配的指纹，输出匹配信息
		if len(targetResult.Matches) > 0 {
			matchCount++
			// 收集所有匹配的指纹名称
			var fingerNames []string
			for _, match := range targetResult.Matches {
				fingerNames = append(fingerNames, match.Finger.Info.Name)

				// 写入结果文件
				if options.Output != "" {
					// 从文件扩展名确定输出格式
					outputFormat := "txt" // 默认为txt格式
					if ext := strings.ToLower(filepath.Ext(options.Output)); ext == ".csv" {
						outputFormat = "csv"
					}

					if err := output.WriteResult(options.Output, outputFormat, targetResult.URL, match.Finger, true); err != nil {
						logger.Error(fmt.Sprintf("写入结果失败: %v", err))
					}
				}
			}

			// 一次性输出所有匹配的指纹
			outputMsg := fmt.Sprintf("%s  指纹：[%s]  匹配结果：%s",
				baseInfo,
				strings.Join(fingerNames, "，"),
				color.GreenString("成功"))
			fmt.Println(outputMsg)
		} else {
			// 如果没有匹配的指纹，输出未匹配信息
			outputMsg := fmt.Sprintf("%s  匹配结果：%s",
				baseInfo,
				color.RedString("未匹配"))
			fmt.Println(outputMsg)
		}
	}

	// 输出统计信息
	fmt.Println(color.CyanString("─────────────────────────────────────────────────────"))
	fmt.Printf("扫描统计: 目标总数 %d, 匹配成功 %d, 匹配失败 %d\n",
		len(targets),
		matchCount,
		len(targets)-matchCount)

	// 输出收集的错误信息
	if len(errors) > 0 {
		logger.Info(fmt.Sprintf("共有 %d 个错误发生", len(errors)))
		for _, err := range errors {
			logger.Error(err)
		}
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
			}
		}

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
