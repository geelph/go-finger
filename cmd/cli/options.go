/*
  - Package cmd
    @Author: zhizhuo
    @IDE：GoLand
    @File: options.go
    @Date: 2025/2/20 下午3:39*
*/
package cli

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
	flagSet.CreateGroup("input", "目标",
		flagSet.StringSliceVarP(&options.Target, "target", "t", nil, "要扫描的目标URL/主机", goflags.NormalizedStringSliceOptions),
		flagSet.StringVarP(&options.TargetsFile, "file", "f", "", "要扫描的目标URL/主机列表（每行一个）"),
	)
	flagSet.CreateGroup("output", "输出",
		flagSet.StringVarP(&options.Output, "output", "o", "", "输出文件类型支持txt/csv"),
	)
	flagSet.CreateGroup("debug", "调试",
		flagSet.StringVar(&options.Proxy, "proxy", "", "要使用的http/socks5代理列表（逗号分隔或文件输入）"),
		flagSet.StringVar(&options.PocYaml, "p", "", "测试单个的yaml文件"),
		flagSet.StringVar(&options.PocFile, "pf", "", "测试指定目录下面所有的yaml文件"),
		flagSet.BoolVar(&options.Debug, "debug", false, "是否开启debug模式，默认关闭"),
	)

	// 实例化操作
	if err := flagSet.Parse(); err != nil {
		logger.Error("无法解析标志: %s\n", err)
	}
	// 验证必参数是否传入
	if err := verifyOptions(options); err != nil {
		return options, err
	}

	return options, nil
}

// verifyOptions 验证命令行选项
func verifyOptions(opt *types.CmdOptions) error {
	fmt.Println("命令行选项：", opt)
	if len(opt.Target) == 0 && len(opt.TargetsFile) == 0 {
		return fmt.Errorf("必须设置 `-target` 或 `-file`")
	}
	return nil
}
