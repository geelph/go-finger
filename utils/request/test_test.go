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
	"github.com/txthinking/socks5"
	"log"
	"testing"
	"time"
)

func sendUDPMessage(message, target, proxyAddress string) error {

	// 创建 SOCKS5 代理客户端
	dialer, err := socks5.NewClient(proxyAddress, "", "", 0, 0)
	if err != nil {
		return fmt.Errorf("failed to create SOCKS5 client: %v", err)
	}

	// 使用代理拨号器创建 UDP 连接
	conn, err := dialer.Dial("udp", target)
	if err != nil {
		return fmt.Errorf("failed to connect to UDP server: %v", err)
	}
	defer conn.Close()

	// 发送消息
	if _, err := conn.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	fmt.Println("Message sent successfully")
	return nil
}

func TestNewClient(*testing.T) {
	address := "193.112.194.72:38219" // 目标地址和端口
	//proxyAddress := "socks5://127.0.0.1:10808"
	//message := "Hello, UDP Server!"
	//err := sendUDPMessage(message, address, proxyAddress)
	//if err != nil {
	//	fmt.Println("Error: ", err)
	//}
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

	//conn, err := net.DialTimeout("udp", address, 5*time.Second)
	//if err != nil {
	//	log.Fatalf("Failed to create UDP connection: %v", err)
	//}
	//defer conn.Close()
	//
	//message := []byte("Hello, UDP server!")
	//_, err = conn.Write(message)
	//if err != nil {
	//	log.Fatalf("Failed to send data: %v", err)
	//}
	//fmt.Println("Data sent successfully")
	//
	//buf := make([]byte, 2048)
	//conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	//n, err := conn.Read(buf)
	//if err != nil {
	//	log.Fatalf("Failed to receive data: %v", err)
	//}
	//fmt.Printf("Received response: %s\n", string(buf[:n]))
}
