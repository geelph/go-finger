package runner

import (
	"fmt"
	"gxx/types"
	"gxx/utils/logger"
	"gxx/utils/output"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
)

// Runner 指纹识别运行器
type Runner struct {
	Config  *ScanConfig              // 配置参数
	Results map[string]*TargetResult // 扫描结果
	mutex   sync.RWMutex             // 保护Results的读写锁
}

// NewRunner 创建一个新的扫描运行器
func NewRunner(options *types.CmdOptions) *Runner {
	// 设置并发参数
	urlWorkerCount := options.Threads
	if urlWorkerCount <= 0 {
		urlWorkerCount = 10 // 默认10个线程
	}
	fingerWorkerCount := 5 * urlWorkerCount // rule线程是URL线程的5倍

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
	return &Runner{
		Config:  config,
		Results: make(map[string]*TargetResult),
		mutex:   sync.RWMutex{},
	}
}

// Run 执行扫描
func (r *Runner) Run(options *types.CmdOptions) error {
	// 处理目标URL列表
	targets := getTargets(options)
	if len(targets) == 0 {
		return fmt.Errorf("未找到有效的目标URL")
	}

	// 加载指纹规则
	if err := LoadFingerprints(options.PocOptions); err != nil {
		return fmt.Errorf("加载指纹规则出错: %v", err)
	}
	logger.Info(fmt.Sprintf("加载指纹数量：%v个", len(AllFinger)))

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

	// 配置ants全局参数以提高性能
	ants.Release()

	// 优化参数调整
	ants.WithOptions(ants.Options{
		ExpiryDuration:   30 * time.Second, // 减少过期时间
		PreAlloc:         true,             // 预分配内存
		MaxBlockingTasks: len(targets) * 5, // 根据目标数量调整最大阻塞任务数
		Nonblocking:      false,            // 使用阻塞模式以避免任务丢失
		PanicHandler: func(err interface{}) { // 添加panic处理
			logger.Error(fmt.Sprintf("Worker panic: %v", err))
		},
	})

	logger.Info(fmt.Sprintf("开始扫描 %d 个目标，使用 %d 个URL并发线程, %d 个规则并发线程...",
		len(targets), r.Config.URLWorkerCount, r.Config.FingerWorkerCount))

	// 执行扫描
	r.runScan(targets, options)

	// 输出统计信息
	printSummary(targets, r.Results)

	return nil
}

// ScanTarget 扫描单个目标URL
func (r *Runner) ScanTarget(target string) (*TargetResult, error) {
	// 设置独立的worker数量以避免全局竞争
	workerCount := r.Config.FingerWorkerCount / 2
	if workerCount <= 0 {
		workerCount = 10
	}

	// 自适应超时设置
	timeout := r.Config.Timeout
	if timeout <= 0 {
		timeout = 10 // 默认10秒
	} else if timeout > 30 {
		timeout = 30 // 最大30秒
	}

	// 处理单个URL
	result, err := ProcessURL(target, r.Config.Proxy, timeout, workerCount)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// runScan 执行扫描过程
func (r *Runner) runScan(targets []string, options *types.CmdOptions) {
	var outputMutex sync.Mutex

	// 创建完成通道和进度条
	doneChan := make(chan struct{}, len(targets))
	bar := output.CreateProgressBar(len(targets))

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

	// 计算合适的URL池大小，根据目标数量动态调整
	poolSize := r.Config.URLWorkerCount
	if poolSize > len(targets) {
		poolSize = len(targets)
	}

	// 增加每批处理的数量，减少调度开销
	batchSize := 50
	if len(targets) < 100 {
		batchSize = 1 // 少量目标不使用批处理
	}

	// 创建URL处理工作池
	urlPool, _ := ants.NewPool(poolSize,
		ants.WithPreAlloc(true),
		ants.WithNonblocking(false),
		ants.WithMaxBlockingTasks(len(targets)),
	)
	defer urlPool.Release()

	var urlWg sync.WaitGroup

	// 分批次提交任务
	for i := 0; i < len(targets); i += batchSize {
		end := i + batchSize
		if end > len(targets) {
			end = len(targets)
		}

		// 对每一批的目标提交任务
		for j := i; j < end; j++ {
			target := targets[j]
			urlWg.Add(1)

			// 提交任务
			_ = urlPool.Submit(func() {
				defer urlWg.Done()

				// 扫描单个目标
				targetResult, _ := r.ScanTarget(target)

				// 线程安全地存储结果
				if targetResult != nil {
					r.mutex.Lock()
					r.Results[target] = targetResult
					r.mutex.Unlock()
				}

				// 将结果写入文件并显示结果
				handleMatchResults(targetResult, options, saveResult, r.Config.OutputFormat)

				// 通知完成一个任务
				doneChan <- struct{}{}
			})
		}

		// 如果批次较大，给系统一点喘息的机会
		if batchSize > 5 && i+batchSize < len(targets) {
			time.Sleep(1 * time.Millisecond)
		}
	}

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
}
