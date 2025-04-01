/*
  - Package main
    @Author: zhizhuo
    @IDE：GoLand
    @File: main.go
    @Date: 2025/3/10 下午2:11*
*/
package main

import (
	"fmt"
	"gxx/cmd/cli"
	"gxx/utils/logger"
	"gxx/utils/output"
	"os"
	"time"

	"github.com/fatih/color"
)

func main() {
	color.Green(cli.Banner)
	options, err := cli.NewCmdOptions()
	if err != nil {
		// 在初始化logger之前的错误使用默认logger
		logger.Error(err.Error())
		os.Exit(1)
	}

	// 配置日志级别
	logLevel := 1 // 默认INFO级别
	if options.Debug {
		logLevel = 4 // DEBUG级别
		color.Blue("DEBUG模式已开启")
	}
	// 初始化日志系统，设置日志保存目录、最大文件数和日志级别
	logger.InitLogger("logs", 5, logLevel)
	// 配置输出文件啊
	if options.Output == "" {
		options.Output = "result_" + fmt.Sprintf("%d", time.Now().Unix()) + ".txt"
	}

	logger.Info(fmt.Sprintf("输出文件：%s", options.Output))
	// 初始化输出文件
	if err := output.InitOutput(options.Output, options.OutputFormat); err != nil {
		logger.Error("初始化输出文件失败: %v", err)
		os.Exit(1)
	}
	defer output.Close()

	// 执行主程序
	cli.Run(options)

	// 确保所有日志都被写入
	time.Sleep(2 * time.Second)
	logger.Success("扫描完成")
}
