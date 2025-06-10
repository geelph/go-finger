package runner

import (
	"fmt"
	"gxx/pkg/finger"
	"gxx/types"
	"gxx/utils/logger"
	"gxx/utils/output"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"
)

// 全局配置常量 - 导出供外部使用
const (
	DefaultURLWorkers  = 5    // URL处理池默认大小
	DefaultRuleWorkers = 200  // 规则处理池默认大小
	MaxRuleWorkers     = 5000 // 最大规则工作线程
	MinRuleWorkers     = 100  // 最小规则工作线程
)

// GlobalRulePool 全局规则处理池，专门用于执行CEL规则识别
var GlobalRulePool *ants.PoolWithFunc

// GlobalPoolStats 全局池统计信息，用于监控性能
type GlobalPoolStats struct {
	TotalTasks     int64 // 总任务数
	CompletedTasks int64 // 已完成任务数
	FailedTasks    int64 // 失败任务数
}

var poolStats GlobalPoolStats

// RuleTask 规则处理任务结构
type RuleTask struct {
	Target     string
	Finger     *finger.Finger
	BaseInfo   *BaseInfo
	Proxy      string
	Timeout    int
	ResultChan chan<- *FingerMatch // 结果通道
	WaitGroup  *sync.WaitGroup     // 等待组
}

// InitGlobalRulePool 初始化全局规则处理池
func InitGlobalRulePool(workerCount int) error {
	// 确保worker数量在合理范围内
	if workerCount < MinRuleWorkers {
		workerCount = MinRuleWorkers
	} else if workerCount > MaxRuleWorkers {
		workerCount = MaxRuleWorkers
	}

	var err error
	GlobalRulePool, err = ants.NewPoolWithFunc(workerCount,
		func(i interface{}) {
			task, ok := i.(*RuleTask)
			if !ok {
				atomic.AddInt64(&poolStats.FailedTasks, 1)
				logger.Error("无效的规则任务类型")
				return
			}

			// 处理任务
			processRuleTask(task)

			// 更新完成统计
			atomic.AddInt64(&poolStats.CompletedTasks, 1)
		},
		ants.WithPreAlloc(true),                   // 预分配goroutine
		ants.WithExpiryDuration(2*time.Minute),    // 52分钟过期时间
		ants.WithNonblocking(false),               // 阻塞提交
		ants.WithMaxBlockingTasks(workerCount*10), // 最大阻塞任务数
		ants.WithPanicHandler(func(i interface{}) {
			atomic.AddInt64(&poolStats.FailedTasks, 1)
			logger.Error(fmt.Sprintf("规则池goroutine异常: %v", i))
		}),
	)

	if err != nil {
		return fmt.Errorf("创建全局规则池失败: %v", err)
	}

	logger.Info(fmt.Sprintf("全局规则池初始化完成，工作线程数: %d", workerCount))
	return nil
}

// processRuleTask 处理单个规则识别任务
func processRuleTask(task *RuleTask) {
	defer func() {
		if task.WaitGroup != nil {
			task.WaitGroup.Done()
		}
	}()

	// 执行指纹识别
	result, err := evaluateFingerprintWithCache(
		task.Finger,
		task.Target,
		task.BaseInfo,
		task.Proxy,
		task.Timeout,
	)

	if err != nil {
		atomic.AddInt64(&poolStats.FailedTasks, 1)
		logger.Debug(fmt.Sprintf("规则 %s 执行失败: %v", task.Finger.Id, err))
		return
	}

	// 只有匹配成功的结果才发送到结果通道
	if result != nil && result.Result {
		select {
		case task.ResultChan <- result:
			// 成功发送结果
		default:
			// 结果通道已满或关闭，丢弃结果
			logger.Debug(fmt.Sprintf("结果通道已满，丢弃规则 %s 的结果", task.Finger.Id))
		}
	}
}

// Runner 指纹识别运行器
type Runner struct {
	Config    *ScanConfig              // 配置参数
	Results   map[string]*TargetResult // 扫描结果
	mutex     sync.RWMutex             // 读写锁保护Results
	urlPool   *ants.PoolWithFunc       // URL处理池
	isRunning atomic.Bool              // 运行状态标志
}

// NewRunner 创建一个新的扫描运行器
func NewRunner(options *types.CmdOptions) *Runner {
	// 设置URL并发参数，默认为10
	urlWorkerCount := options.Threads
	if urlWorkerCount <= 0 {
		urlWorkerCount = DefaultURLWorkers
	}

	// 设置规则并发参数，默认为500
	var ruleWorkerCount int
	if options.RuleThreads > 0 {
		ruleWorkerCount = options.RuleThreads
	} else {
		ruleWorkerCount = DefaultRuleWorkers
	}

	// 确定输出格式
	outputFormat := output.GetOutputFormat(options.JSONOutput, options.Output)

	// 创建配置
	config := &ScanConfig{
		Proxy:             options.Proxy,
		Timeout:           options.Timeout,
		URLWorkerCount:    urlWorkerCount,
		FingerWorkerCount: ruleWorkerCount,
		OutputFormat:      outputFormat,
		OutputFile:        options.Output,
		SockOutputFile:    options.SockOutput,
	}

	// 创建Runner实例
	runner := &Runner{
		Config:  config,
		Results: make(map[string]*TargetResult),
		mutex:   sync.RWMutex{},
	}

	return runner
}

