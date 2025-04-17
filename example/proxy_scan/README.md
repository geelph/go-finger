# 代理扫描示例

这个示例展示了如何使用代理服务器进行指纹识别扫描，适用于需要匿名扫描或访问特定网络环境的场景。

## 功能特点

- 配置HTTP或SOCKS5代理
- 设置适当的超时时间和重试次数
- 通过代理访问目标，实现匿名扫描

## 运行示例

```bash
# 进入示例目录
cd example/proxy_scan

# 运行示例 (注意：请先修改代码中的代理地址为您可用的代理)
go run main.go
```

## 代码说明

1. **创建选项**：使用`NewFingerOptions()`创建配置选项
2. **设置目标**：指定要扫描的目标URL
3. **配置代理**：设置HTTP或SOCKS5代理地址
4. **调整参数**：增加超时时间和重试次数以适应代理环境
5. **执行扫描**：通过代理执行指纹识别

## 关键代码

```go
// 创建选项
options, err := gxx.NewFingerOptions()

// 设置目标
options.Target = []string{"example.com"}

// 设置代理 (HTTP或SOCKS5)
options.Proxy = "http://127.0.0.1:7890"

// 适应代理环境的参数
options.Timeout = 10
options.Retries = 2

// 执行扫描
gxx.FingerScan(options)
```

## 支持的代理类型

- **HTTP代理**：`http://ip:port` 或 `http://username:password@ip:port`
- **SOCKS5代理**：`socks5://ip:port` 或 `socks5://username:password@ip:port`

## 使用提示

1. **检查代理可用性**：在使用前请确保代理服务器可用
2. **增加超时时间**：使用代理时网络延迟通常更高，建议设置更长的超时时间
3. **开启调试模式**：对于代理问题的排查，开启调试模式能提供更多信息
4. **适当重试**：设置重试次数可以降低单次代理失败的影响

## 常见问题

- **代理连接失败**：检查代理地址是否正确，代理服务器是否在线
- **扫描速度慢**：代理会引入额外的网络延迟，这是正常现象
- **某些功能不可用**：部分代理可能会过滤或修改HTTP请求，影响某些功能 