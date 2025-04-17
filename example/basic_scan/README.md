# 基本扫描示例

这个示例展示了GXX最基本的使用方式，通过命令行选项对单个或多个目标进行指纹识别。

## 功能特点

- 设置单个或多个目标URL进行扫描
- 配置基本的扫描参数（调试模式、超时时间、线程数）
- 使用默认输出方式展示结果

## 运行示例

```bash
# 进入示例目录
cd example/basic_scan

# 运行示例
go run main.go
```

## 代码说明

1. **创建选项**：使用`NewFingerOptions()`创建配置选项
2. **设置目标**：通过`options.Target`设置一个或多个目标URL
3. **配置参数**：设置调试模式、超时时间和线程数
4. **执行扫描**：使用`FingerScan(options)`执行扫描并输出结果

## 关键代码

```go
// 创建选项
options, err := gxx.NewFingerOptions()

// 设置目标
options.Target = []string{"example.com"}

// 配置参数
options.Debug = true
options.Timeout = 5
options.Threads = 5

// 执行扫描
gxx.FingerScan(options)
```

## 输出示例

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

## 扩展建议

- 尝试添加多个目标URL进行批量扫描
- 调整线程数观察对扫描速度的影响
- 修改超时时间来适应不同网络环境 