// Run 执行扫描
func (r *Runner) Run(options *types.CmdOptions) error {
	if !r.isRunning.CompareAndSwap(false, true) {
		return fmt.Errorf("扫描器已在运行中")
	}
	defer r.isRunning.Store(false)

	// 处理目标URL列表
	targets := getTargets(options)
	if len(targets) == 0 {
		return fmt.Errorf("未找到有效的目标URL")
	}

	logger.Info(fmt.Sprintf("准备扫描 %d 个目标", len(targets)))

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

	// 初始化全局规则池
	if GlobalRulePool == nil {
		if err := InitGlobalRulePool(r.Config.FingerWorkerCount); err != nil {
			return err
		}
	}

	// 在函数返回时释放全局池资源
	defer func() {
		if GlobalRulePool != nil {
			GlobalRulePool.Release()
			GlobalRulePool = nil
		}
	}()

	logger.Info(fmt.Sprintf("开始扫描 %d 个目标，使用 %d 个URL并发线程, %d 个规则并发线程...",
		len(targets), r.Config.URLWorkerCount, r.Config.FingerWorkerCount))

	// 执行扫描
	if err := r.runScan(targets, options); err != nil {
		return err
	}

	// 清除所有缓存
	ClearAllCache()

	// 打印统计信息
	r.mutex.RLock()
	printSummary(targets, r.Results)
	r.mutex.RUnlock()

	return nil
}

// ScanTarget 扫描单个目标URL
func (r *Runner) ScanTarget(target string) (*TargetResult, error) {
	if !r.isRunning.Load() {
		return nil, fmt.Errorf("扫描器未运行")
	}

	// 处理单个URL
	result, err := ProcessURL(target, r.Config.Proxy, r.Config.Timeout, r.Config.FingerWorkerCount)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// runScan 执行扫描过程
func (r *Runner) runScan(targets []string, options *types.CmdOptions) error {
	// 使用缓冲通道收集结果，避免阻塞
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
				if err := bar.RenderBlank(); err != nil {
					logger.Debug(fmt.Sprintf("刷新进度条出错: %v", err))
				}
			case <-stopRefreshChan:
				return
			}
		}
	}()

	// 启动进度条更新协程
	startTime := time.Now()
	go func() {
		for range doneChan {
			if err := bar.Add(1); err != nil {
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

	// 存储输出的结果 - 线程安全的结果输出
	saveResult := func(msg string) {
		fmt.Print("\033[2K\r")
		fmt.Println(msg)
		if err := bar.RenderBlank(); err != nil {
			logger.Debug(fmt.Sprintf("重新显示进度条出错: %v", err))
		}
	}

	// 定义URL处理任务结构体
	type urlTask struct {
		target string
	}

	var urlWg sync.WaitGroup

	// 创建URL处理工作池
	urlPool, err := ants.NewPoolWithFunc(r.Config.URLWorkerCount,
		func(i interface{}) {
			defer urlWg.Done()
			task, ok := i.(urlTask)
			if !ok {
				logger.Error("无效的URL任务类型")
				return
			}

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
			select {
			case resultChan <- struct {
				target string
				result *TargetResult
			}{target, targetResult}:
			default:
				logger.Debug("结果通道已满，丢弃结果")
			}

			// 通知完成一个任务
			select {
			case doneChan <- struct{}{}:
			default:
				logger.Debug("完成通道已满")
			}
		},
		ants.WithPreAlloc(true),
		ants.WithExpiryDuration(3*time.Minute),
		ants.WithNonblocking(false),
		ants.WithMaxBlockingTasks(r.Config.URLWorkerCount*5), // 限制阻塞任务数
		ants.WithPanicHandler(func(i interface{}) {
			logger.Error(fmt.Sprintf("URL池goroutine异常: %v", i))
		}),
	)

	if err != nil {
		return fmt.Errorf("创建URL处理池失败: %v", err)
	}
	defer urlPool.Release()

	// 提交所有目标到线程池
	for _, target := range targets {
		urlWg.Add(1)
		if err := urlPool.Invoke(urlTask{target: target}); err != nil {
			urlWg.Done()
			logger.Error(fmt.Sprintf("提交目标 %s 到线程池失败: %v", target, err))
		}
	}

	// 等待当前批次完成
	urlWg.Wait()

	// 等待所有URL处理完成
	close(resultChan)
	close(doneChan)

	// 停止刷新进度条
	close(stopRefreshChan)

	// 确保最终完成100%进度
	if err := bar.Finish(); err != nil {
		logger.Debug(fmt.Sprintf("完成进度条出错: %v", err))
	}

	// 显示扫描耗时信息
	elapsedTime := time.Since(startTime)
	itemsPerSecond := float64(len(targets)) / elapsedTime.Seconds()

	maxProgress := fmt.Sprintf("指纹识别 100%% [==================================================] (%d/%d, %.2f it/s)",
		len(targets), len(targets), itemsPerSecond)
	fmt.Println(maxProgress)

	// 打印池统计信息
	logger.Info(fmt.Sprintf("规则池统计 - 总任务: %d, 已完成: %d, 失败: %d",
		atomic.LoadInt64(&poolStats.TotalTasks),
		atomic.LoadInt64(&poolStats.CompletedTasks),
		atomic.LoadInt64(&poolStats.FailedTasks)))

	return nil
}

// GetPoolStats 获取池统计信息
func GetPoolStats() GlobalPoolStats {
	return GlobalPoolStats{
		TotalTasks:     atomic.LoadInt64(&poolStats.TotalTasks),
		CompletedTasks: atomic.LoadInt64(&poolStats.CompletedTasks),
		FailedTasks:    atomic.LoadInt64(&poolStats.FailedTasks),
	}
}

// ResetPoolStats 重置池统计信息
func ResetPoolStats() {
	atomic.StoreInt64(&poolStats.TotalTasks, 0)
	atomic.StoreInt64(&poolStats.CompletedTasks, 0)
	atomic.StoreInt64(&poolStats.FailedTasks, 0)
}
