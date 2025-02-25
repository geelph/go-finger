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
	"gxx/utils/common"
	"gxx/utils/pkg/request"
	"io"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
)

// SendRequest yaml poc发送http请求
func SendRequest(target string, req RuleRequest, rule Rule, variableMap map[string]any, proxy string) (map[string]any, error) {
	options := request.OptionsRequest{
		Proxy:              "",              // 不使用代理
		Timeout:            5 * time.Second, // 使用默认超时时间5s
		Retries:            1,               // 默认重试次数
		FollowRedirects:    !rule.Request.FollowRedirects,
		InsecureSkipVerify: false, // 使用默认值（忽略 SSL 证书错误）
		CustomHeaders:      map[string]string{},
	}

	// 设置代理地址
	proxyURL, err := url.Parse(proxy)
	if err != nil {
		return nil, err
	}
	options.Proxy = proxyURL.String()

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
			nc, err := request.NewTcpClient(rule.Request.Host, request.TcpOrUdpConfig{
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
			nc.Close()
			err = request.RawParse(nc, []byte(data), res, variableMap)
			if err != nil {
				fmt.Println("tcp or udp parse error:", err.Error())
			}
			return variableMap, nil
		case common.UDP_Type:
			fmt.Println("执行udp请求，当前模块未完成")
			rule.Request.Host = SetVariableMap(rule.Request.Host, variableMap)
			info, err := common.ParseAddress(rule.Request.Host)
			if err != nil {
				return nil, fmt.Errorf("Error parsing address: %v\n", err)
			}
			nc, err := request.NewUdpClient(rule.Request.Host, request.TcpOrUdpConfig{
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
			nc.Close()
			err = request.RawParse(nc, []byte(data), res, variableMap)
			if err != nil {
				fmt.Println("udp or udp parse error:", err.Error())
			}
			return variableMap, nil
		case common.GO_Type:
			fmt.Println("执行go模块调用发送请求，当前模块未完成")
			return nil, err
		}
	} else {
		if len(rule.Request.Raw) > 0 {
			// 执行raw格式请求
			rt := request.RawHttp{RawhttpClient: request.GetRawHTTP(int(options.Timeout))}
			err = rt.RawHttpRequest(rule.Request.Raw, target, variableMap)
		}
		// 继续走下面代码执行
	}

	// 处理协议，增加通信协议
	NewUrlStr, err := request.CheckProtocol(urlStr)
	if err != nil {
		fmt.Println("检查http通信协议出错，错误信息：", err)
		if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
			NewUrlStr = "http://" + target
		}
	}

	fmt.Println("请求URL：", NewUrlStr)

	// 发送请求
	resp, err := request.SendRequestHttp(req.Method, NewUrlStr, rule.Request.Body, options)
	if err != nil {
		fmt.Println("发送请求出错，错误信息：", err)
	}
	defer resp.Body.Close()

	// 处理请求的raw
	protoReq := buildProtoRequest(resp, rule.Request.Method, rule.Request.Body, rule.Request.Path)
	variableMap["request"] = protoReq
	variableMap["request"] = protoReq

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应体出错:", err)
	}
	utf8RespBody := common.Str2UTF8(string(body))

	// 计算响应时间
	var milliseconds int64
	start := time.Now()
	trace := httptrace.ClientTrace{}
	trace.GotFirstResponseByte = func() {
		milliseconds = time.Since(start).Nanoseconds() / 1e6
	}
	// 处理响应的raw
	protoResp := buildProtoResponse(resp, utf8RespBody, milliseconds)
	variableMap["response"] = protoResp
	return variableMap, nil
}
