/*
  - Package finger
    @Author: zhizhuo
    @IDE：GoLand
    @File: req.go
    @Date: 2025/2/21 下午3:06*
*/
package finger

import (
	"fmt"
	"gxx/pkg/network"
	"gxx/utils/common"
	"gxx/utils/logger"
	"gxx/utils/proto"
	"net/http"
	"strings"
)

// formatPath 格式化路径
func formatPath(path string) string {
	newPath := strings.TrimSpace(path)
	if strings.HasPrefix(newPath, "^") {
		newPath = "/" + newPath[1:]
	}
	if !strings.HasPrefix(newPath, "/") {
		newPath = "/" + newPath
	}
	newPath = strings.ReplaceAll(newPath, " ", "%20")
	newPath = strings.ReplaceAll(newPath, "#", "%23")
	return newPath
}

// formatBody 格式化请求体
func formatBody(body, contentType string, variableMap map[string]any) string {
	body = SetVariableMap(strings.TrimSpace(body), variableMap)
	if strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") && strings.Contains(body, "\n\n") {
		multipartBody, err := common.DealMultipart(contentType, body)
		if err != nil {
			fmt.Println("处理multipart/form-data出错:", err)
			return body
		}
		body = SetVariableMap(strings.TrimSpace(multipartBody), variableMap)
	}
	return body
}

// buildProtoRequest 构造proto.Request结构体
func buildProtoRequest(resp *http.Response, method, body, path string) *proto.Request {
	protoReq := &proto.Request{
		Method:      method,
		Url:         network.Url2ProtoUrl(resp.Request.URL),
		ContentType: resp.Request.Header.Get("Content-Type"),
		Body:        []byte(body),
	}

	headers := make(map[string]string)
	rawReqHeaderBuilder := strings.Builder{}
	for k := range resp.Request.Header {
		headers[k] = resp.Request.Header.Get(k)
		rawReqHeaderBuilder.WriteString(k)
		rawReqHeaderBuilder.WriteString(": ")
		rawReqHeaderBuilder.WriteString(resp.Request.Header.Get(k))
		rawReqHeaderBuilder.WriteString("\n")
	}

	protoReq.Headers = headers
	protoReq.Raw = []byte(fmt.Sprintf("%s %s %s\nHost: %s\n%s\n\n%s", resp.Request.Method, path, resp.Request.Proto, resp.Request.URL.Host, strings.Trim(rawReqHeaderBuilder.String(), "\n"), body))
	protoReq.RawHeader = []byte(strings.Trim(rawReqHeaderBuilder.String(), "\n"))

	return protoReq
}

// buildProtoResponse 构造proto.Response结构体
func buildProtoResponse(resp *http.Response, utf8RespBody string, latency int64, proxy string) *proto.Response {
	headers := make(map[string]string)
	rawHeaderBuilder := strings.Builder{}
	for k := range resp.Header {
		headers[strings.ToLower(k)] = resp.Header.Get(k)
		rawHeaderBuilder.WriteString(k)
		rawHeaderBuilder.WriteString(": ")
		rawHeaderBuilder.WriteString(resp.Header.Get(k))
		rawHeaderBuilder.WriteString("\n")
	}
	// 获取icon url
	iconUrl := GetIconURL(resp.Request.URL.String(), utf8RespBody)
	// 计算icon hash，传入代理参数
	iconHash := NewGetIconHash(iconUrl, proxy).Run()
	logger.Debug(fmt.Sprintf("icon hash：%s", iconHash))
	return &proto.Response{
		Status:      int32(resp.StatusCode),
		Url:         network.Url2ProtoUrl(resp.Request.URL),
		Headers:     headers,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        []byte(utf8RespBody),
		Raw:         []byte(fmt.Sprintf("%s %s\n%s\n\n%s", resp.Proto, resp.Status, strings.Trim(rawHeaderBuilder.String(), "\n"), utf8RespBody)),
		RawHeader:   []byte(strings.Trim(rawHeaderBuilder.String(), "\n")),
		Latency:     latency,
		IconHash:    iconHash,
	}
}
