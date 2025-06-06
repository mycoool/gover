#!/bin/bash

# Gover 服务器部署脚本

set -e

echo "🚀 Gover 服务器部署脚本"
echo "=========================="

# 检查参数
if [ $# -eq 0 ]; then
    echo "使用方法："
    echo "  $0 <部署目录> [版本号]"
    echo ""
    echo "示例："
    echo "  $0 /var/www/gover v0.0.6"
    echo "  $0 /opt/gover latest"
    exit 1
fi

DEPLOY_DIR="$1"
VERSION="${2:-latest}"

echo "📁 部署目录: $DEPLOY_DIR"
echo "📋 版本: $VERSION"

# 创建部署目录
sudo mkdir -p "$DEPLOY_DIR"

# 切换到部署目录
cd "$DEPLOY_DIR"

# 如果目录已存在 Git 仓库，更新它
if [ -d ".git" ]; then
    echo "🔄 更新现有 Git 仓库..."
    sudo git fetch --all --tags
    if [ "$VERSION" = "latest" ]; then
        LATEST_TAG=$(git describe --tags --abbrev=0)
        echo "📋 最新版本: $LATEST_TAG"
        sudo git checkout "$LATEST_TAG"
    else
        sudo git checkout "$VERSION"
    fi
else
    echo "📥 克隆 Git 仓库..."
    sudo git clone https://github.com/mycoool/gover.git .
    sudo git fetch --all --tags
    
    if [ "$VERSION" = "latest" ]; then
        LATEST_TAG=$(git describe --tags --abbrev=0)
        echo "📋 最新版本: $LATEST_TAG"
        sudo git checkout "$LATEST_TAG"
    else
        sudo git checkout "$VERSION"
    fi
fi

# 下载对应的预编译二进制文件
echo "📦 下载预编译二进制文件..."

# 检测系统架构
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

case $ARCH in
    x86_64) GOARCH="amd64" ;;
    aarch64|arm64) GOARCH="arm64" ;;
    i386|i686) GOARCH="386" ;;
    *) echo "❌ 不支持的架构: $ARCH"; exit 1 ;;
esac

case $OS in
    linux) GOOS="linux" ;;
    darwin) GOOS="darwin" ;;
    *) echo "❌ 不支持的操作系统: $OS"; exit 1 ;;
esac

if [ "$VERSION" = "latest" ]; then
    CURRENT_VERSION=$(git describe --tags --abbrev=0)
else
    CURRENT_VERSION="$VERSION"
fi

BINARY_NAME="gover-${CURRENT_VERSION}-${GOOS}-${GOARCH}"
DOWNLOAD_URL="https://github.com/mycoool/gover/releases/download/${CURRENT_VERSION}/${BINARY_NAME}.tar.gz"

echo "🌐 下载地址: $DOWNLOAD_URL"

# 下载并解压
sudo wget -O "${BINARY_NAME}.tar.gz" "$DOWNLOAD_URL"
sudo tar -xzf "${BINARY_NAME}.tar.gz"
sudo mv gover gover-binary
sudo chmod +x gover-binary
sudo rm "${BINARY_NAME}.tar.gz"

# 创建启动脚本
sudo tee gover > /dev/null << 'EOF'
#!/bin/bash

# 确保在正确的目录中运行
cd "$(dirname "$0")"

# 检查是否是 Git 仓库
if [ ! -d ".git" ]; then
    echo "❌ 错误: 当前目录不是 Git 仓库"
    echo "请确保在包含 .git 目录的项目根目录中运行此脚本"
    exit 1
fi

# 显示当前仓库信息
echo "📋 当前仓库信息:"
echo "   路径: $(pwd)"
echo "   分支: $(git branch --show-current 2>/dev/null || echo '未知')"
echo "   版本: $(git describe --tags 2>/dev/null || echo '无标签')"
echo "   标签数: $(git tag -l | wc -l)"
echo ""

# 运行应用
exec ./gover-binary "$@"
EOF

sudo chmod +x gover

# 设置配置文件权限（如果不存在则使用默认配置）
if [ ! -f config.yaml ]; then
    echo "📝 使用默认配置文件"
fi

sudo chmod 600 config.yaml 2>/dev/null || true

echo ""
echo "✅ 部署完成！"
echo ""
echo "🚀 启动应用:"
echo "   cd $DEPLOY_DIR"
echo "   sudo ./gover"
echo ""
echo "🔧 配置文件: $DEPLOY_DIR/config.yaml"
echo "📁 项目路径: $DEPLOY_DIR (包含完整 Git 仓库)"
echo ""
echo "📋 验证部署:"
echo "   cd $DEPLOY_DIR"
echo "   git tag -l  # 查看所有标签"
echo "   ./gover -version  # 查看版本信息" 