# API接口扫描百度示例

这个示例展示了如何使用GXX的API接口直接进行指纹识别，目标为百度网站。该示例演示了在代码中集成GXX API的完整流程。

## 功能特点

- 直接使用API进行单个目标的指纹识别
- 获取目标网站的基础信息（标题、服务器、状态码）
- 执行详细的指纹识别过程
- 展示匹配的指纹详细信息

## 运行示例

```bash
# 进入示例目录
cd example/api_scan_baidu

# 运行示例
go run main.go
```

## 代码说明

1. **初始化配置**：创建并配置扫描选项
2. **加载指纹库**：初始化指纹规则库
3. **获取基础信息**：使用`GetBaseInfo`获取目标网站的基本信息
4. **执行指纹识别**：使用`FingerScan`进行指纹识别
5. **结果处理**：处理并展示匹配的指纹信息

## 关键API

```go
// 创建选项
options, err := gxx.NewFingerOptions()

// 初始化指纹规则
err := gxx.InitFingerRules(options)

// 获取基础信息
title, serverInfo, statusCode, resp, err := gxx.GetBaseInfo(target, proxy, timeout)

// 执行指纹识别
result, err := gxx.FingerScan(target, proxy, timeout, workerCount)

// 获取匹配结果
matches := gxx.GetFingerMatches(result)
```

## 输出示例

```
初始化指纹规则库...
初始化完成，耗时: 1.234s

开始获取目标基础信息: https://www.baidu.com
状态码: 200
标题: 百度一下，你就知道
服务器: BWS/1.0

开始进行指纹识别: https://www.baidu.com
指纹识别完成，耗时: 3.456s

识别结果:
URL: https://www.baidu.com
状态码: 200
标题: 百度一下，你就知道
服务器: BWS/1.0

匹配的指纹 (2个):
  1. jQuery
     - 匹配结果: true
     - 指纹ID: jquery-001
     - 描述: 流行的JavaScript库

  2. Nginx
     - 匹配结果: true
     - 指纹ID: nginx-001
     - 描述: 高性能Web服务器

扫描完成
```

## 注意事项

- 该示例需要网络连接以访问百度网站
- 对于一些网站可能需要配置代理以绕过网络限制
- 调整超时时间和并发线程数可以优化性能