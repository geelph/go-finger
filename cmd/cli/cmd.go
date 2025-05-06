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
	r := runner.NewRunner(options)
	// 运行扫描
	if err := r.Run(options); err != nil {
		// 错误已在Run函数内部记录，这里无需额外处理
		return
	}
}
