#!/bin/bash

# 设置应用名称和版本号
APP_NAME="gxx"
VERSION="1.1.4"
AUTHOR="zhizhuo"
# 添加构建时间
BUILD_DATE=$(date +"%Y-%m-%d")

# 创建构建目录
BUILD_DIR="build"
mkdir -p $BUILD_DIR

# 显示帮助信息
show_help() {
    echo "GXX 构建脚本"
    echo "用法: $0 [选项]"
    echo "选项:"
    echo "  -h, --help     显示帮助信息"
    echo "  -v, --version  显示版本信息"
    echo "  -c, --clean    清理构建目录"
    echo "  -a, --all      构建所有平台版本"
    echo "  -d, --debug    启用调试模式"
    echo "  -e, --embed    嵌入指纹库"
    echo ""
    echo "支持的平台和架构:"
    echo "  - darwin/amd64"
    echo "  - darwin/arm64"
    echo "  - linux/amd64"
    echo "  - linux/arm64"
    echo "  - windows/amd64"
    echo "  - windows/arm64"
}

# 显示版本信息
show_version() {
    echo "GXX 版本: $VERSION"
}

# 清理构建目录
clean_build() {
    echo "清理构建目录..."
    rm -rf $BUILD_DIR
    echo "清理完成"
}

# 构建函数
build() {
    GOOS=$1
    GOARCH=$2
    DEBUG=$3
    EMBED=$4

    # 根据平台和架构设置输出名称中的平台和架构部分
    case $GOOS in
        windows)
            PLATFORM="win"
            ;;
        darwin)
            PLATFORM="mac"
            ;;
        linux)
            PLATFORM="linux"
            ;;
        *)
            echo "未知的平台: $GOOS"
            exit 1
    esac

    case $GOARCH in
        amd64)
            ARCH="x64"
            ;;
        386)
            ARCH="x86"
            ;;
        arm64)
            ARCH="arm64"
            ;;
        arm)
            ARCH="arm"
            ;;
        *)
            echo "未知的架构: $GOARCH"
            exit 1
    esac

    OUTPUT_NAME="${APP_NAME}"
    OUTPUT_ZIP_NAME="${APP_NAME}_${PLATFORM}_${ARCH}_${VERSION}"

    echo "正在为 ${PLATFORM}/${ARCH} 构建..."

    # 设置输出文件名
    if [ "$GOOS" == "windows" ]; then
        OUTPUT_FILE="$BUILD_DIR/$OUTPUT_NAME.exe"
    else
        OUTPUT_FILE="$BUILD_DIR/$OUTPUT_NAME"
    fi

    # 构建标志
    LDFLAGS="-w -s"
    if [ "$DEBUG" == "true" ]; then
        LDFLAGS="-w"
    fi

    # 添加版本和作者信息到 ldflags
    LDFLAGS="$LDFLAGS -X 'gxx/cmd/cli.defaultVersion=$VERSION' -X 'gxx/cmd/cli.defaultAuthor=$AUTHOR' -X 'gxx/cmd/cli.defaultBuildDate=$BUILD_DATE'"

    # 构建命令
    BUILD_CMD="env GOOS=$GOOS GOARCH=$GOARCH go build -ldflags \"$LDFLAGS\" -o $OUTPUT_FILE ./cmd/main.go"
    
    # 如果启用了嵌入指纹库
    if [ "$EMBED" == "true" ]; then
        BUILD_CMD="$BUILD_CMD -tags embed"
    fi

    # 执行构建
    if ! eval $BUILD_CMD; then
        echo "${PLATFORM}/${ARCH} 的构建失败。"
        exit 1
    fi

    # 使用 UPX 压缩可执行文件（如果安装了 upx）
    if command -v upx &> /dev/null; then
        if [ "$GOOS" != "windows" ] || [ "$GOARCH" != "arm64" ]; then
            if [ "$GOOS" != "darwin" ]; then
                echo "正在使用 UPX 压缩 ${OUTPUT_FILE}..."
                if ! upx $OUTPUT_FILE; then
                    echo "${PLATFORM}/${ARCH} 的 UPX 压缩失败。"
                    exit 1
                fi
            else
                echo "跳过对 macOS 的 UPX 压缩。"
            fi
        else
            echo "跳过对 Windows ARM64 的 UPX 压缩。"
        fi
    else
        echo "未找到 UPX，跳过压缩步骤。"
    fi

    # 压缩成 zip 文件
    ZIP_FILE="$BUILD_DIR/$OUTPUT_ZIP_NAME.zip"
    if ! zip -j "$ZIP_FILE" "$OUTPUT_FILE"; then
        echo "${PLATFORM}/${ARCH} 的 ZIP 打包失败。"
        exit 1
    fi

    rm "$OUTPUT_FILE"
    echo "${PLATFORM}/${ARCH} 的构建和打包完成：${ZIP_FILE}"
}

# 构建所有平台版本
build_all() {
    local DEBUG=$1
    local EMBED=$2
    
    # 构建所有支持的平台和架构组合
    build "darwin" "amd64" "$DEBUG" "$EMBED"
    build "darwin" "arm64" "$DEBUG" "$EMBED"
    build "linux" "amd64" "$DEBUG" "$EMBED"
    build "linux" "arm64" "$DEBUG" "$EMBED"
    build "windows" "amd64" "$DEBUG" "$EMBED"
    build "windows" "arm64" "$DEBUG" "$EMBED"
}

# 解析命令行参数
DEBUG=false
EMBED=false
CLEAN=false
ALL=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -v|--version)
            show_version
            exit 0
            ;;
        -c|--clean)
            CLEAN=true
            shift
            ;;
        -a|--all)
            ALL=true
            shift
            ;;
        -d|--debug)
            DEBUG=true
            shift
            ;;
        -e|--embed)
            EMBED=true
            shift
            ;;
        *)
            echo "未知选项: $1"
            show_help
            exit 1
            ;;
    esac
done

# 清理构建目录
if [ "$CLEAN" == "true" ]; then
    clean_build
    exit 0
fi

# 构建
if [ "$ALL" == "true" ]; then
    build_all "$DEBUG" "$EMBED"
else
    # 默认构建当前平台
    GOOS=$(go env GOOS)
    GOARCH=$(go env GOARCH)
    build "$GOOS" "$GOARCH" "$DEBUG" "$EMBED"
fi

echo "构建完成！"

