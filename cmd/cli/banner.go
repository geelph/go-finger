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
const defaultBuildDate = "2025-03-10" // 默认构建日期

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
	buildDate := getEnvOrDefault("GXX_BUILD_DATE", defaultBuildDate)
	// 如果是在构建时，使用当前日期
	if buildDate == "BUILD_TIME" {
		return time.Now().Format("2006-01-02")
	}
	return buildDate
}
