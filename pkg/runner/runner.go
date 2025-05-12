package runner

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"gxx/types"
	"gxx/utils/logger"
	"gxx/utils/output"
	"sync"
	"time"
)

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

	fingerWorkerCount := 5 * urlWorkerCount

	// 限制范围在 500 到 1000
	if fingerWorkerCount < 500 {
		fingerWorkerCount = 500
	} else if fingerWorkerCount > 1000 {
		fingerWorkerCount = 1000
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

	logger.Info(fmt.Sprintf("开始扫描 %d 个目标，使用 %d 个URL并发线程, %d 个规则并发线程...",
		len(targets), r.Config.URLWorkerCount, r.Config.FingerWorkerCount))

	// 执行扫描
	r.runScan(targets, options)

	r.mutex.Lock()
	// 清楚所有缓存文件
	ClearAllCache()
	r.mutex.Unlock()
	printSummary(targets, r.Results)

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

	// 提交所有目标到线程池
	targetsCount := len(targets)
	for i, target := range targets {
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
			// 可能是池满了，暂停一下
			if i < targetsCount-1 {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	// 等待所有URL处理完成
	urlWg.Wait()
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
