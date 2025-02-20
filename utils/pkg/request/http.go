/*
  - Package request
    @Author: zhizhuo
    @IDE：GoLand
    @File: http.go
    @Date: 2025/2/20 下午4:12*
*/
package request

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"gxx/utils/common"
	"gxx/utils/logger"
	"gxx/utils/pkg/proto"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/projectdiscovery/retryablehttp-go"
	"golang.org/x/net/context"
)

// 全局客户端配置
var (
	RetryClient    *retryablehttp.Client                    // 可处理重定向的客户端
	maxDefaultBody int64                 = 10 * 1024 * 1024 // 最大读取响应体限制（10MB）
	defaultTimeout                       = 10 * time.Second // 默认请求超时时间
)

// 定义http通信协议
var (
	HttpPrefix  = "http://"
	HttpsPrefix = "https://"
)

// OptionsRequest 请求配置参数结构体
type OptionsRequest struct {
	Proxy              string            // 代理地址，格式：scheme://host:port
	Timeout            time.Duration     // 请求超时时间（默认5秒）
	Retries            int               // 最大重试次数（默认3次）
	FollowRedirects    bool              // 是否跟随重定向（默认true）
	InsecureSkipVerify bool              // 是否跳过SSL证书验证（默认true）
	CustomHeaders      map[string]string // 自定义请求头
}

// init 包初始化函数
func init() {
	initGlobalClient()
}

// initGlobalClient 初始化全局客户端实例
func initGlobalClient() {
	opts := retryablehttp.DefaultOptionsSingle
	RetryClient = retryablehttp.NewClient(opts)
}

// NewRequestHttp 创建并发送HTTP请求
func NewRequestHttp(urlStr string, options OptionsRequest) (*http.Response, error) {
	setDefaults(&options)
	if options.Proxy != "" {
		logger.Debug("使用代理 ", options.Proxy)
	}

	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	configureHeaders(req, options)

	client := configureClient(options)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// SendRequestHttp yaml poc or 指纹 yaml 构建发送http请求
func SendRequestHttp(Method string, UrlStr string, Body string, options OptionsRequest) (*http.Response, error) {
	setDefaults(&options)
	if options.Proxy != "" {
		logger.Debug("使用代理 ", options.Proxy)
	}

	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	req, err := retryablehttp.NewRequestWithContext(ctx, Method, UrlStr, Body)
	if err != nil {
		return nil, err
	}
	configureHeaders(req, options)

	client := configureClient(options)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// setDefaults 设置配置参数的默认值
func setDefaults(options *OptionsRequest) {
	if options.Timeout == 0 {
		options.Timeout = 5 * time.Second
	}

	if options.Retries == 0 {
		options.Retries = 3
	}
}

// configureHeaders 配置请求头信息
func configureHeaders(req *retryablehttp.Request, options OptionsRequest) {
	req.Header.Set("User-Agent", common.RandomUA())
	// default post content-type
	if req.Method == http.MethodPost && len(req.Header.Get("Content-Type")) == 0 {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}
	// 添加自定义headers
	for key, value := range options.CustomHeaders {
		req.Header.Set(key, value)
	}
}

// configureClient 配置HTTP客户端参数
func configureClient(options OptionsRequest) *retryablehttp.Client {
	client := RetryClient
	client.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !options.FollowRedirects {
			return http.ErrUseLastResponse // 禁止重定向
		}
		return nil
	}

	client.HTTPClient.Transport = &http.Transport{
		Proxy: getProxyFunc(options.Proxy),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: options.InsecureSkipVerify,
		},
	}
	client.HTTPClient.Timeout = options.Timeout
	return client
}

// ReverseGet 发送GET请求并返回响应内容
func ReverseGet(target string) ([]byte, error) {
	if target == "" {
		return nil, errors.New("目标地址不能为空")
	}

	body, _, err := simpleRetryHttpGet(target)
	return body, err
}

// simpleRetryHttpGet 简化版HTTP GET请求实现
func simpleRetryHttpGet(target string) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("User-Agent", common.RandomUA())

	resp, err := RetryClient.Do(req)
	if err != nil {
		if resp != nil {
			defer func(Body io.ReadCloser) {
				_ = Body.Close()
			}(resp.Body)
		}
		return nil, 0, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	reader := io.LimitReader(resp.Body, maxDefaultBody)
	respBody, err := io.ReadAll(reader)
	if err != nil {
		return nil, 0, err
	}

	return respBody, resp.StatusCode, nil
}

// getProxyFunc 获取代理配置函数
func getProxyFunc(proxyURL string) func(*http.Request) (*url.URL, error) {
	if proxyURL == "" {
		return nil
	}
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		logger.Error("代理地址解析失败:", err)
		return nil
	}
	return http.ProxyURL(parsedURL)
}

// CheckProtocol 检查网络通信协议
func CheckProtocol(host string) (string, error) {
	var (
		err       error
		result    string
		parsePort string
	)

	if len(strings.TrimSpace(host)) == 0 {
		return result, fmt.Errorf("host %q is empty", host)
	}

	if strings.HasPrefix(host, HttpsPrefix) {
		_, _, err := simpleRetryHttpGet(host)
		if err != nil {
			return result, err
		}

		return host, nil
	}

	if strings.HasPrefix(host, HttpPrefix) {
		_, _, err := simpleRetryHttpGet(host)
		if err != nil {
			return result, err
		}

		return host, nil
	}

	u, err := url.Parse(HttpPrefix + host)
	if err != nil {
		return result, err
	}
	parsePort = u.Port()

	switch {
	case parsePort == "80":
		_, _, err := simpleRetryHttpGet(HttpPrefix + host)
		if err != nil {
			return result, err
		}

		return HttpPrefix + host, nil

	case parsePort == "443":
		_, _, err := simpleRetryHttpGet(HttpsPrefix + host)
		if err != nil {
			return result, err
		}

		return HttpsPrefix + host, nil

	default:
		_, _, err := simpleRetryHttpGet(HttpsPrefix + host)
		if err == nil {
			return HttpsPrefix + host, err
		}

		body, _, err := simpleRetryHttpGet(HttpPrefix + host)
		if err == nil {
			if strings.Contains(string(body), "<title>400 The plain HTTP request was sent to HTTPS port</title>") {
				return HttpsPrefix + host, nil
			}
			return HttpPrefix + host, nil
		}

	}

	return "", fmt.Errorf("host %q is empty", host)
}

// Url2ProtoUrl 参数定义
func Url2ProtoUrl(u *url.URL) *proto.UrlType {
	return &proto.UrlType{
		Scheme:   u.Scheme,
		Domain:   u.Hostname(),
		Host:     u.Host,
		Port:     u.Port(),
		Path:     u.EscapedPath(),
		Query:    u.RawQuery,
		Fragment: u.Fragment,
	}
}

// ParseRequest 解析请求raw数据包
func ParseRequest(oReq *http.Request) (*proto.Request, error) {
	req := &proto.Request{}
	req.Method = oReq.Method
	req.Url = common.Url2UrlType(oReq.URL)
	header := make(map[string]string)
	for k := range oReq.Header {
		header[k] = oReq.Header.Get(k)
	}
	req.Headers = header
	req.ContentType = oReq.Header.Get("Content-Type")
	if oReq.Body == nil || oReq.Body == http.NoBody {
	} else {
		data, err := ioutil.ReadAll(oReq.Body)
		if err != nil {
			return nil, err
		}
		req.Body = data
		oReq.Body = ioutil.NopCloser(bytes.NewBuffer(data))
	}
	return req, nil
}
