/*
  - Package cli
    @Author: zhizhuo
    @IDE：GoLand
    @File: cmd.go
    @Date: 2025/2/20 下午3:32*
*/
package cli

import (
	"gxx/pkg/runner"
	"gxx/types"
)

// Run 执行指纹识别
func Run(options *types.CmdOptions) {
	// 开启内存监控
	runner.StartMemoryMonitor()
	// 停止内存监控，延时调用，后进先出
	defer runner.StopMemoryMonitor()
	// 声明一个新的Runner
	r := runner.NewRunner(options)
	// 运行扫描
	if err := r.Run(options); err != nil {
		// 错误已在Run函数内部记录，这里无需额外处理
		return
	}
}
