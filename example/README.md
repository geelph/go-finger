# GXX 指纹识别工具使用示例

本目录包含了 GXX 指纹识别工具的使用示例，帮助您快速上手和了解工具的核心功能。

## 示例说明

### 1. 基本扫描 ([basic_scan.go](basic_scan.go))

最基本的扫描示例，展示如何进行单个目标的指纹识别。

```bash
# 运行示例
go run basic_scan.go
```

**主要功能**:
- 设置单个或多个目标URL
- 配置调试模式
- 设置线程数和超时时间

### 2. 代理扫描 ([proxy_scan.go](proxy_scan.go))

演示如何通过代理服务器进行扫描，适用于需要匿名扫描或访问特定网络环境的场景。

```bash
# 运行示例
go run proxy_scan.go
```

**主要功能**:
- 配置HTTP/SOCKS5代理
- 设置合适的超时时间
- 配置重试次数

### 3. 文件目标扫描 ([file_target_scan.go](file_target_scan.go))

展示如何从文件中读取目标列表进行批量扫描，适用于大规模扫描任务。

```bash
# 创建目标文件
echo "example.com" > targets.txt
echo "github.com" >> targets.txt

# 运行示例
go run file_target_scan.go
```

**主要功能**:
- 从文件读取目标列表
- 配置输出文件保存结果
- 设置并发线程数优化性能

## API 使用说明

GXX 提供了简单易用的 API，便于集成到您的项目中：

```go
// 创建选项
options, err := gxx.NewFingerOptions()
if err != nil {
    // 错误处理
}

// 配置选项
options.Target = []string{"example.com"}
options.Threads = 10
options.Debug = true

// 执行扫描
gxx.FingerScan(options)
```

### 主要配置选项

| 选项 | 类型 | 说明 |
|------|------|------|
| Target | []string | 要扫描的目标URL/主机列表 |
| TargetsFile | string | 包含目标列表的文件路径 |
| Threads | int | 并发线程数（默认：10） |
| Output | string | 输出文件路径 |
| Proxy | string | 代理服务器地址 |
| Timeout | int | 请求超时时间（秒） |
| Retries | int | 请求失败重试次数 |
| Debug | bool | 是否开启调试模式 |

## 使用提示

1. **合理设置线程数**：根据您的网络环境和目标数量调整线程数，过高可能导致IP被封禁
2. **处理大量目标**：对于大规模扫描，建议使用文件输入并保存结果到输出文件
3. **使用代理**：对于敏感目标，请考虑使用代理服务器进行扫描
4. **错误处理**：在实际应用中，确保添加适当的错误处理逻辑
5. **结果处理**：根据您的需求处理和分析扫描结果

## 自定义扩展

您可以基于这些示例进行扩展，例如：

- 添加更复杂的目标筛选逻辑
- 自定义结果处理和分析
- 结合其他工具形成完整的安全评估流程
- 开发Web界面或其他形式的用户交互 