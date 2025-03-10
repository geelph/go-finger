# GXX 新一代基于yaml的指纹识别工具
## 目录结构

```
gxx/
├── cmd/                    # 命令行应用程序入口点
│   ├── cmd.go              # 命令行处理
│   ├── options.go          # 命令行选项
│   └── banner.go           # 应用程序横幅
├── utils/                  # 工具和核心功能代码
│   ├── config/             # 配置管理
│   │   └── config.go
│   ├── logger/             # 日志管理
│   │   └── logger.go
│   ├── common/             # 通用工具函数
│   │   ├── common.go
│   │   └── file.go
│   ├── finger/             # 核心指纹识别功能
│   │   ├── runner.go
│   │   ├── icon.go
│   │   ├── req.go
│   │   ├── eval.go
│   │   └── yaml.go         # 重命名自yaml_finger.go
│   ├── proto/              # 协议相关代码
│   ├── cel/                # CEL表达式处理
│   ├── request/            # 请求处理
│   └── pkg/                # 可以被外部应用程序使用的库代码
│       ├── network/        # 网络相关功能
│       ├── parser/         # 解析器功能
│       └── helper/         # 辅助功能
├── finger/                 # 指纹文件目录
│   ├── tcp_demo.yml
│   ├── udp_demo.yml
│   └── finger_demo.yaml
├── test/                   # 测试目录
├── logs/                   # 日志输出目录
├── go.mod                  # Go模块定义
├── go.sum                  # Go模块依赖校验和
├── main.go                 # 程序入口点
└── README.md               # 项目说明文档
```

## 命名规范

1. **目录命名**:
    - 使用小写字母
    - 使用单数形式（如 `finger` 而非 `fingers`）
    - 使用简短、描述性的名称
    - 避免使用缩写，除非是广泛接受的缩写（如 `cmd`）

2. **文件命名**:
    - 使用小写字母
    - 使用下划线分隔单词（如 `yaml.go` 而非 `yaml_finger.go`）
    - 文件名应反映其内容和功能
    - 避免在文件名中包含包名（如 `finger_yaml.go`）

3. **包命名**:
    - 使用小写字母
    - 使用单个单词，避免下划线
    - 包名应与目录名一致
    - 包名应具有描述性，但保持简短

## 目录结构调整说明

1. **cmd 目录**:
    - 保留原有的 cmd 目录，但将其扁平化
    - 将 main.go 移至 cmd/ 目录下

2. **utils 目录**:
    - 保留 utils 目录作为主要代码存放位置
    - 将 utils/pkg/finger 移至 utils/finger
    - 保持 utils/config、utils/logger、utils/common 不变
    - 将 utils/pkg 下的 proto、cel、request 直接移至 utils 目录下
    - 新增 utils/pkg 目录，用于存放可被外部应用程序使用的库代码
    - 根据功能将代码组织到 utils/pkg 的子目录中

3. **configs 目录**:
    - 新增 configs 目录，用于存放配置文件
    - 将原 Finger 目录下的配置文件移至 configs 目录

4. **文件重命名**:
    - 将 yaml_finger.go 重命名为 yaml.go，以保持命名简洁

## 功能模块组织

1. **核心功能模块**:
    - utils/finger: 指纹识别核心功能
    - utils/proto: 协议处理相关功能
    - utils/cel: CEL表达式处理功能
    - utils/request: HTTP请求处理功能

2. **基础设施模块**:
    - utils/config: 配置管理
    - utils/logger: 日志管理
    - utils/common: 通用工具函数

3. **可重用库模块**:
    - utils/pkg/network: 网络相关功能库
    - utils/pkg/parser: 解析器功能库
    - utils/pkg/helper: 辅助功能库


## 编译

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
# 基本编译（不嵌入指纹库）
go build -o gxx main.go

# 使用构建脚本（不嵌入指纹库）
chmod +x build.sh
./build.sh

# 使用构建脚本（嵌入指纹库）
chmod +x build.sh
./build.sh --embed
```

## 使用

```bash
# 基本用法
./gxx --target https://example.com

# 使用代理
./gxx --target https://example.com --proxy http://127.0.0.1:8080

# 使用目标文件
./gxx --targets-file targets.txt

# 指定POC文件目录
./gxx --target https://example.com --poc-file /path/to/finger

# 指定单个POC YAML文件
./gxx --target https://example.com --poc-yaml /path/to/specific.yaml
```

## 命令行选项

- `--target`: 指定目标URL
- `--targets-file`: 指定包含多个目标的文件
- `--poc-file`: 指定POC文件目录
- `--poc-yaml`: 指定单个POC YAML文件
- `--proxy`: 指定代理地址
- `--timeout`: 设置请求超时时间（秒）
- `--retries`: 设置重试次数
- `--output`: 指定输出文件
- `--debug`: 启用调试模式

## 打包说明

在打包二进制文件时，需要确保 `finger` 目录被正确包含。有以下几种方式：

### 方法1: 将 finger 目录放在二进制文件同级目录

```
/your/deploy/directory/
├── gxx (二进制文件)
└── finger/ (指纹库目录)
    ├── cms/
    ├── framework/
    └── ...
```

### 方法2: 使用 go-bindata 或 embed 将指纹库嵌入二进制

1. 使用 Go 1.16+ 的 embed 功能:

```go
package finger

import (
    "embed"
)

//go:embed finger/*
var fingerFS embed.FS

// 然后修改 GetFingerPath 和相关函数以使用嵌入的文件系统
```

2. 或者使用 go-bindata:

```bash
# 安装 go-bindata
go get -u github.com/go-bindata/go-bindata/...

# 生成嵌入数据
go-bindata -o utils/finger/bindata.go -pkg finger finger/...

# 然后修改代码以使用生成的 bindata.go
```

## 开发说明

1. 指纹库结构应遵循特定格式
2. 可以通过添加新的 YAML 文件扩展指纹库
3. 支持 CEL 表达式进行复杂匹配

## 许可证

[许可证信息]