#!/bin/bash

# 设置应用名称和版本号
APP_NAME="gxx"
VERSION="1.1.0"

# 创建构建目录
BUILD_DIR="build"
mkdir -p $BUILD_DIR

# 构建函数
build() {
    GOOS=$1
    GOARCH=$2

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

    OUTPUT_NAME="${APP_NAME}_${VERSION}_${PLATFORM}_${ARCH}"

    echo "正在为 ${PLATFORM}/${ARCH} 构建..."

    # 设置输出文件名
    if [ "$GOOS" == "windows" ]; then
        OUTPUT_FILE="$BUILD_DIR/$OUTPUT_NAME.exe"
    else
        OUTPUT_FILE="$BUILD_DIR/$OUTPUT_NAME"
    fi

    # 构建可执行文件，使用 -ldflags "-w -s" 参数以减小文件大小，并检查是否成功构建。
    if ! env GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-w -s" -o $OUTPUT_FILE .; then
        echo "${PLATFORM}/${ARCH} 的构建失败。"
        exit 1
    fi

    # 使用 UPX 压缩可执行文件（如果安装了 upx），并且不是 Windows ARM64。
    if command -v upx &> /dev/null; then
      if [ "$GOOS" != "windows" ] || [ "$GOARCH" != "arm64" ]; then
          echo "正在使用 UPX 压缩 ${OUTPUT_FILE}..."
          if ! upx $OUTPUT_FILE; then
              echo "${PLATFORM}/${ARCH} 的 UPX 压缩失败。"
              exit 1
          fi
      else
          echo "跳过对 Windows ARM64 的 UPX 压缩。"
      fi

      else
      echo "未找到 UPX，跳过压缩步骤。"

fi

# 压缩成 zip 文件，并删除可执行文件，保留 zip 文件
ZIP_FILE="$BUILD_DIR/$OUTPUT_NAME.zip"

if ! zip -j "$ZIP_FILE" "$OUTPUT_FILE"; then
echo "${PLATFORM}/${ARCH} 的 ZIP 打包失败。"
exit 1
fi

rm "$OUTPUT_FILE"

echo "${PLATFORM}/${ARCH} 的构建和打包完成：${ZIP_FILE}"
}

# 构建 Windows 各版本
build windows amd64 # x64 架构
build windows 386   # x86 架构（即 amd32）
build windows arm64 # ARM64 架构

# 构建 macOS ARM64版本（默认是 arm64，如果需要 amd64，请调整）
build darwin arm64

# 构建 Linux 各版本
build linux amd64   # x64 架构
build linux 386     # x86 架构（即 amd32）

echo "所有平台的构建和打包已完成，文件保存在 $BUILD_DIR 目录下。"

