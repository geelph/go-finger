package runner

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"gxx/pkg/finger"
	"gxx/types"
	"gxx/utils/logger"
	"gxx/utils/output"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// 全局FingerPool，用于处理所有指纹任务
var GlobalFingerPool *ants.PoolWithFunc

// 指纹任务结果结构
type FingerTaskResult struct {
	Target    string
	FingerID  string
	Timestamp int64
	Success   bool
	Result    *FingerMatch
}

// 任务监控结构
type FingerTaskMonitor struct {
	sync.Mutex
	results       map[string][]FingerTaskResult // 按URL地址存储结果
	totalTasks    map[string]int                // 每个URL的总任务数
	pendingTasks  map[string]int                // 每个URL的待处理任务数
	completedURLs []string                      // 已完成处理的URL列表，用于清理
	maxStoredURLs int                           // 最大存储的已完成URL数量
}

// 新建指纹任务监控器
func NewFingerTaskMonitor() *FingerTaskMonitor {
	return &FingerTaskMonitor{
		results:      make(map[string][]FingerTaskResult),
		totalTasks:   make(map[string]int),
		pendingTasks: make(map[string]int),
	}
}

// 初始化URL任务
func (tm *FingerTaskMonitor) InitUrlTask(target string, taskCount int) {
	tm.Lock()
	defer tm.Unlock()
	tm.results[target] = make([]FingerTaskResult, 0, taskCount)
	tm.totalTasks[target] = taskCount
	tm.pendingTasks[target] = taskCount
}

// 添加结果
func (tm *FingerTaskMonitor) AddResult(result FingerTaskResult) {
	tm.Lock()
	defer tm.Unlock()
	target := result.Target

	// 初始化结果集（如果不存在）
	if _, exists := tm.results[target]; !exists {
		tm.results[target] = make([]FingerTaskResult, 0, 100)
	}

	// 添加结果
	tm.results[target] = append(tm.results[target], result)

	// 更新待处理任务数
	if tm.pendingTasks[target] > 0 {
		tm.pendingTasks[target]--
	}
}

// 获取URL的所有结果
func (tm *FingerTaskMonitor) GetResults(target string) []FingerTaskResult {
	tm.Lock()
	defer tm.Unlock()
	if results, exists := tm.results[target]; exists {
		return results
	}
	return []FingerTaskResult{}
}
func (tm *FingerTaskMonitor) ClearURLResults(target string) {
	tm.Lock()
	defer tm.Unlock()

	// 清除该URL的所有数据
	delete(tm.results, target)
	delete(tm.totalTasks, target)
	delete(tm.pendingTasks, target)

	// 从已完成列表中移除
	for i, url := range tm.completedURLs {
		if url == target {
			tm.completedURLs = append(tm.completedURLs[:i], tm.completedURLs[i+1:]...)
			break
		}
	}
}

// 创建全局任务监控器
var GlobalFingerTaskMonitor = NewFingerTaskMonitor()

// 初始化全局FingerPool
func InitGlobalFingerPool(workerCount int) {
	var err error
	GlobalFingerPool, err = ants.NewPoolWithFunc(workerCount, func(i interface{}) {
		task := i.(map[string]interface{})

		// 检查并获取WaitGroup
		if wg, ok := task["wg"].(*sync.WaitGroup); ok {
			defer wg.Done()
		}

		// 执行指纹识别
		fingerFg := task["fingerRule"].(*finger.Finger)
		target := task["target"].(string)
		baseInfo := task["baseInfo"].(*BaseInfo)
		proxy := task["proxy"].(string)
		timeout := task["timeout"].(int)

		// 将结果转换为FingerTaskResult
		result, err := evaluateFingerprintWithCache(fingerFg, target, baseInfo, proxy, timeout)
		success := err == nil && result != nil && result.Result

		// 生成任务结果
		taskResult := FingerTaskResult{
			Target:    target,
			FingerID:  fingerFg.Id,
			Timestamp: task["timestamp"].(int64),
			Success:   success,
			Result:    result,
		}

		// 如果成功匹配，添加到结果中
		if success {
			GlobalFingerTaskMonitor.AddResult(taskResult)
		}
	},
		ants.WithPreAlloc(true),
		ants.WithExpiryDuration(1*time.Minute),
		ants.WithNonblocking(false),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to create GlobalFingerPool: %v", err))
	}
}

