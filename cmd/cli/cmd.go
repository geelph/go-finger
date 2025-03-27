/*
  - Package cli
    @Author: zhizhuo
    @IDE：GoLand
    @File: cmd.go
    @Date: 2025/2/20 下午3:32*
*/
package cli

import (
	"gxx/types"
	"gxx/utils"
)

// Run 执行指纹识别
func Run(options *types.CmdOptions) {
	utils.NewFingerRunner(options)
}
