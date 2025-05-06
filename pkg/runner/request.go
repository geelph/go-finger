package runner

import (
	"bytes"
	"context"
	"fmt"
	"gxx/pkg/finger"
	"gxx/pkg/network"
	"gxx/pkg/wappalyzer"
	"gxx/types"
	"gxx/utils/common"
	"gxx/utils/logger"
	"gxx/utils/proto"
	"io"
	"net/http"
	"strings"
	"time"
)

// initializeCache 初始化请求响应缓存
func initializeCache(httpResp *http.Response, proxy string) (*proto.Response, *proto.Request) {
	if httpResp == nil {
		return nil, nil
	}

	// 读取响应体
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		logger.Debug(fmt.Sprintf("读取响应体出错: %v", err))
		respBody = []byte{}
	}
	// 关闭原始响应体并重置
	_ = httpResp.Body.Close()
	// 重要：重置响应体以供后续使用
	httpResp.Body = io.NopCloser(bytes.NewReader(respBody))

	utf8RespBody := common.Str2UTF8(string(respBody))

	// 构建响应对象
	initialResponse := finger.BuildProtoResponse(httpResp, utf8RespBody, 0, proxy)

	// 构建请求对象
	reqMethod := "GET"
	reqPath := "/"
	initialRequest := finger.BuildProtoRequest(httpResp, reqMethod, "", reqPath)

	return initialResponse, initialRequest
}

// prepareRequest 确保目标地址包含适当的协议前缀
func prepareRequest(target string) (*http.Request, error) {
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}

	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return nil, fmt.Errorf("创建临时请求失败: %v", err)
	}
	return req, nil
}

// GetBaseInfo 获取目标的基础信息并返回 BaseInfoResponse 结构体
func GetBaseInfo(target, proxy string, timeout int) (*BaseInfoResponse, error) {
	// 检查并规范化URL协议
	if checkedURL, err := network.CheckProtocol(target, proxy); err == nil && checkedURL != "" {
		target = checkedURL
	}
	logger.Debug(fmt.Sprintf("请求协议修正后url: %s", target))
	// 二次验证URL
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}
	// 设置超时时间
	timeoutDuration := time.Duration(timeout) * time.Second
	if timeout <= 0 {
		timeoutDuration = 5 * time.Second
	}

	// 创建请求选项
	options := network.OptionsRequest{
		Proxy:              proxy,
		Timeout:            timeoutDuration,
		Retries:            3,
		FollowRedirects:    true,
		InsecureSkipVerify: true,
	}

	// 发送请求
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	resp, err := network.SendRequestHttp(ctx, "GET", target, "", options)
	if err != nil {
		return &BaseInfoResponse{
			Url:        target,
			Title:      "",
			Server:     types.EmptyServerInfo(),
			StatusCode: 0,
			Response:   resp,
			Wappalyzer: nil,
		}, fmt.Errorf("发送请求失败: %v", err)
	}

	// 提取基本信息
	statusCode := int32(resp.StatusCode)
	title := finger.GetTitle(target, resp)
	serverInfo := finger.GetServerInfoFromResponse(resp)

	// 获取站点技术信息
	wapp, err := wappalyzer.NewWappalyzer()
	if err != nil {
		// 即使获取站点技术信息失败，仍然返回基本信息
		return &BaseInfoResponse{
			Url:        target,
			Title:      title,
			Server:     serverInfo,
			StatusCode: statusCode,
			Response:   resp,
			Wappalyzer: nil,
		}, nil
	}

	// 读取响应体并重置
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Debug(fmt.Sprintf("读取响应体出错: %v", err))
		data = []byte{}
	}
	// 重置响应体以供后续使用
	resp.Body = io.NopCloser(bytes.NewReader(data))

	wappData, err := wapp.GetWappalyzer(resp.Header, data)
	if err != nil {
		// 即使获取Wappalyzer数据失败，仍然返回基本信息
		return &BaseInfoResponse{
			Url:        target,
			Title:      title,
			Server:     serverInfo,
			StatusCode: statusCode,
			Response:   resp,
			Wappalyzer: nil,
		}, nil
	}

	logger.Debug(fmt.Sprintf("当前站点使用技术：%s", wappData))

	return &BaseInfoResponse{
		Url:        target,
		Title:      title,
		Server:     serverInfo,
		StatusCode: statusCode,
		Response:   resp,
		Wappalyzer: wappData,
	}, nil
}
