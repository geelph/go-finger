# GXX - 新一代基于YAML的指纹识别工具

GXX是一款强大的指纹识别工具，基于YAML配置的规则进行目标系统识别。
本工具支持多种协议（HTTP/HTTPS、TCP、UDP），可进行高效的批量目标扫描和精准识别。

## 💡 主要特性

- **强大的指纹识别** - 基于YAML的规则配置，简洁而强大
- **高性能并发** - 使用协程池实现高效并发扫描，支持大规模目标
- **多协议支持** - 全面支持HTTP/HTTPS、TCP、UDP协议
- **代理功能** - 支持HTTP/SOCKS5代理，可配置多个代理地址
- **批量扫描** - 支持从文件读取多个目标进行批量扫描
- **多格式输出** - 支持TXT/CSV/JSON等多种输出格式
- **自定义规则** - 灵活的指纹规则自定义功能
- **技术栈识别** - 内置Wappalyzer引擎，快速识别网站技术组件
- **CEL表达式** - 使用强大的CEL表达式引擎进行规则匹配
- **实时输出** - 支持Unix domain socket实时结果输出

## 🚀 快速开始

### 安装

```bash
# 使用 go install 安装
go install github.com/yourusername/gxx@latest

# 或下载预编译版本
# 从 releases 页面下载对应平台的二进制文件
```

### 基本使用

```bash
# 扫描单个目标
gxx -u https://example.com

# 从文件读取目标列表
gxx -f targets.txt

# 使用代理
gxx -u https://example.com --proxy http://127.0.0.1:8080

# 指定输出文件
gxx -u https://example.com -o results.txt

# 开启调试模式
gxx -u https://example.com --debug

# 禁用文件日志记录
gxx -u https://example.com --no-file-log
```

## 📖 命令行参数

### 输入选项
- `-u, --url`：要扫描的目标URL/主机（可指定多个）
- `-f, --file`：包含目标URL/主机列表的文件（每行一个）
- `-t, --threads`：并发线程数（默认：10）

### 输出选项
- `-o, --output`：输出文件路径
- `--format`：输出文件格式（支持 txt/csv/json，默认：txt）
- `--sock-output`：Unix domain socket输出路径（用于实时数据流）

### 调试选项
- `--proxy`：HTTP/SOCKS5代理（支持逗号分隔的列表或文件输入）
- `-p, --poc`：测试单个YAML文件
- `-pf, --poc-file`：测试指定目录下的所有YAML文件
- `--debug`：开启调试模式
- `--no-file-log`：禁用文件日志记录，仅输出日志到控制台
- `--timeout`：设置请求超时时间（秒，默认：5）

## 🧰 API使用

GXX提供了简单易用的API，便于集成到您的项目中。以下是主要API和使用示例：

### 导入包
```go
import (
    "gxx"
    "gxx/types"
)
```

### 主要数据类型

#### BaseInfoType
包含目标站点的基本信息，包括标题、服务器信息、状态码和技术栈等：

```go
type BaseInfoType struct {
    Target     string                     // 目标URL
    Title      string                     // 网页标题
    ServerInfo *ServerInfo                // 服务器信息
    StatusCode int32                      // HTTP状态码
    Response   *http.Response             // HTTP原始响应
    Wappalyzer *TypeWappalyzer            // 技术栈信息
}
```

#### ServerInfo
服务器信息结构体：

```go
type ServerInfo struct {
    OriginalServer string                 // 原始Server头信息
    ServerType     string                 // 服务器类型
    Version        string                 // 版本信息
}
```

#### TargetResult
包含扫描结果的结构体：

```go
type TargetResult struct {
    URL        string                     // 目标URL
    StatusCode int32                      // HTTP状态码
    Title      string                     // 网页标题
    Server     *ServerInfo                // 服务器信息
    Matches    []*FingerMatch             // 匹配的指纹信息
    Wappalyzer *TypeWappalyzer            // 技术栈信息
}
```

#### TypeWappalyzer
技术栈信息结构体：

```go
type TypeWappalyzer struct {
    WebServers           []string         // Web服务器
    ReverseProxies       []string         // 反向代理
    JavaScriptFrameworks []string         // JS框架
    JavaScriptLibraries  []string         // JS库
    WebFrameworks        []string         // Web框架
    ProgrammingLanguages []string         // 编程语言 
    Caching              []string         // 缓存技术
    Security             []string         // 安全组件
    StaticSiteGenerator  []string         // 静态站点生成器
    HostingPanels        []string         // 主机面板
    Other                []string         // 其他杂项
}
```

