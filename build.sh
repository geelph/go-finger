#!/bin/bash

# 设置版本号
VERSION="1.0.0"
BUILD_DATE=$(date "+%Y-%m-%d")
COMMIT_ID=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 设置输出目录
OUTPUT_DIR="dist"
mkdir -p $OUTPUT_DIR

# 检查是否启用嵌入模式
EMBED_MODE=false
while [[ $# -gt 0 ]]; do
  case $1 in
    --embed)
      EMBED_MODE=true
      shift
      ;;
    *)
      shift
      ;;
  esac
done

# 编译函数
build() {
    local os=$1
    local arch=$2
    local ext=$3
    
    echo "Building for $os/$arch..."
    
    # 设置输出文件名
    local output="$OUTPUT_DIR/gxx-$VERSION-$os-$arch$ext"
    
    # 设置环境变量
    export GOOS=$os
    export GOARCH=$arch
    
    # 编译
    go build -ldflags "-s -w -X 'main.Version=$VERSION' -X 'main.BuildDate=$BUILD_DATE' -X 'main.CommitID=$COMMIT_ID'" -o $output
    
    if [ $? -eq 0 ]; then
        echo "✅ Successfully built $output"
    else
        echo "❌ Failed to build for $os/$arch"
        return 1
    fi
    
    # 创建包含指纹库的发布包
    local package_dir="$OUTPUT_DIR/gxx-$VERSION-$os-$arch"
    mkdir -p "$package_dir"
    cp $output "$package_dir/gxx$ext"
    
    # 如果不是嵌入模式，复制finger目录
    if [ "$EMBED_MODE" = "false" ]; then
        cp -r finger "$package_dir/"
    fi
    
    cp README.md "$package_dir/"
    
    # 创建压缩包
    if [ "$os" = "windows" ]; then
        (cd "$OUTPUT_DIR" && zip -r "gxx-$VERSION-$os-$arch.zip" "gxx-$VERSION-$os-$arch")
    else
        (cd "$OUTPUT_DIR" && tar -czf "gxx-$VERSION-$os-$arch.tar.gz" "gxx-$VERSION-$os-$arch")
    fi
    
    # 清理临时目录
    rm -rf "$package_dir"
}

# 清理旧的构建
echo "Cleaning old builds..."
rm -rf $OUTPUT_DIR/*

# 检查指纹库目录是否存在
if [ ! -d "finger" ]; then
    echo "❌ Error: 'finger' directory not found!"
    echo "Please create a 'finger' directory with your fingerprint files before building."
    exit 1
fi

# 检查是否有demo.yaml文件
if [ ! -f "finger/demo.yaml" ] && [ ! -f "finger/demo.yml" ]; then
    echo "⚠️ Warning: No 'demo.yaml' found in the finger directory."
    echo "The application expects at least one demo fingerprint file."
fi

# 如果是嵌入模式，修改embed.go文件
if [ "$EMBED_MODE" = "true" ]; then
    echo "启用嵌入模式，修改embed.go文件..."
    # 备份原文件
    cp utils/finger/embed.go utils/finger/embed.go.bak
    
    # 取消注释embed指令
    sed -i.bak 's|// //go:embed all:finger|//go:embed all:finger|g' utils/finger/embed.go
    
    echo "已启用嵌入模式，编译完成后将恢复原文件"
fi

# 构建各平台版本
build "linux" "amd64" ""
build "darwin" "amd64" ""
build "darwin" "arm64" ""
build "windows" "amd64" ".exe"

# 如果是嵌入模式，恢复embed.go文件
if [ "$EMBED_MODE" = "true" ]; then
    echo "恢复embed.go文件..."
    mv utils/finger/embed.go.bak utils/finger/embed.go
fi

echo ""
echo "Build completed! Packages are available in the '$OUTPUT_DIR' directory."
if [ "$EMBED_MODE" = "true" ]; then
    echo "指纹库已嵌入到二进制文件中。"
else
    echo "Each package contains the binary and the 'finger' directory."
fi

echo ""
echo "使用说明:"
echo "1. 默认模式 (./build.sh): 生成的二进制文件需要与finger目录一起部署"
echo "2. 嵌入模式 (./build.sh --embed): 生成的二进制文件已包含指纹库，无需额外部署finger目录" 