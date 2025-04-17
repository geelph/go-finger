# Wappalyzer 技术栈识别示例

本示例展示如何使用GXX的`WappalyzerScan`功能进行网站技术栈识别，该功能基于Wappalyzer引擎，可以快速识别网站使用的技术组件，而无需执行完整的指纹扫描流程。

## 功能特点

1. **快速技术栈识别** - 快速分析并识别目标网站使用的各种技术组件
2. **多维度技术分析** - 包括以下多种技术类别的识别：
   - Web服务器（如Nginx、Apache等）
   - 编程语言（如PHP、Python、Node.js等）
   - Web框架（如Laravel、Django、React等）
   - JavaScript框架和库（如jQuery、Vue.js等）
   - CMS系统（如WordPress、Drupal等）
   - 数据库技术（如MySQL、MongoDB等）
   - 操作系统信息（如Windows Server、Linux等）
   - 安全组件（如WAF、SSL等）
   - 缓存系统（如Redis、Memcached等）
   - 其他技术组件

3. **独立API** - 提供独立的API函数，可以单独调用而无需执行完整的指纹识别

## 运行方法

```bash
# 进入示例目录
cd example/wappalyzer_scan

# 编译并运行
go build -o wappalyzer_scan
./wappalyzer_scan
```

或直接运行：

```bash
go run main.go
```

## 示例代码解析

该示例主要展示了两种使用方式：

1. **直接获取基本信息** - 使用 `GetBaseInfo` 函数同时获取基本信息和技术栈
2. **专用技术栈识别** - 使用 `WappalyzerScan` 函数单独进行技术栈分析

### 核心代码说明

```go
// 方式1：通过基本信息获取技术栈
baseInfo, err := gxx.GetBaseInfo(target, "", timeout)
// 使用baseInfo.Wappalyzer访问技术栈信息

// 方式2：直接使用专用API进行技术栈分析
wappalyzerResult, err := gxx.WappalyzerScan(target, "", timeout)
```

## 输出示例

下面是运行示例后的典型输出：

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

## 可识别的技术类别

使用Wappalyzer技术栈识别功能，可以识别的技术类别包括但不限于：

1. Web服务器 (`wappalyzerResult.WebServers`)
2. 编程语言 (`wappalyzerResult.ProgrammingLanguages`)
3. Web框架 (`wappalyzerResult.WebFrameworks`)
4. JavaScript框架 (`wappalyzerResult.JavaScriptFrameworks`)
5. JavaScript库 (`wappalyzerResult.JavaScriptLibraries`)
6. CMS系统 (`wappalyzerResult.CmsSystems`)
7. 数据库技术 (`wappalyzerResult.Databases`)
8. 操作系统 (`wappalyzerResult.OperatingSystems`)
9. 缓存系统 (`wappalyzerResult.Caching`)
10. 安全组件 (`wappalyzerResult.Security`)
11. 反向代理 (`wappalyzerResult.ReverseProxies`)
12. 静态站点生成器 (`wappalyzerResult.StaticSiteGenerator`)
13. 主机面板 (`wappalyzerResult.HostingPanels`)
14. 其他组件 (`wappalyzerResult.Other`)

## 使用场景

- 网站技术调研和分析
- 安全评估前的技术侦察
- 竞品分析
- 网站架构调研

## 相关API

- `gxx.GetBaseInfo(target, proxy, timeout)` - 获取目标基本信息，包含技术栈
- `gxx.WappalyzerScan(target, proxy, timeout)` - 单独进行技术栈识别 