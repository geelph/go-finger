/*
  - Package request
    @Author: zhizhuo
    @IDE：GoLand
    @File: test_test.go
    @Date: 2025/2/25 下午3:16*
*/
package network

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/net/context"
	"io"
	"log"
	"net/http"
	"testing"
	"time"
)

func TestNewTcpClient(t *testing.T) {

	address := "4.ipw.cn"

	conf := TcpOrUdpConfig{
		Network:      "tcp",
		MaxRetries:   3,
		ReadSize:     2048,
		DialTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
		ProxyURL:     "http://127.0.0.1:10809",
	}

	client, err := NewTcpClient(address, conf)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	message := []byte("GET / HTTP/1.1\r\nHost: 4.ipw.cn\r\nConnection: close\r\n\r\n")
	err = client.SendTcp(message)
	if err != nil {
		log.Fatalf("Failed to send data: %v", err)
	}
	fmt.Println("Data sent successfully")

	response, err := client.RecvTcp()
	if err != nil {
		log.Fatalf("Failed to receive data: %v", err)
	}
	fmt.Printf("Received response: %s\n", string(response))
}
func TestNewUdpClient(*testing.T) {
	address := "193.112.194.72:35469" // 目标地址和端口
	conf := TcpOrUdpConfig{
		Network:      "udp",
		MaxRetries:   3,
		ReadSize:     2048,
		DialTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
		ProxyURL:     "socks5://127.0.0.1:10808",
	}

	client, err := NewUdpClient(address, conf)
	if err != nil {
		log.Fatalf("Failed to create UDP client: %v", err)
	}
	defer client.Close()

	message := []byte("Hello, UDP server!")
	err = client.SendUDP(message)
	if err != nil {
		log.Fatalf("Failed to send data: %v", err)
	}
	fmt.Println("Data sent successfully")

	response, err := client.RecvUdp()
	if err != nil {
		log.Fatalf("Failed to receive data: %v", err)
	}
	fmt.Printf("Received response: %s\n", string(response))

}

func TestNewTHttpClient(t *testing.T) {
	url := "http://172.22.3.251"
	res, code, err := simpleRetryHttpGet(url, "", 0)
	if err == nil {
		fmt.Printf("Http response received: %s\n", string(res))
	}
	fmt.Printf("Http response code: %d\n", code)

}

func TestHttpCtxClient(t *testing.T) {
	url := "https://172.22.3.251/"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	options := OptionsRequest{
		Proxy:              "",               // 初始化为空，后面设置
		Timeout:            10 * time.Second, // 使用与context相同的超时时间
		Retries:            3,                // 重试次数
		FollowRedirects:    true,
		InsecureSkipVerify: true,
		CustomHeaders: map[string]string{
			"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		},
	}
	defer cancel()

	fmt.Println("开始发送请求...")
	fmt.Printf("请求URL: %s\n", url)
	fmt.Printf("使用代理: %s\n", options.Proxy)
	fmt.Printf("超时设置: %v\n", options.Timeout)

	resp, err := SendRequestHttp(ctx, "GET", url, "", options)
	if err != nil {
		fmt.Printf("发送请求出错，错误信息：%v\n", err)
		return
	}
	if resp == nil {
		fmt.Println("响应为空")
		return
	}
	defer func(Body io.ReadCloser) {
		if Body != nil {
			_ = Body.Close()
		}
	}(resp.Body)

	fmt.Printf("响应状态码: %d\n", resp.StatusCode)

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应体出错: %v\n", err)
		return
	}
	fmt.Printf("响应内容长度: %d bytes\n", len(body))
	fmt.Printf("响应内容: %s\n", string(body))
}

func TestHTTPRequest(t *testing.T) {
	// 创建自定义的 Transport
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // 注意：仅用于测试目的，不推荐用于生产环境
			MinVersion:         tls.VersionTLS10,
			CipherSuites: []uint16{
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
			},
		},
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second, // 设置超时时间
	}

	resp, err := client.Get("https://172.22.3.251/")
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("响应状态码: %d\n", resp.StatusCode)
}
