# 基本扫描示例

本示例展示了GXX最基本的使用方式，通过API对单个或多个目标进行指纹识别。

## 🎯 功能特点

- 单目标和多目标扫描
- 配置基本扫描参数（超时时间、线程数）
- 获取详细的指纹识别结果
- 提取技术栈信息
- 输出扫描统计和耗时

## 🚀 运行示例

```bash
# 进入示例目录
cd example/basic_scan

# 运行示例
go run main.go
```

## 📖 代码说明

本示例展示了GXX核心功能的使用方法：

1. **初始化指纹规则库**：使用 `gxx.InitFingerRules()` 加载指纹规则
2. **执行指纹识别**：使用 `gxx.FingerScan()` 对目标进行扫描
3. **获取匹配结果**：使用 `gxx.GetFingerMatches()` 提取匹配的指纹信息
4. **获取技术栈信息**：通过 `result.Wappalyzer` 获取技术栈详细数据

## 💻 核心代码

```go
// 初始化指纹规则库
if err := gxx.InitFingerRules(options); err != nil {
    fmt.Printf("初始化指纹规则库失败: %v\n", err)
    os.Exit(1)
}

// 使用API接口方式扫描单个目标
result, err := gxx.FingerScan(target, "", timeout, workerCount)
if err != nil {
    fmt.Printf("扫描失败: %v\n", err)
    return
}

// 获取匹配的指纹
matches := gxx.GetFingerMatches(result)
```

## 📋 运行结果示例

```
开始扫描目标: [example.com]
[INFO] 原始目标数量：1个，重复目标数量：0个，去重后目标数量：1个
[INFO] 使用默认指纹库
[INFO] 加载指纹数量：156个
[INFO] 开始扫描 1 个目标，使用 5 个并发线程...
URL：example.com （200）  标题：Example Domain  Server：ECS  指纹：[Akamai]  匹配结果：成功
指纹识别 100% [==================================================] (1/1, 0.33 it/s)
──────────────────────────────────────────────────
扫描统计: 目标总数 1, 匹配成功 1, 匹配失败 0
扫描完成
```

## 🔍 自定义扫描

您可以通过修改以下参数来自定义扫描行为：

```go
// 设置目标URL
targets := []string{"example.com", "github.com"}

// 设置超时时间（秒）
timeout := 10

// 设置并发线程数
workerCount := 20
```

## 📚 进阶使用

- 尝试添加多个目标URL进行批量扫描
- 调整线程数观察对扫描速度的影响
- 修改超时时间来适应不同网络环境 