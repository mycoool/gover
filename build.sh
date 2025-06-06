#!/bin/bash

# Gover 构建脚本
# 将模板文件嵌入到二进制文件中，便于部署

set -e

# 显示帮助信息
show_help() {
    echo "Gover 构建脚本"
    echo ""
    echo "用法: $0 [VERSION]"
    echo ""
    echo "VERSION 选项:"
    echo "  dev        - 开发版本 (默认)"
    echo "  prod       - 生产版本"
    echo "  v1.0.0     - 指定版本号"
    echo "  auto       - 自动版本 (日期+Git提交)"
    echo "  release    - 使用最新Git标签"
    echo ""
    echo "示例:"
    echo "  $0              # 构建开发版本"
    echo "  $0 v1.0.0       # 构建指定版本"
    echo "  $0 prod         # 构建生产版本"
    echo "  $0 auto         # 自动生成版本号"
    echo "  $0 release      # 使用Git标签版本"
    echo ""
    exit 0
}

# 检查帮助参数
if [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
    show_help
fi

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 获取版本信息
VERSION=${1:-"dev"}
BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 版本验证和处理
if [[ "$VERSION" == "auto" ]]; then
    # 自动生成版本号：日期+提交哈希
    VERSION="v$(date '+%Y.%m.%d')-${GIT_COMMIT}"
elif [[ "$VERSION" == "release" ]]; then
    # 从 Git 标签获取最新版本
    LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v1.0.0")
    VERSION="$LATEST_TAG"
fi

echo -e "${CYAN}🚀 Gover 构建脚本${NC}"
echo -e "${BLUE}📋 版本: ${VERSION}${NC}"
echo -e "${BLUE}🕐 构建时间: ${BUILD_TIME}${NC}"
echo -e "${BLUE}🔗 Git 提交: ${GIT_COMMIT}${NC}"
echo

# 检查 Go 版本
echo -e "${YELLOW}🔍 检查 Go 环境...${NC}"
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go 未安装或不在 PATH 中${NC}"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo -e "${GREEN}✅ Go 版本: ${GO_VERSION}${NC}"

# 检查必要的文件
echo -e "${YELLOW}📂 检查项目文件...${NC}"
REQUIRED_FILES=("main.go" "embed.go" "views/version/index.html" "views/auth/login.html")
for file in "${REQUIRED_FILES[@]}"; do
    if [[ ! -f "$file" ]]; then
        echo -e "${RED}❌ 缺少必要文件: ${file}${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ 找到: ${file}${NC}"
done

# 清理旧的构建文件
echo -e "${YELLOW}🧹 清理旧的构建文件...${NC}"
rm -f gover gover-embedded gover-*.tar.gz

# 构建项目
echo -e "${YELLOW}🔨 正在构建项目...${NC}"
echo -e "${BLUE}   目标: gover (嵌入模板)${NC}"

# 设置构建标志
LDFLAGS="-X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}' -X 'main.GitCommit=${GIT_COMMIT}' -w -s"

# 构建二进制文件
if go build -ldflags="${LDFLAGS}" -o gover; then
    echo -e "${GREEN}✅ 构建成功: gover${NC}"
else
    echo -e "${RED}❌ 构建失败${NC}"
    exit 1
fi

# 显示文件信息
echo -e "${YELLOW}📊 构建结果:${NC}"
ls -lh gover
echo

# 测试二进制文件
echo -e "${YELLOW}🧪 测试二进制文件...${NC}"
if ./gover --version; then
    echo -e "${GREEN}✅ 版本检查通过${NC}"
else
    echo -e "${RED}❌ 版本检查失败${NC}"
    exit 1
fi

# 创建发布包
echo -e "${YELLOW}📦 创建发布包...${NC}"
PACKAGE_NAME="gover-${VERSION}-$(uname -s)-$(uname -m).tar.gz"

# 创建临时目录
TEMP_DIR=$(mktemp -d)
PACKAGE_DIR="${TEMP_DIR}/gover-${VERSION}"
mkdir -p "${PACKAGE_DIR}"

# 复制文件到包目录
cp gover "${PACKAGE_DIR}/"
cp config.yaml "${PACKAGE_DIR}/config.yaml.example"
cp README.md "${PACKAGE_DIR}/" 2>/dev/null || echo "# Gover" > "${PACKAGE_DIR}/README.md"

# 创建简单的部署说明
cat > "${PACKAGE_DIR}/DEPLOY.md" << 'EOF'
# Gover 部署说明

## 快速部署

1. 解压文件：
   ```bash
   tar -xzf gover-*.tar.gz
   cd gover-*
   ```

2. 复制配置文件：
   ```bash
   cp config.yaml.example config.yaml
   ```

3. 编辑配置文件：
   ```bash
   vim config.yaml
   ```

4. 运行应用：
   ```bash
   ./gover
   ```

## 模板文件

此版本已将模板文件嵌入到二进制文件中，无需额外的 views 目录。

## 权限问题

如果遇到 Git 权限问题，运行：
```bash
./gover --fix-git
```

## 服务模式

建议使用 systemd 或 supervisor 管理服务：

```bash
# 创建 systemd 服务文件
sudo tee /etc/systemd/system/gover.service > /dev/null << EOL
[Unit]
Description=Gover Git Version Manager
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/path/to/gover
ExecStart=/path/to/gover/gover
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOL

# 启用并启动服务
sudo systemctl enable gover
sudo systemctl start gover
```
EOF

# 创建压缩包
cd "${TEMP_DIR}"
tar -czf "${PACKAGE_NAME}" "gover-${VERSION}"
mv "${PACKAGE_NAME}" "${OLDPWD}/"
cd "${OLDPWD}"

# 清理临时目录
rm -rf "${TEMP_DIR}"

echo -e "${GREEN}✅ 发布包创建成功: ${PACKAGE_NAME}${NC}"

# 显示最终结果
echo
echo -e "${PURPLE}🎉 构建完成！${NC}"
echo -e "${CYAN}📁 生成的文件:${NC}"
echo -e "   • ${GREEN}gover${NC} - 主程序（嵌入模板）"
echo -e "   • ${GREEN}${PACKAGE_NAME}${NC} - 发布包"
echo
echo -e "${CYAN}💡 使用说明:${NC}"
echo -e "   • 开发测试: ${YELLOW}./gover --debug${NC}"
echo -e "   • 快速模式: ${YELLOW}./gover --fast --skip-fetch${NC}"
echo -e "   • 查看版本: ${YELLOW}./gover --version${NC}"
echo -e "   • 修复权限: ${YELLOW}./gover --fix-git${NC}"
echo -e "   • 生产部署: 解压 ${YELLOW}${PACKAGE_NAME}${NC} 到目标服务器"
echo
echo -e "${CYAN}🔧 构建其他版本:${NC}"
echo -e "   • ${YELLOW}./build.sh prod${NC}        - 生产版本"
echo -e "   • ${YELLOW}./build.sh v2.0.0${NC}      - 指定版本"
echo -e "   • ${YELLOW}./build.sh auto${NC}        - 自动版本"
echo -e "   • ${YELLOW}./build.sh release${NC}     - Git标签版本"
echo -e "   • ${YELLOW}./build.sh --help${NC}      - 查看帮助"
echo
echo -e "${GREEN}🚀 部署只需要二进制文件和配置文件！${NC}" 