### 主要API函数

#### 1. 初始化指纹规则
```go
// 创建默认配置选项
options, err := gxx.NewFingerOptions()
if err != nil {
    // 错误处理
}

// 初始化指纹规则（仅需执行一次）
err = gxx.InitFingerRules(options)
if err != nil {
    // 错误处理
}
```

#### 2. 单个URL识别
```go
// 扫描单个URL
target := "https://example.com"
proxy := "" // 如果需要代理，指定代理地址
timeout := 5 // 超时时间，单位：秒
workerCount := 10 // 并发工作线程数

// 执行指纹识别
result, err := gxx.FingerScan(target, proxy, timeout, workerCount)
if err != nil {
    // 错误处理
}
```

#### 3. 获取匹配结果
```go
// 获取所有匹配的指纹
matches := gxx.GetFingerMatches(result)
for _, match := range matches {
    // 指纹ID: match.Finger.Id
    // 指纹名称: match.Finger.Info.Name
    // 匹配结果: match.Result
    // 请求信息: match.Request
    // 响应信息: match.Response
}
```

#### 4. 获取目标基础信息
```go
// 获取目标站点的基础信息
baseInfo, err := gxx.GetBaseInfo(target, proxy, timeout)
if err != nil {
    // 错误处理
}

// 处理结果
fmt.Printf("标题: %s\n", baseInfo.Title)
fmt.Printf("状态码: %d\n", baseInfo.StatusCode)
if baseInfo.ServerInfo != nil {
    fmt.Printf("服务器: %s\n", baseInfo.ServerInfo.ServerType)
}
if baseInfo.Wappalyzer != nil {
    fmt.Printf("Web服务器: %v\n", baseInfo.Wappalyzer.WebServers)
    fmt.Printf("编程语言: %v\n", baseInfo.Wappalyzer.ProgrammingLanguages)
}
```

#### 5. 技术栈识别
```go
// 单独进行技术栈识别，不执行指纹匹配
wappResult, err := gxx.WappalyzerScan(target, proxy, timeout)
if err != nil {
    // 错误处理
}

// 处理技术栈分析结果
if len(wappResult.WebServers) > 0 {
    fmt.Printf("Web服务器: %v\n", wappResult.WebServers)
}
if len(wappResult.ProgrammingLanguages) > 0 {
    fmt.Printf("编程语言: %v\n", wappResult.ProgrammingLanguages)
}
if len(wappResult.JavaScriptFrameworks) > 0 {
    fmt.Printf("JS框架: %v\n", wappResult.JavaScriptFrameworks)
}
```

### 完整使用示例

```go
package main

import (
    "fmt"
    "gxx"
    "log"
)

func main() {
    // 1. 创建配置选项
    options, err := gxx.NewFingerOptions()
    if err != nil {
        log.Fatalf("创建选项错误: %v", err)
    }

    // 2. 初始化指纹规则库（仅需执行一次）
    if err := gxx.InitFingerRules(options); err != nil {
        log.Fatalf("初始化指纹规则错误: %v", err)
    }

    // 3. 处理单个URL
    target := "https://example.com"
    proxy := "" // 如果不需要代理，设为空字符串
    timeout := 5 // 超时时间，单位：秒
    workerCount := 10 // 并发工作线程数

    result, err := gxx.FingerScan(target, proxy, timeout, workerCount)
    if err != nil {
        log.Printf("处理URL错误: %v", err)
        return
    }

    // 4. 输出基本信息
    fmt.Printf("URL: %s\n", result.URL)
    fmt.Printf("状态码: %d\n", result.StatusCode)
    fmt.Printf("标题: %s\n", result.Title)
    if result.Server != nil {
        fmt.Printf("服务器: %s\n", result.Server.ServerType)
    }

    // 5. 处理匹配结果
    matches := gxx.GetFingerMatches(result)
    if len(matches) > 0 {
        fmt.Println("\n匹配的指纹:")
        for i, match := range matches {
            fmt.Printf("  %d. %s (ID: %s, 匹配结果: %v)\n", 
                i+1, match.Finger.Info.Name, match.Finger.Id, match.Result)
        }
    } else {
        fmt.Println("\n未匹配到任何指纹")
    }
    
    // 6. 获取技术栈信息
    if result.Wappalyzer != nil {
        fmt.Println("\n技术栈信息:")
        if len(result.Wappalyzer.WebServers) > 0 {
            fmt.Printf("  Web服务器: %v\n", result.Wappalyzer.WebServers)
        }
        if len(result.Wappalyzer.ProgrammingLanguages) > 0 {
            fmt.Printf("  编程语言: %v\n", result.Wappalyzer.ProgrammingLanguages)
        }
        if len(result.Wappalyzer.WebFrameworks) > 0 {
            fmt.Printf("  Web框架: %v\n", result.Wappalyzer.WebFrameworks)
        }
        if len(result.Wappalyzer.JavaScriptFrameworks) > 0 {
            fmt.Printf("  JS框架: %v\n", result.Wappalyzer.JavaScriptFrameworks)
        }
    }
}
```

