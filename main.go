/*
  - Package gxx
    @Author: zhizhuo
    @IDE：GoLand
    @File: main.go
    @Date: 2025/3/10 下午2:11*
*/
package gxx

import (
	"gxx/cmd/cli"
	"gxx/types"
)

// FingerScan 指纹扫描 API
func FingerScan(options *types.CmdOptions) {
	cli.Run(options)
}

// NewFingerOptions 创建新的指纹扫描选项
func NewFingerOptions() (*types.CmdOptions, error) {
	return cli.NewCmdOptions()
}
