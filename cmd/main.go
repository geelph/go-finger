/*
  - Package main
    @Author: zhizhuo
    @IDE：GoLand
    @File: main.go
    @Date: 2025/3/10 下午2:11*
*/
package main

import (
	"gxx/cmd/cli"
	"gxx/utils/logger"
	"os"
	"time"

	"github.com/fatih/color"
)

func main() {
	color.Green(cli.Banner)
	logger.InitLogger("logs", 5, 1)
	options, err := cli.NewCmdOptions()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(0)
	}
	if options.Debug {
		logger.InitLogger("debug", 5, 1)
	}
	cli.Run(options)
	time.Sleep(time.Second * 2)
}
