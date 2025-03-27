# GXX 示例代码

本目录包含了 GXX 指纹识别工具的各种使用示例。

## 示例说明

### 1. 基本扫描 (basic_scan.go)
最基本的扫描示例，展示如何使用 GXX 进行单个目标的指纹识别。

```bash
go run basic_scan.go
```

### 2. 代理扫描 (proxy_scan.go)
展示如何使用代理进行扫描。

```bash
go run proxy_scan.go
```

### 3. 文件目标扫描 (file_target_scan.go)
展示如何使用目标文件列表进行批量扫描。

```bash
go run file_target_scan.go
```

## 使用说明

1. 确保已经正确安装了 GXX
2. 进入 example 目录
3. 根据需要修改示例代码中的参数
4. 运行相应的示例

## 注意事项

- 运行示例前请确保有正确的网络连接
- 使用代理示例时，请确保代理服务器可用
- 使用文件目标扫描时，请确保目标文件存在且格式正确 