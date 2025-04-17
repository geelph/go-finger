# GXX 指纹识别工具使用示例

本目录包含了 GXX 指纹识别工具的使用示例，帮助您快速上手和了解工具的核心功能。每个示例都放在单独的目录中，并配有详细的说明文档。

## 示例目录

### 1. [基本扫描](basic_scan/)

最基本的扫描示例，展示如何进行单个目标的指纹识别。

**主要功能**:
- 设置单个或多个目标URL
- 初始化指纹规则库
- 使用API方式扫描每个目标
- 展示匹配的指纹信息

### 2. [代理扫描](proxy_scan/)

演示如何通过代理服务器进行扫描，适用于需要匿名扫描或访问特定网络环境的场景。

**主要功能**:
- 配置HTTP/SOCKS5代理
- 获取目标基础信息
- 通过代理进行指纹识别
- 输出详细的识别结果

### 3. [文件目标扫描](file_target_scan/)

展示如何从文件中读取目标列表进行批量扫描，适用于大规模扫描任务。

**主要功能**:
- 自动创建和读取目标文件
- 使用并发方式扫描多个目标
- 汇总展示扫描结果
- 将结果保存为CSV格式

### 4. [API接口扫描百度](api_scan_baidu/)

展示如何使用GXX API接口直接进行指纹识别，目标为百度网站。该示例演示了如何在代码中集成GXX的API功能。

**主要功能**:
- 直接使用API接口进行扫描
- 获取目标基础信息
- 执行指纹识别并处理结果
- 详细展示匹配的指纹信息

### 5. [Wappalyzer技术栈识别](wappalyzer_scan/)

展示如何使用集成的Wappalyzer功能单独进行网站技术栈识别，无需执行完整的指纹扫描。

**主要功能**:
- 使用WappalyzerScan API进行技术栈识别
- 获取Web服务器、编程语言、框架等技术信息
- 详细展示分析结果
- 快速识别网站使用的技术组件

## 运行示例

每个示例目录下都有独立的README文件和main.go文件。要运行特定示例，进入其目录并执行：

```bash
cd example/<示例目录>
go run main.go
```

## API 使用说明

GXX 提供了简单易用的 API，便于集成到您的项目中：

```go
// 1. 创建配置选项
options, err := gxx.NewFingerOptions()

// 2. 初始化指纹规则库(仅需执行一次)
err := gxx.InitFingerRules(options)

// 3. 获取目标基础信息
baseInfo, err := gxx.GetBaseInfo(target, proxy, timeout)
// 访问基础信息
fmt.Printf("标题: %s\n", baseInfo.Title)
fmt.Printf("状态码: %d\n", baseInfo.StatusCode)
if baseInfo.ServerInfo != nil {
    fmt.Printf("服务器: %s\n", baseInfo.ServerInfo.ServerType)
}

// 4. 执行指纹识别
result, err := gxx.FingerScan(target, proxy, timeout, workerCount)

// 5. 获取匹配结果
matches := gxx.GetFingerMatches(result)

// 6. 单独执行技术栈分析
wappResult, err := gxx.WappalyzerScan(target, proxy, timeout)
```

## 主要API函数

| 函数 | 描述 |
|------|------|
| `NewFingerOptions()` | 创建新的扫描选项 |
| `InitFingerRules(options)` | 初始化指纹规则库 |
| `GetBaseInfo(target, proxy, timeout)` | 获取目标基础信息(包含标题、服务器信息、技术栈等) |
| `FingerScan(target, proxy, timeout, workerCount)` | 执行指纹识别 |
| `GetFingerMatches(result)` | 获取匹配的指纹列表 |
| `WappalyzerScan(target, proxy, timeout)` | 单独进行技术栈识别(不执行指纹匹配) |

## 主要数据类型

| 类型 | 描述 |
|------|------|
| `gxx.BaseInfoType` | 目标基础信息，包含标题、服务器信息、状态码、技术栈等 |
| `gxx.TargetResult` | 扫描结果，包含匹配的指纹和基础信息 |
| `gxx.FingerMatch` | 指纹匹配结果，包含指纹详情和匹配数据 |
| `gxx.TypeWappalyzer` | 技术栈分析结果，包含Web服务器、编程语言、框架等信息 |
| `gxx.ServerInfo` | 服务器信息，包含类型、版本等 |

## 主要配置选项

| 选项 | 类型 | 说明 |
|------|------|------|
| Target | []string | 要扫描的目标URL/主机列表 |
| TargetsFile | string | 包含目标列表的文件路径 |
| Threads | int | 并发线程数（默认：10） |
| Output | string | 输出文件路径 |
| Proxy | string | 代理服务器地址 |
| Timeout | int | 请求超时时间（秒） |
| Debug | bool | 是否开启调试模式 |

## 使用提示

1. **选择合适的示例**：根据您的需求选择最接近的示例作为起点
2. **查看示例README**：每个示例目录下都有详细的README文件
3. **API初始化顺序**：确保先调用`InitFingerRules`再执行指纹识别
4. **并发与性能**：利用并发特性可以显著提高扫描效率
5. **技术栈识别**：如果只需获取技术栈信息，可以直接使用`WappalyzerScan`函数
6. **基础信息获取**：使用`GetBaseInfo`函数可快速获取目标网站的基本信息

## 问题反馈

如果您在使用过程中遇到任何问题，或有改进建议，请通过以下方式联系我们：

- 提交Issue
- 发送邮件到开发者邮箱 