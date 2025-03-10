/*
  - Package cmd
    @Author: zhizhuo
    @IDE：GoLand
    @File: cmd.go
    @Date: 2025/2/20 下午3:32*
*/
package cmd

import (
	"fmt"
	"gxx/utils"
	"gxx/utils/logger"
	"os"
	"time"

	"github.com/fatih/color"
)

func Run() {
	color.Green(Banner)
	logger.InitLogger("logs", 5, 1)
	options, err := NewCmdOptions()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(0)
	}
	fmt.Println("cmd: ", options)
	utils.NewFingerRunner(options)
	time.Sleep(time.Second * 2)
}
