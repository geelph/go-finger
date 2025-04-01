/*
  - Package request
    @Author: zhizhuo
    @IDE：GoLand
    @File: test_test.go
    @Date: 2025/2/25 下午3:16*
*/
package request

import (
	"fmt"
	"log"
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
