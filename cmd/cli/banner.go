/*
  - Package cmd
    @Author: zhizhuo
    @IDE：GoLand
    @File: banner.go
    @Date: 2025/2/20 下午3:39*
*/
package cli

import (
	"os"
	"strings"
	"time"
)

// 默认版本信息
const defaultVersion = "1.0.0"
const defaultAuthor = "zhizhuo"

// 使用 var 定义 defaultBuildDate，使其可以在编译时通过环境变量注入
var defaultBuildDate string

func init() {
	// 在包初始化时设置 defaultBuildDate
	if value, exists := os.LookupEnv("BUILD_DATE"); exists && value != "" {
		defaultBuildDate = value
	} else {
		defaultBuildDate = "BUILD_TIME"
	}
}

// 从环境变量获取值，如果不存在则使用默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

// Banner 程序启动时显示的横幅
var Banner = strings.TrimSpace(`
 _______  ___  _
/  __\  \/\  \//
| |  _\  / \  / 
| |_///  \ /  \ 
\____/__/\/__/\\`) + "\n\n\t\tVersion: " + getEnvOrDefault("GXX_VERSION", defaultVersion) +
	"\n\t\tAuthor: " + getEnvOrDefault("GXX_AUTHOR", defaultAuthor) +
	"\n\t\tBuild: " + getBuildDate()

// getBuildDate 获取构建日期
func getBuildDate() string {
	// 优先从环境变量获取
	if value, exists := os.LookupEnv("GXX_BUILD_DATE"); exists && value != "" {
		return value
	}
	
	// 如果是默认的 BUILD_TIME 或为空，使用当前日期
	if defaultBuildDate == "BUILD_TIME" || defaultBuildDate == "" {
		return time.Now().Format("2006-01-02")
	}
	
	return defaultBuildDate
}
