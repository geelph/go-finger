/*
  - Package cli
    @Author: zhizhuo
    @IDE：GoLand
    @File: cmd.go
    @Date: 2025/2/20 下午3:32*
*/
package cli

import (
	"gxx/pkg"
	"gxx/types"
)

// Run 执行指纹识别
func Run(options *types.CmdOptions) {
	pkg.NewFingerRunner(options)
}
