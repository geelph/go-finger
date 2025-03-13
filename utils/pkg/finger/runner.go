/*
  - Package finger
    @Author: zhizhuo
    @IDE：GoLand
    @File: runner.go
    @Date: 2025/2/20 下午3:37*
*/
package finger

import (
	"fmt"
	"golang.org/x/net/context"
	"gxx/utils/common"
	"gxx/utils/logger"
	request2 "gxx/utils/request"
	"io"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
)

var (
	maxDefaultBody int64 = 5 * 1024 * 1024  // 最大读取响应体限制（5MB）
	defaultTimeout       = 10 * time.Second // 默认请求超时时间
)

// SendRequest yaml poc发送http请求
func SendRequest(target string, req RuleRequest, rule Rule, variableMap map[string]any, proxy string) (map[string]any, error) {

	options := request2.OptionsRequest{
		Proxy:              "",             // 初始化为空，后面设置
		Timeout:            defaultTimeout, // 增加超时时间到10秒
		Retries:            2,              // 增加重试次数
		FollowRedirects:    !rule.Request.FollowRedirects,
		InsecureSkipVerify: true, // 忽略SSL证书错误
		CustomHeaders:      map[string]string{},
	}
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel() // 在读取完响应后取消

	// 设置代理地址
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			fmt.Println("代理地址解析失败:", err)
		} else {
			options.Proxy = proxyURL.String()
		}
	}

	// 处理path
	rule.Request.Path = SetVariableMap(strings.TrimSpace(rule.Request.Path), variableMap)
	newPath := formatPath(rule.Request.Path)

	// 处理url
	urlStr := common.ParseTarget(target, newPath)

	// 处理body
	rule.Request.Body = formatBody(rule.Request.Body, rule.Request.Headers["Content-Type"], variableMap)

	// 处理自定义headers
	for k, v := range rule.Request.Headers {
		options.CustomHeaders[k] = v
	}

	// 判断请求方式
	reqType := strings.ToLower(rule.Request.Type)
	if len(reqType) > 0 && reqType != common.HTTP_Type {
		switch reqType {
		case common.TCP_Type:
			rule.Request.Host = SetVariableMap(rule.Request.Host, variableMap)
			info, err := common.ParseAddress(rule.Request.Host)
			if err != nil {
				return nil, fmt.Errorf("Error parsing address: %v\n", err)
			}
			nc, err := request2.NewTcpClient(rule.Request.Host, request2.TcpOrUdpConfig{
				Network:     rule.Request.Type,
				ReadTimeout: time.Duration(rule.Request.ReadTimeout),
				ReadSize:    rule.Request.ReadSize,
				MaxRetries:  1,
				ProxyURL:    options.Proxy,
				IsLts:       info.IsLts,
				ServerName:  info.Host,
			})
			if err != nil {
				fmt.Println("tcp error:", err.Error())
				return nil, err
			}
			data := rule.Request.Data

			if len(rule.Request.DataType) > 0 {
				dataType := strings.ToLower(rule.Request.DataType)
				if dataType == "hex" {
					data = common.FromHex(data)
				}
			}
			errs := nc.Send([]byte(data))
			if errs != nil {
				fmt.Println("tcp send error:", errs.Error())
			}
			res, err := nc.RecvTcp()
			if err != nil {
				fmt.Println("tcp receive error:", err.Error())
			}
			_ = nc.Close()
			err = request2.RawParse(nc, []byte(data), res, variableMap)
			if err != nil {
				fmt.Println("tcp or udp parse error:", err.Error())
			}
			return variableMap, nil
		case common.UDP_Type:
			rule.Request.Host = SetVariableMap(rule.Request.Host, variableMap)
			info, err := common.ParseAddress(rule.Request.Host)
			if err != nil {
				return nil, fmt.Errorf("Error parsing address: %v\n", err)
			}
			nc, err := request2.NewUdpClient(rule.Request.Host, request2.TcpOrUdpConfig{
				Network:     rule.Request.Type,
				ReadTimeout: time.Duration(rule.Request.ReadTimeout),
				ReadSize:    rule.Request.ReadSize,
				MaxRetries:  1,
				ProxyURL:    options.Proxy,
				IsLts:       info.IsLts,
				ServerName:  info.Host,
			})
			if err != nil {
				fmt.Println("udp error:", err.Error())
				return nil, err
			}
			data := rule.Request.Data

			if len(rule.Request.DataType) > 0 {
				dataType := strings.ToLower(rule.Request.DataType)
				if dataType == "hex" {
					data = common.FromHex(data)
				}
			}
			errs := nc.Send([]byte(data))
			if errs != nil {
				fmt.Println("udp send error:", errs.Error())
			}
			res, err := nc.RecvTcp()
			if err != nil {
				fmt.Println("udp receive error:", err.Error())
			}
			_ = nc.Close()
			err = request2.RawParse(nc, []byte(data), res, variableMap)
			if err != nil {
				fmt.Println("udp or udp parse error:", err.Error())
			}
			return variableMap, nil
		case common.GO_Type:
			fmt.Println("执行go模块调用发送请求，当前模块未完成")
			return nil, fmt.Errorf("go module not implemented")
		}
	} else {
		if len(rule.Request.Raw) > 0 {
			// 执行raw格式请求
			fmt.Println("执行raw格式请求")
			rt := request2.RawHttp{RawhttpClient: request2.GetRawHTTP(int(options.Timeout))}
			err := rt.RawHttpRequest(rule.Request.Raw, target, variableMap)
			if err != nil {
				return variableMap, err
			}
		}
	}

	// 处理协议，增加通信协议
	NewUrlStr, err := request2.CheckProtocol(urlStr)
	if err != nil {
		fmt.Println("检查http通信协议出错，错误信息：", err)
		if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
			NewUrlStr = "http://" + target
		}
	}

	logger.Debug("请求URL：", NewUrlStr)

	// 发送请求
	resp, err := request2.SendRequestHttp(ctx, req.Method, NewUrlStr, rule.Request.Body, options)
	if err != nil {
		fmt.Println("发送请求出错，错误信息：", err)
		return variableMap, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	// 处理请求的raw
	protoReq := buildProtoRequest(resp, rule.Request.Method, rule.Request.Body, rule.Request.Path)
	variableMap["request"] = protoReq

	// 读取响应体
	reader := io.LimitReader(resp.Body, maxDefaultBody)
	body, err := io.ReadAll(reader)
	if err != nil {
		fmt.Println("读取响应体出错:", err)
		// 即使读取响应体出错，也继续处理，使用空响应体
		body = []byte{}
	}
	utf8RespBody := common.Str2UTF8(string(body))

	// 计算响应时间
	var milliseconds int64
	start := time.Now()
	trace := httptrace.ClientTrace{}
	trace.GotFirstResponseByte = func() {
		milliseconds = time.Since(start).Nanoseconds() / 1e6
	}
	// 处理响应的raw，传入代理参数
	protoResp := buildProtoResponse(resp, utf8RespBody, milliseconds, proxy)
	variableMap["response"] = protoResp
	return variableMap, nil
}