// Runner 指纹识别运行器
type Runner struct {
	Config  *ScanConfig              // 配置参数
	Results map[string]*TargetResult // 扫描结果
	mutex   sync.Mutex               // 保护Results的读写锁
}

// NewRunner 创建一个新的扫描运行器
func NewRunner(options *types.CmdOptions) *Runner {
	// 设置并发参数
	urlWorkerCount := options.Threads
	if urlWorkerCount <= 0 {
		urlWorkerCount = 10
	}

	// 计算指纹规则线程池大小，使用与TestGlobalRulePool类似的策略
	fingerWorkerCount := 50 * urlWorkerCount

	// 限制范围在 500 到 1000
	if fingerWorkerCount < 500 {
		fingerWorkerCount = 500
	} else if fingerWorkerCount > 5000 {
		fingerWorkerCount = 5000
	}

	// 确定输出格式
	outputFormat := output.GetOutputFormat(options.JSONOutput, options.Output)

	// 创建配置
	config := &ScanConfig{
		Proxy:             options.Proxy,
		Timeout:           options.Timeout,
		URLWorkerCount:    urlWorkerCount,
		FingerWorkerCount: fingerWorkerCount,
		OutputFormat:      outputFormat,
		OutputFile:        options.Output,
		SockOutputFile:    options.SockOutput,
	}

	// 创建Runner实例
	runner := &Runner{
		Config:  config,
		Results: make(map[string]*TargetResult),
		mutex:   sync.Mutex{},
	}

	return runner
}

// Run 执行扫描
func (r *Runner) Run(options *types.CmdOptions) error {
	// 处理目标URL列表
	targets := getTargets(options)
	if len(targets) == 0 {
		return fmt.Errorf("未找到有效的目标URL")
	}

	// 在扫描开始前主动触发一次GC，清理之前可能的内存占用
	runtime.GC()

	logger.Info(fmt.Sprintf("准备扫描 %d 个目标", len(targets)))

	// 启动内存监控协程
	go monitorMemoryUsage()

	// 初始化输出文件
	if r.Config.OutputFile != "" {
		if err := output.InitOutput(r.Config.OutputFile, r.Config.OutputFormat); err != nil {
			return fmt.Errorf("初始化输出文件失败: %v", err)
		}
		defer func() {
			_ = output.Close()
		}()
	}

	// 初始化socket文件输出
	if r.Config.SockOutputFile != "" {
		if err := output.InitSockOutput(r.Config.SockOutputFile); err != nil {
			return fmt.Errorf("初始化socket输出文件失败: %v", err)
		}
		logger.Info(fmt.Sprintf("Socket输出文件：%s", r.Config.SockOutputFile))
	}

	// 加载指纹规则
	if err := LoadFingerprints(options.PocOptions); err != nil {
		return fmt.Errorf("加载指纹规则出错: %v", err)
	}
	logger.Info(fmt.Sprintf("加载指纹数量：%v个", len(AllFinger)))

	// 初始化全局指纹池
	if GlobalFingerPool == nil {
		InitGlobalFingerPool(r.Config.FingerWorkerCount)
		logger.Info("初始化全局指纹任务池")
	}

	// 在函数返回时释放全局池资源
	defer func() {
		if GlobalFingerPool != nil {
			GlobalFingerPool.Release()
		}
	}()

	// 执行垃圾回收，减少内存占用
	runtime.GC()

	logger.Info(fmt.Sprintf("开始扫描 %d 个目标，使用 %d 个URL并发线程, %d 个规则并发线程...",
		len(targets), r.Config.URLWorkerCount, r.Config.FingerWorkerCount))

	// 执行扫描
	r.runScan(targets, options)
	r.mutex.Lock()
	// 清除所有缓存文件
	ClearAllCache()
	printSummary(targets, r.Results)
	r.mutex.Unlock()

	return nil
}

