# GXX 新一代基于yaml的指纹识别工具
## 目录结构

```
gxx/
├── cmd/                    # 命令行应用程序入口点
│   ├── main.go            # 主程序入口点
│   ├── cli/               # 命令行处理模块
│   │   ├── cmd.go         # 命令行处理
│   │   ├── options.go     # 命令行选项
│   │   └── banner.go      # 应用程序横幅
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
└── README.md               # 项目说明文档
```

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
# 基本编译
CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build -ldflags "-w -s" -o gxx main.go

# 使用构建脚本
chmod +x build.sh
./build.sh

```
使用goreleaser编译

```shell
goreleaser build  --snapshot --clean --snapshot
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


## 开发说明

1. 指纹库结构应遵循特定格式
2. 可以通过添加新的 YAML 文件扩展指纹库
3. 支持 CEL 表达式进行复杂匹配

## 许可证

[许可证信息]