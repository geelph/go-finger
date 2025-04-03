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
	Target       goflags.StringSlice // 测试目标
	TargetsFile  string              // 测试目标文件
	Threads      int                 // 并发线程数
	Output       string              // 输出文件路径
	PocFile      string              // POC文件路径
	PocYaml      string              // 单个POC yaml文件
	Timeout      int                 // 超时时间
	Retries      int                 // 重试次数，默认3次
	Proxy        string              // 代理地址
	Debug        bool                // 设置debug模式
}
