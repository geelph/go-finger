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
		flagSet.StringSliceVarP(&options.Target, "url", "u", nil, "要扫描的目标URL/主机", goflags.NormalizedStringSliceOptions),
		flagSet.StringVarP(&options.TargetsFile, "file", "f", "", "要扫描的目标URL/主机列表（每行一个）"),
		flagSet.IntVarP(&options.Threads, "threads", "t", 10, "并发线程数"),
	)
	flagSet.CreateGroup("output", "输出",
		flagSet.StringVarP(&options.Output, "output", "o", "", "输出文件路径（支持txt/csv格式）"),
		flagSet.StringVarP(&options.OutputFormat, "format", "fmt", "txt", "输出文件格式（支持txt/csv）"),
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
	// 使用反射自动序列化命令行选项用于调试
	//optionsStr := fmt.Sprintf("%+v", *opt)
	//fmt.Println("命令行选项：", optionsStr)

	// 验证目标输入
	if len(opt.Target) == 0 && opt.TargetsFile == "" {
		return fmt.Errorf("必须设置 `-url` 或 `-file` 参数指定扫描目标")
	}

	// 验证输出格式
	if opt.Output != "" && opt.OutputFormat != "txt" && opt.OutputFormat != "csv" {
		return fmt.Errorf("输出格式只支持 txt 或 csv")
	}

	// 验证线程数
	if opt.Threads <= 0 {
		logger.Warn("线程数无效，将使用默认值 10")
		opt.Threads = 10
	}

	return nil
}
