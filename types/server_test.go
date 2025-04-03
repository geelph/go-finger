/*
  - Package types
    @Author: zhizhuo
    @IDE：GoLand
    @File: server_test.go
    @Date: 2025/4/4 上午11:15*
*/
package types

import (
	"encoding/json"
	"testing"
)

func TestServerInfo(t *testing.T) {
	// 测试创建新的ServerInfo对象
	originalServer := "Apache/2.4.41 (Ubuntu)"
	serverType := "Apache"
	version := "2.4.41"
	
	serverInfo := NewServerInfo(originalServer, serverType, version)
	
	if serverInfo.OriginalServer != originalServer {
		t.Errorf("OriginalServer = %v, 期望 %v", serverInfo.OriginalServer, originalServer)
	}
	if serverInfo.ServerType != serverType {
		t.Errorf("ServerType = %v, 期望 %v", serverInfo.ServerType, serverType)
	}
	if serverInfo.Version != version {
		t.Errorf("Version = %v, 期望 %v", serverInfo.Version, version)
	}
	
	// 测试创建空的ServerInfo对象
	emptyInfo := EmptyServerInfo()
	
	if emptyInfo.OriginalServer != "" {
		t.Errorf("EmptyServerInfo().OriginalServer = %v, 期望为空字符串", emptyInfo.OriginalServer)
	}
	if emptyInfo.ServerType != "" {
		t.Errorf("EmptyServerInfo().ServerType = %v, 期望为空字符串", emptyInfo.ServerType)
	}
	if emptyInfo.Version != "" {
		t.Errorf("EmptyServerInfo().Version = %v, 期望为空字符串", emptyInfo.Version)
	}
}

func TestServerInfoJSON(t *testing.T) {
	// 测试JSON序列化
	serverInfo := &ServerInfo{
		OriginalServer: "nginx/1.18.0",
		ServerType:     "nginx",
		Version:        "1.18.0",
	}
	
	jsonData, err := json.Marshal(serverInfo)
	if err != nil {
		t.Errorf("json.Marshal() 失败: %v", err)
	}
	
	// 验证JSON格式正确
	expectedJSON := `{"original_server":"nginx/1.18.0","server_type":"nginx","version":"1.18.0"}`
	if string(jsonData) != expectedJSON {
		t.Errorf("JSON序列化结果 = %v, 期望 %v", string(jsonData), expectedJSON)
	}
	
	// 测试JSON反序列化
	var newServerInfo ServerInfo
	err = json.Unmarshal(jsonData, &newServerInfo)
	if err != nil {
		t.Errorf("json.Unmarshal() 失败: %v", err)
	}
	
	if newServerInfo.OriginalServer != serverInfo.OriginalServer {
		t.Errorf("反序列化后 OriginalServer = %v, 期望 %v", newServerInfo.OriginalServer, serverInfo.OriginalServer)
	}
	if newServerInfo.ServerType != serverInfo.ServerType {
		t.Errorf("反序列化后 ServerType = %v, 期望 %v", newServerInfo.ServerType, serverInfo.ServerType)
	}
	if newServerInfo.Version != serverInfo.Version {
		t.Errorf("反序列化后 Version = %v, 期望 %v", newServerInfo.Version, serverInfo.Version)
	}
}

func TestServerInfoUsage(t *testing.T) {
	// 测试典型使用场景
	
	// 1. 创建不同类型的服务器信息
	httpServer := NewServerInfo("Apache/2.4.41", "Apache", "2.4.41")
	tcpServer := NewServerInfo("TCP 192.168.1.1:80", "TCP", "未知")
	emptyServer := EmptyServerInfo()
	
	// 2. 转换为JSON
	httpJSON, _ := json.Marshal(httpServer)
	tcpJSON, _ := json.Marshal(tcpServer)
	emptyJSON, _ := json.Marshal(emptyServer)
	
	// 3. 模拟variableMap使用场景
	variableMap := map[string]interface{}{
		"http_server": string(httpJSON),
		"tcp_server":  string(tcpJSON),
		"no_server":   string(emptyJSON),
	}
	
	// 4. 从variableMap中提取并解析
	for key, value := range variableMap {
		jsonStr, ok := value.(string)
		if !ok {
			t.Errorf("无法将 %s 转换为字符串", key)
			continue
		}
		
		var serverInfo ServerInfo
		err := json.Unmarshal([]byte(jsonStr), &serverInfo)
		if err != nil {
			t.Errorf("无法解析 %s 的JSON: %v", key, err)
			continue
		}
		
		// 验证字段存在且类型正确
		if serverInfo.OriginalServer == "" && key != "no_server" {
			t.Errorf("%s 的 OriginalServer 不应为空", key)
		}
		if serverInfo.ServerType == "" && key != "no_server" {
			t.Errorf("%s 的 ServerType 不应为空", key)
		}
		
		// 特定验证
		if key == "http_server" {
			if serverInfo.Version != "2.4.41" {
				t.Errorf("http_server 的 Version = %v, 期望 2.4.41", serverInfo.Version)
			}
		}
		if key == "tcp_server" {
			if serverInfo.ServerType != "TCP" {
				t.Errorf("tcp_server 的 ServerType = %v, 期望 TCP", serverInfo.ServerType)
			}
		}
		if key == "no_server" {
			if serverInfo.OriginalServer != "" || serverInfo.ServerType != "" || serverInfo.Version != "" {
				t.Errorf("no_server 应该包含空字段")
			}
		}
	}
} 