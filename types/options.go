/*
  - Package types
    @Author: zhizhuo
    @IDE：GoLand
    @File: options.go
    @Date: 2025/3/10 下午2:39*
*/
package types

import (
	"github.com/projectdiscovery/goflags"
)

// CmdOptions 命令行选项结构体
type CmdOptions struct {
	Target      goflags.StringSlice // 测试目标
	TargetsFile string              // 测试目标文件
	PocFile     string              // POC文件路径
	Timeout     int                 // 超时时间
	Retries     int                 // 重试次数，默认3次
	Output      string              //输出位置
	Proxy       string              // 代理地址
}
