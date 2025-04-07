/*
  - Package request
    @Author: zhizhuo
    @IDE：GoLand
    @File: http.go
    @Date: 2025/2/20 下午4:12*
*/
package network

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"gxx/utils/common"
	"gxx/utils/logger"
	"gxx/utils/proto"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chainreactors/proxyclient"
	"github.com/zan8in/retryablehttp"
	"golang.org/x/net/context"
)

// 全局客户端配置
var (
	RetryClient    *retryablehttp.Client                    // 可处理重定向的客户端
	maxDefaultBody int64                 = 5 * 1024 * 1024  // 最大读取响应体限制（5MB）
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
		logger.Debug(fmt.Sprintf("使用代理：%s", options.Proxy))
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
func SendRequestHttp(ctx context.Context, Method string, UrlStr string, Body string, options OptionsRequest) (*http.Response, error) {
	setDefaults(&options)
	if options.Proxy != "" {
		logger.Debug(fmt.Sprintf("使用代理：%s", options.Proxy))
	}
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
	req.Header.Set("Accept", "application/x-shockwave-flash, image/gif, image/x-xbitmap, image/jpeg, image/pjpeg, application/vnd.ms-excel, application/vnd.ms-powerpoint, application/msword, */*'")
	req.Header.Set("X-Forwarded-For", common.GetRandomIP())
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")
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

	// 只有当代理地址不为空时才进行代理设置
	if options.Proxy != "" {
		parsedURL, err := url.Parse(options.Proxy)
		if err != nil {
			logger.Error("代理地址解析失败:", err)
			// 不返回nil，继续使用默认client
		} else {
			dialer, err := proxyclient.NewClient(parsedURL)
			if err != nil {
				logger.Error("创建代理客户端失败:", err)
				// 不返回nil，继续使用默认client
			} else {
				client.HTTPClient.Transport = &http.Transport{
					DialContext: dialer.DialContext,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: options.InsecureSkipVerify,
					},
				}
			}
		}
	} else {
		// 没有代理时使用默认Transport
		client.HTTPClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: options.InsecureSkipVerify,
			},
		}
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

	// 备份原有的 CheckRedirect 设置
	originalCheckRedirect := RetryClient.HTTPClient.CheckRedirect

	// 临时禁用重定向
	RetryClient.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	// 确保在函数结束时恢复原有设置
	defer func() {
		RetryClient.HTTPClient.CheckRedirect = originalCheckRedirect
	}()

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

// CheckProtocol 检查网络通信协议
func CheckProtocol(host string) (string, error) {
	if len(strings.TrimSpace(host)) == 0 {
		return "", fmt.Errorf("host %q is empty", host)
	}

	if strings.HasPrefix(host, HttpPrefix) || strings.HasPrefix(host, HttpsPrefix) {
		return host, nil
	}

	u, err := url.Parse(HttpPrefix + host)
	if err != nil {
		return "", err
	}

	switch u.Port() {
	case "80":
		return checkAndReturnProtocol(HttpPrefix + host)
	case "443":
		return checkAndReturnProtocol(HttpsPrefix + host)
	default:
		if result, err := checkAndReturnProtocol(HttpsPrefix + host); err == nil {
			return result, nil
		}
		return checkAndReturnProtocol(HttpPrefix + host)
	}
}

func checkAndReturnProtocol(url string) (string, error) {
	body, _, err := simpleRetryHttpGet(url)
	if err != nil {
		return "", err
	}

	if strings.Contains(string(body), "<title>400 The plain HTTP request was sent to HTTPS port</title>") && strings.HasPrefix(url, HttpPrefix) {
		return HttpsPrefix + url[len(HttpPrefix):], nil
	}

	return url, nil
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
	if oReq.Body != nil && oReq.Body != http.NoBody {
		data, err := io.ReadAll(oReq.Body)
		if err != nil {
			return nil, err
		}
		req.Body = data
		oReq.Body = io.NopCloser(bytes.NewBuffer(data))
	}

	return req, nil
}