## 🔍 示例代码

查看 [example](example/) 目录获取完整使用示例：

- [基本扫描](example/basic_scan/)：单目标扫描示例
- [代理扫描](example/proxy_scan/)：使用代理进行扫描
- [文件目标扫描](example/file_target_scan/)：批量扫描多个目标
- [百度API扫描](example/api_scan_baidu/)：API集成示例
- [Wappalyzer技术栈识别](example/wappalyzer_scan/)：网站技术栈识别

## 📂 项目目录结构

```
gxx/
├── cmd/                    # 命令行应用程序入口点
├── pkg/                    # 核心功能实现包
│   ├── finger/             # 指纹识别实现
│   ├── runner/             # 扫描运行器
│   ├── wappalyzer/         # Wappalyzer技术栈识别
│   └── cel/                # CEL表达式处理
├── types/                  # 类型定义
├── utils/                  # 工具和核心功能代码
│   ├── config/             # 配置管理
│   ├── logger/             # 日志管理
│   ├── common/             # 通用工具函数
│   ├── request/            # 请求处理
│   └── output/             # 结果输出处理
├── logs/                   # 日志输出目录
├── example/                # 示例代码
├── docs/                   # 文档目录
├── go.mod                  # Go模块定义
└── README.md               # 项目说明文档
```

## 🔨 编译与构建

### 使用 Makefile (推荐)

```bash
# 构建项目（不嵌入指纹库）
make build

# 构建项目（嵌入指纹库）
make build-embed

# 构建发布包（不嵌入指纹库）
make release

# 构建发布包（嵌入指纹库）
make release-embed

# 查看所有可用命令
make help
```

### 手动编译

```bash
# 基本编译
CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build -ldflags "-w -s" -o gxx main.go

# 使用构建脚本
chmod +x build.sh
./build.sh
```

### 使用 goreleaser 编译

```bash
goreleaser build --snapshot --clean --snapshot
```

## 📝 指纹规则格式

GXX使用YAML格式定义指纹规则，规则设计简洁明了，易于理解与扩展。详细说明请参考：
- [指纹规则格式说明](docs/指纹规则格式说明.md)
- [指纹开发快速参考](docs/指纹开发快速参考.md)

### 基本结构示例

```yaml
id: web-application
    
info:
  name: Web应用识别
  author: 作者名
  description: 识别特定Web应用
  reference:
    - https://example.com
  created: 2025/04/01
    
rules:
  r0:
    request:
      method: GET
      path: /
    expression: response.status == 200 && response.body.ibcontains(b"特征字符串")
    
expression: r0()
```

**提示**: 推荐使用`ibcontains`函数进行大小写不敏感的关键词匹配，这能提高识别的准确性。

## 🚀 性能优化

GXX专为高性能设计，主要优化点包括：
- 使用高效的协程池管理并发
- 实现智能的请求超时控制
- 采用分级并发控制，提高扫描效率
- 针对大规模扫描优化内存使用

## 🤝 贡献指南

欢迎为GXX贡献代码或指纹规则：

- **规则贡献**：通过添加新的YAML格式指纹规则文件扩展指纹库
- **代码贡献**：遵循项目代码结构进行功能开发
- **问题反馈**：通过Issues提交问题或功能建议
- **文档改进**：完善项目文档和使用示例

## ⚠️ 免责声明

本工具仅用于授权的安全测试和研究目的。使用者应遵守相关法律法规，未经授权不得对目标系统进行扫描。工具作者不对任何滥用行为负责。

## 📜 许可证

[MIT License](LICENSE)