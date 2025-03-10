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

```shell
go mod tidy

CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin go build -ldflags "-w -s" -o gxx main.go
chmod u+x ./StartsScan
./gxx -h
```

## 使用

```shell
./gxx -h
```