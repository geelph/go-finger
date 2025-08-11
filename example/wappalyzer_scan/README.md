# 🔍 Wappalyzer 技术栈识别示例

本示例展示如何使用GXX的强大技术栈识别功能，基于Wappalyzer引擎快速分析目标网站使用的技术组件，无需执行完整的指纹扫描流程。

## ✨ 功能特点

- **全面的技术识别** - 识别20+种类别的技术组件
- **高效性能** - 快速分析目标网站使用的技术栈
- **低资源占用** - 相比完整指纹扫描，占用更少的系统资源
- **独立API** - 可单独调用，不依赖指纹识别流程

## 🛠️ 技术类别

Wappalyzer技术栈识别可检测以下类别的技术组件：

| 类别 | 说明 | 结构体字段 |
|------|------|------------|
| Web服务器 | Nginx、Apache、IIS等 | `wappalyzerResult.WebServers` |
| 编程语言 | PHP、Python、Java等 | `wappalyzerResult.ProgrammingLanguages` |
| Web框架 | Laravel、Django、Spring等 | `wappalyzerResult.WebFrameworks` |
| JavaScript框架 | React、Vue、Angular等 | `wappalyzerResult.JavaScriptFrameworks` |
| JavaScript库 | jQuery、Lodash等 | `wappalyzerResult.JavaScriptLibraries` |
| 安全组件 | WAF、SSL等 | `wappalyzerResult.Security` |
| 缓存系统 | Redis、Memcached等 | `wappalyzerResult.Caching` |
| 反向代理 | Nginx Proxy、CloudFlare等 | `wappalyzerResult.ReverseProxies` |
| 静态站点生成器 | Hugo、Jekyll等 | `wappalyzerResult.StaticSiteGenerator` |
| 主机面板 | cPanel、Plesk等 | `wappalyzerResult.HostingPanels` |

## 🚀 运行示例

```bash
# 进入示例目录
cd example/wappalyzer_scan

# 运行示例
go run main.go
```

## 💻 核心代码

本示例展示了两种获取技术栈信息的方法：

### 1. 通过基本信息获取技术栈

```go
// 获取目标基本信息，包含技术栈数据
baseInfo, err := gxx.GetBaseInfo(target, "", timeout)
if err != nil {
    fmt.Printf("获取目标基本信息失败: %v\n", err)
    return
}

// 访问技术栈信息
if baseInfo.Wappalyzer != nil {
    // 使用baseInfo.Wappalyzer访问各类技术信息
    fmt.Printf("Web服务器: %v\n", baseInfo.Wappalyzer.WebServers)
}
```

### 2. 专用技术栈分析API

```go
// 直接进行技术栈分析，不执行指纹识别
wappalyzerResult, err := gxx.WappalyzerScan(target, "", timeout)
if err != nil {
    fmt.Printf("技术栈分析失败: %v\n", err)
    return
}

// 处理分析结果
if len(wappalyzerResult.WebServers) > 0 {
    fmt.Printf("Web服务器: %v\n", wappalyzerResult.WebServers)
}
```

## 📋 输出示例

```
开始分析目标: https://github.com
--------------------------------------------
目标基本信息:
URL: https://github.com
状态码: 200
标题: GitHub: Let's build from here
服务器: GitHub.com

技术栈详细信息:

Web服务器:
  - GitHub.com

编程语言:
  - Ruby
  - JavaScript

JavaScript框架:
  - React
  - jQuery

安全组件:
  - GitHub Security
  - CSP
  - HSTS

--------------------------------------------
总耗时: 1.523s
分析完成
```

## 🔍 应用场景

- **技术调研** - 快速了解目标网站使用的技术栈
- **安全评估** - 安全测试前的目标侦察
- **竞品分析** - 分析竞争对手使用的技术
- **兼容性测试** - 确定目标网站的技术兼容性

## 📚 进阶使用

- **设置代理** - 通过指定代理参数绕过访问限制：`gxx.WappalyzerScan(target, "http://127.0.0.1:8080", timeout)`
- **自定义超时** - 针对不同网络环境设置合适的超时时间，如：`timeout := 15`
- **结果过滤** - 根据需要只关注特定类别的技术组件，如只提取Web框架信息 