// ScanTarget 扫描单个目标URL
func (r *Runner) ScanTarget(target string) (*TargetResult, error) {
	// 使用目标特定的worker数量
	workerCount := r.Config.FingerWorkerCount
	timeout := r.Config.Timeout

	// 处理单个URL
	result, err := ProcessURL(target, r.Config.Proxy, timeout, workerCount)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// runScan 执行扫描过程
func (r *Runner) runScan(targets []string, options *types.CmdOptions) {
	// 使用通道替代互斥锁来收集结果
	resultChan := make(chan struct {
		target string
		result *TargetResult
	}, len(targets))

	// 创建进度条
	bar := output.CreateProgressBar(len(targets))

	// 创建上下文用于控制goroutine
	doneChan := make(chan struct{}, len(targets))
	stopRefreshChan := make(chan struct{})

	// 添加定时刷新进度条的功能
	refreshTicker := time.NewTicker(500 * time.Millisecond)
	go func() {
		defer refreshTicker.Stop()
		for {
			select {
			case <-refreshTicker.C:
				// 定时刷新进度条显示
				err := bar.RenderBlank()
				if err != nil {
					logger.Debug(fmt.Sprintf("刷新进度条出错: %v", err))
				}
			case <-stopRefreshChan:
				// 收到停止信号时退出
				return
			}
		}
	}()

	// 启动进度条更新协程
	startTime := time.Now()
	go func() {
		for range doneChan {
			err := bar.Add(1)
			if err != nil {
				logger.Debug(fmt.Sprintf("更新进度条出错: %v", err))
			}
		}
	}()

	// 收集结果的协程
	go func() {
		for data := range resultChan {
			r.mutex.Lock()
			r.Results[data.target] = data.result
			r.mutex.Unlock()
		}
	}()

	// 存储输出的结果 - 无需互斥锁
	saveResult := func(msg string) {
		// 暂时清除进度条并输出结果
		fmt.Print("\033[2K\r")
		fmt.Println(msg)

		// 重新显示进度条
		err := bar.RenderBlank()
		if err != nil {
			logger.Debug(fmt.Sprintf("重新显示进度条出错: %v", err))
		}
	}

	// 定义任务结构体
	type scanTask struct {
		target string
	}

	var urlWg sync.WaitGroup

	// 创建URL处理工作池，使用PoolWithFunc预定义处理函数
	urlPool, _ := ants.NewPoolWithFunc(r.Config.URLWorkerCount,
		func(i interface{}) {
			defer urlWg.Done()
			task := i.(scanTask)
			target := task.target

			// 处理单个URL
			targetResult, err := ProcessURL(target, options.Proxy, options.Timeout, r.Config.FingerWorkerCount)
			if err != nil {
				logger.Error(fmt.Sprintf("处理目标 %s 失败: %v", target, err))
				targetResult = &TargetResult{
					URL:     target,
					Matches: make([]*FingerMatch, 0),
				}
			}

			// 将结果写入文件并显示结果
			handleMatchResults(targetResult, options, saveResult, r.Config.OutputFormat)

			// 通过通道发送结果
			resultChan <- struct {
				target string
				result *TargetResult
			}{target, targetResult}

			// 通知完成一个任务
			doneChan <- struct{}{}
		},
		ants.WithPreAlloc(true),
		ants.WithExpiryDuration(3*time.Minute),
		ants.WithNonblocking(false), // 使用非阻塞模式提高并发性能
	)
	defer urlPool.Release()

	// 分批提交任务到线程池
	batchSize := 100 // 增加每批次处理的目标数
	if len(targets) < batchSize {
		batchSize = len(targets)
	}

	// 提交所有目标到线程池
	for _, target := range targets {
		//fmt.Println(fmt.Sprintf("Runner goroutines：%d", urlPool.Running()))
		//fmt.Println(fmt.Sprintf("Free goroutines：%d", urlPool.Free()))
		urlWg.Add(1)
		err := urlPool.Invoke(scanTask{
			target: target,
		})

		// 如果提交失败，手动减少等待计数并记录错误
		if err != nil {
			urlWg.Done()
			logger.Error(fmt.Sprintf("提交目标 %s 到线程池失败: %v", target, err))
		}
	}
	runtime.GC()
	// 等待当前批次完成
	urlWg.Wait()

	// 等待所有URL处理完成
	close(resultChan)
	close(doneChan)

	// 停止刷新进度条
	close(stopRefreshChan)

	// 确保最终完成100%进度
	err := bar.Finish()
	if err != nil {
		logger.Debug(fmt.Sprintf("完成进度条出错: %v", err))
	}

	// 显示扫描耗时信息
	elapsedTime := time.Since(startTime)
	itemsPerSecond := float64(len(targets)) / elapsedTime.Seconds()

	maxProgress := fmt.Sprintf("指纹识别 100%% [==================================================] (%d/%d, %.2f it/s)",
		len(targets), len(targets), itemsPerSecond)
	fmt.Println(maxProgress)
}

// monitorMemoryUsage 内存监控函数
func monitorMemoryUsage() {
	ticker := time.NewTicker(10 * time.Second) // 降低检查频率到10秒
	defer ticker.Stop()

	var memStats runtime.MemStats
	var lastGC uint32 = 0                 // 记录上次GC时间
	var highMemWarningIssued bool = false // 内存高使用警告标志
	var lastMemAlloc uint64 = 0           // 上次检查的内存分配
	var growthRate float64 = 0            // 内存增长率

	for range ticker.C {
		runtime.ReadMemStats(&memStats)

		// 计算内存使用百分比（相对于系统内存）
		memUsagePercent := (float64(memStats.HeapAlloc) / float64(memStats.Sys)) * 100

		// 计算内存增长率
		if lastMemAlloc > 0 {
			growthRate = float64(memStats.HeapAlloc-lastMemAlloc) / float64(lastMemAlloc) * 100
		}
		lastMemAlloc = memStats.HeapAlloc

		// 记录内存使用情况，但在生产环境中不输出这些日志
		// logger.Debug(fmt.Sprintf("内存使用情况: 堆分配 %.2f MB (%.1f%%), 系统内存 %.2f MB, GC次数 %d, 增长率 %.1f%%",
		//	float64(memStats.HeapAlloc)/1024/1024,
		//	memUsagePercent,
		//	float64(memStats.Sys)/1024/1024,
		//	memStats.NumGC,
		//	growthRate))

		// 智能GC触发条件:
		// 1. 内存使用超过阈值 (1GB或85%)
		// 2. 内存增长率超过10%
		// 3. 距离上次GC时间超过30秒
		shouldGC := false

		if memStats.HeapAlloc > 1024*1024*1024 || memUsagePercent > 85 {
			// 条件1: 内存使用超过阈值
			shouldGC = true
		} else if growthRate > 10 {
			// 条件2: 内存增长率超过10%
			shouldGC = true
		} else if lastGC > 0 && memStats.NumGC > lastGC &&
			time.Since(time.Unix(int64(memStats.LastGC/1e9), 0)) > 30*time.Second {
			// 条件3: 距离上次GC时间超过30秒
			shouldGC = true
		}

		if shouldGC {
			if !highMemWarningIssued {
				logger.Debug("内存使用已触发智能GC")
				highMemWarningIssued = true
			}

			runtime.GC()
			lastGC = memStats.NumGC

			// 仅在极端情况下才强制归还内存
			if memUsagePercent > 95 || growthRate > 50 {
				logger.Debug("内存使用极限，强制释放系统内存")
				debug.FreeOSMemory()
			}
		} else {
			highMemWarningIssued = false
		}
	}
}
