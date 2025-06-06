#!/bin/bash

# Gover 多平台构建脚本

set -e

echo "🚀 开始构建 Gover..."

# 获取版本信息
if git describe --tags --exact-match HEAD >/dev/null 2>&1; then
    VERSION=$(git describe --tags --exact-match HEAD)
    echo "📋 使用标签版本: $VERSION"
else
    VERSION="dev-$(git rev-parse --short HEAD)"
    echo "📋 使用开发版本: $VERSION"
fi

BUILD_TIME=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
GIT_COMMIT=$(git rev-parse --short HEAD)

echo "🕐 构建时间: $BUILD_TIME"
echo "📝 Git 提交: $GIT_COMMIT"

# 构建选项
LDFLAGS="-s -w -X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}' -X 'main.GitCommit=${GIT_COMMIT}'"

# 创建构建目录
mkdir -p dist

echo ""
echo "🏗️ 开始多平台构建..."

# 定义构建目标
platforms=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

for platform in "${platforms[@]}"
do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    if [ "$GOOS" = "windows" ]; then
        output_name="gover-${VERSION}-${GOOS}-${GOARCH}.exe"
    else
        output_name="gover-${VERSION}-${GOOS}-${GOARCH}"
    fi
    
    echo "📦 构建 ${GOOS}/${GOARCH}..."
    
    env GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "${LDFLAGS}" -o "dist/${output_name}" .
    
    if [ $? -ne 0 ]; then
        echo "❌ 构建 ${GOOS}/${GOARCH} 失败"
        exit 1
    fi
    
    # 创建压缩包
    cd dist
    if [ "$GOOS" = "windows" ]; then
        zip "${output_name}.zip" "${output_name}" ../config.yaml ../README.md
        rm "${output_name}"
        echo "✅ 已创建: ${output_name}.zip"
    else
        tar -czf "${output_name}.tar.gz" "${output_name}" ../config.yaml ../README.md
        rm "${output_name}"
        echo "✅ 已创建: ${output_name}.tar.gz"
    fi
    cd ..
done

echo ""
echo "🎉 构建完成！"
echo "📁 构建产物位于 dist/ 目录："
ls -la dist/

echo ""
echo "📋 版本信息验证："
# 验证一个构建产物的版本信息
if [ -f "dist/gover-${VERSION}-linux-amd64.tar.gz" ]; then
    cd dist
    tar -xzf "gover-${VERSION}-linux-amd64.tar.gz" "gover-${VERSION}-linux-amd64"
    ./gover-${VERSION}-linux-amd64 -version
    rm "gover-${VERSION}-linux-amd64"
    cd ..
fi

echo ""
echo "�� 构建完成！可以分发这些文件了。" 