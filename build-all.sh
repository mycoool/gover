#!/bin/bash

# Gover 多平台构建脚本
# 将模板文件嵌入到二进制文件中，支持多个平台

set -e

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

echo -e "${CYAN}🚀 Gover 多平台构建脚本${NC}"
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
rm -rf dist/
mkdir -p dist/

# 设置构建标志
LDFLAGS="-X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}' -X 'main.GitCommit=${GIT_COMMIT}' -w -s"

# 定义构建目标
platforms=(
    "linux/amd64"
    "linux/arm64"
    "linux/386"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
    "windows/386"
)

echo -e "${YELLOW}🔨 开始多平台构建...${NC}"
echo

# 构建函数
build_platform() {
    local platform=$1
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    echo -e "${BLUE}📦 构建 ${GOOS}/${GOARCH}...${NC}"
    
    # 设置二进制文件名
    local binary_name="gover"
    if [ "$GOOS" = "windows" ]; then
        binary_name="gover.exe"
    fi
    
    # 构建二进制文件（静态编译）
    if env GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o "dist/${binary_name}" .; then
        echo -e "   ${GREEN}✅ 构建成功${NC}"
    else
        echo -e "   ${RED}❌ 构建失败${NC}"
        return 1
    fi
    
    # 创建发布包
    local package_dir="dist/gover-${VERSION}-${GOOS}-${GOARCH}"
    mkdir -p "${package_dir}"
    
    # 复制文件
    cp "dist/${binary_name}" "${package_dir}/"
    cp config.yaml "${package_dir}/config.yaml.example"
    cp README.md "${package_dir}/" 2>/dev/null || echo "# Gover v${VERSION}" > "${package_dir}/README.md"
    
    # 为 Windows 创建批处理文件
    if [ "$GOOS" = "windows" ]; then
        cat > "${package_dir}/start.bat" << 'EOF'
@echo off
echo Starting Gover...
gover.exe
pause
EOF
        cat > "${package_dir}/install-service.bat" << 'EOF'
@echo off
echo Installing Gover as Windows Service...
sc create "Gover" binPath= "%~dp0gover.exe" start= auto
echo Service installed. Use 'sc start Gover' to start the service.
pause
EOF
    fi
    
    # 创建部署说明
    cat > "${package_dir}/DEPLOY.md" << EOF
# Gover v${VERSION} 部署说明

## 平台信息
- 操作系统: ${GOOS}
- 架构: ${GOARCH}
- 构建时间: ${BUILD_TIME}

## 快速部署

1. 复制配置文件：
   \`\`\`bash
   cp config.yaml.example config.yaml
   \`\`\`

2. 编辑配置文件（设置项目路径、用户名密码等）

3. 运行应用：
EOF

    if [ "$GOOS" = "windows" ]; then
        echo "   - 直接运行: \`gover.exe\`" >> "${package_dir}/DEPLOY.md"
        echo "   - 或双击: \`start.bat\`" >> "${package_dir}/DEPLOY.md"
        echo "   - 安装服务: \`install-service.bat\`" >> "${package_dir}/DEPLOY.md"
    else
        echo "   \`\`\`bash" >> "${package_dir}/DEPLOY.md"
        echo "   ./gover" >> "${package_dir}/DEPLOY.md"
        echo "   \`\`\`" >> "${package_dir}/DEPLOY.md"
    fi

    cat >> "${package_dir}/DEPLOY.md" << 'EOF'

## 模板文件

此版本已将模板文件嵌入到二进制文件中，无需额外的 views 目录。

## 兼容性

- 使用静态编译，无 glibc 版本依赖
- 支持较老的 Linux 发行版（CentOS 7、Ubuntu 16.04+）
- 单二进制文件，无需额外依赖库

## 权限问题

如果遇到 Git 权限问题，运行：
```bash
./gover --fix-git
```

## 更多选项

- 查看版本: `./gover --version`
- 调试模式: `./gover --debug`
- 清除会话: `./gover --clear-sessions`
EOF
    
    # 创建压缩包
    cd dist/
    if [ "$GOOS" = "windows" ]; then
        local package_name="gover-${VERSION}-${GOOS}-${GOARCH}.zip"
        zip -r "${package_name}" "$(basename "${package_dir}")" > /dev/null
        echo -e "   ${GREEN}📦 创建: ${package_name}${NC}"
    else
        local package_name="gover-${VERSION}-${GOOS}-${GOARCH}.tar.gz"
        tar -czf "${package_name}" "$(basename "${package_dir}")" 2>/dev/null
        echo -e "   ${GREEN}📦 创建: ${package_name}${NC}"
    fi
    cd ..
    
    # 清理临时文件
    rm -rf "${package_dir}" "dist/${binary_name}"
}

# 并行构建所有平台
success_count=0
total_count=${#platforms[@]}

for platform in "${platforms[@]}"; do
    if build_platform "$platform"; then
        ((success_count++))
    fi
done

echo
echo -e "${PURPLE}🎉 构建完成！${NC}"
echo -e "${CYAN}📊 构建统计:${NC}"
echo -e "   • 成功: ${GREEN}${success_count}${NC}/${total_count}"
echo -e "   • 失败: ${RED}$((total_count - success_count))${NC}/${total_count}"

echo
echo -e "${CYAN}📁 生成的发布包:${NC}"
ls -la dist/*.{tar.gz,zip} 2>/dev/null || echo "   无发布包生成"

echo
echo -e "${CYAN}💡 使用说明:${NC}"
echo -e "   • 解压对应平台的包到目标服务器"
echo -e "   • 复制配置文件: ${YELLOW}cp config.yaml.example config.yaml${NC}"
echo -e "   • 编辑配置文件并运行程序"
echo -e "   • 所有平台都已嵌入模板文件，无需额外依赖"

if [ $success_count -eq $total_count ]; then
    echo
    echo -e "${GREEN}🚀 所有平台构建成功！可以分发这些文件了。${NC}"
else
    echo
    echo -e "${YELLOW}⚠️ 部分平台构建失败，请检查错误信息。${NC}"
fi 