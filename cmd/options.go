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
	"gxx/types"
	"gxx/utils/logger"

	"github.com/projectdiscovery/goflags"
)

// NewCmdOptions 创建并解析命令行选项
func NewCmdOptions() (*types.CmdOptions, error) {
	options := &types.CmdOptions{}
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
	if err := verifyOptions(options); err != nil {
		return options, err
	}

	return options, nil
}

// verifyOptions 验证命令行选项
func verifyOptions(opt *types.CmdOptions) error {
	fmt.Println("Cmd Options：", opt)
	if len(opt.Target) == 0 && len(opt.TargetsFile) == 0 {
		return fmt.Errorf("either `-target` or `-file` must be set")
	}
	return nil
}
