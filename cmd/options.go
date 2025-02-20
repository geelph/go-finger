/*
  - Package cmd
    @Author: zhizhuo
    @IDE：GoLand
    @File: options.go
    @Date: 2025/2/20 下午3:39*
*/
package cmd

import (
	"fmt"
	"github.com/projectdiscovery/goflags"
	"gxx/utils/logger"
)

type CmdOptions struct {
	Target      goflags.StringSlice // 测试目标
	TargetsFile string              // 测试目标文件
	PocFile     string              // POC文件路径
	Timeout     int                 // 超时时间
	Retries     int                 // 重试次数，默认3次
	Output      string              //输出位置
	Proxy       string              // 代理地址
}

func NewCmdOptions() (*CmdOptions, error) {
	options := &CmdOptions{}
	flagSet := goflags.NewFlagSet()
	flagSet.CreateGroup("input", "Target",
		flagSet.StringSliceVarP(&options.Target, "target", "t", nil, "target URLs/hosts to scan", goflags.NormalizedStringSliceOptions),
		flagSet.StringVarP(&options.TargetsFile, "file", "f", "", "list of target URLs/hosts to scan (one per line)"),
	)
	flagSet.CreateGroup("output", "Output",
		flagSet.StringVarP(&options.Output, "output", "o", "", "list of http/socks5 proxy to use (comma separated or file input)"),
	)
	flagSet.CreateGroup("debug", "Debug",
		flagSet.StringVar(&options.Proxy, "proxy", "", "list of http/socks5 proxy to use (comma separated or file input)"),
	)

	// 实例化操作
	if err := flagSet.Parse(); err != nil {
		logger.Error("Could not parse flags: %s\n", err)
	}
	// 验证必参数是否传入
	if err := options.verifyOptions(); err != nil {
		return options, err
	}

	return options, nil

}
func (opt *CmdOptions) verifyOptions() error {
	fmt.Println("Cmd Options：", opt)
	if len(opt.Target) == 0 && len(opt.TargetsFile) == 0 {
		return fmt.Errorf("either `-target` or `-file` must be set")
	}
	return nil
}
