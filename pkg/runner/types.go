/*
Package runner 提供指纹识别的运行时功能
@Author: zhizhuo
*/
package runner

import (
	"gxx/pkg/finger"
	"gxx/pkg/wappalyzer"
	"gxx/types"
	"gxx/utils/proto"
	"net/http"
)

// BaseInfoResponse 包含目标基础信息和HTTP响应
type BaseInfoResponse struct {
	Url        string
	Title      string
	Server     *types.ServerInfo
	StatusCode int32
	Response   *http.Response
	Wappalyzer *wappalyzer.TypeWappalyzer
}

// TargetResult 存储每个目标的扫描结果
type TargetResult struct {
	URL          string                     // 目标地址
	StatusCode   int32                      // 状态码
	Title        string                     // 站点标题
	Server       *types.ServerInfo          // server信息
	Matches      []*FingerMatch             // 匹配信息
	Wappalyzer   *wappalyzer.TypeWappalyzer // 站点信息数据
	LastRequest  *proto.Request             // 该URL的请求缓存
	LastResponse *proto.Response            // 该URL的响应缓存
}

// FingerMatch 存储每个匹配的指纹信息
type FingerMatch struct {
	Finger   *finger.Finger  // 指纹信息
	Result   bool            // 识别结果
	Request  *proto.Request  // 请求数据
	Response *proto.Response // 响应数据
}

// BaseInfo 存储目标的基础信息
type BaseInfo struct {
	Title      string
	Server     *types.ServerInfo
	StatusCode int32
}

// ScanConfig 存储扫描配置参数
type ScanConfig struct {
	Proxy             string
	Timeout           int
	URLWorkerCount    int
	FingerWorkerCount int
	OutputFormat      string
	OutputFile        string
	SockOutputFile    string
}
