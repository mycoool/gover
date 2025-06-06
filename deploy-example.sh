#!/bin/bash

# Gover 部署示例脚本
# 演示如何在新服务器上快速部署 gover

set -e

echo "🚀 Gover 部署示例"
echo "=================="
echo

# 检查参数
if [ $# -eq 0 ]; then
    echo "用法: $0 <发布包文件>"
    echo "示例: $0 gover-v1.0.0-Linux-x86_64.tar.gz"
    echo
    echo "或者提供下载 URL:"
    echo "示例: $0 https://github.com/your-repo/gover/releases/download/v1.0.0/gover-v1.0.0-Linux-x86_64.tar.gz"
    exit 1
fi

PACKAGE_SOURCE="$1"
DEPLOY_DIR="/opt/gover"
SERVICE_USER="gover"

echo "📦 部署源: $PACKAGE_SOURCE"
echo "📁 部署目录: $DEPLOY_DIR"
echo "👤 服务用户: $SERVICE_USER"
echo

# 检查是否有 root 权限
if [ "$EUID" -ne 0 ]; then
    echo "❌ 此脚本需要 root 权限运行"
    echo "请使用: sudo $0 $PACKAGE_SOURCE"
    exit 1
fi

# 1. 下载或复制发布包
echo "📥 1. 获取发布包..."
if [[ $PACKAGE_SOURCE == http* ]]; then
    PACKAGE_FILE=$(basename "$PACKAGE_SOURCE")
    echo "   从 URL 下载: $PACKAGE_SOURCE"
    if command -v wget &> /dev/null; then
        wget -O "/tmp/$PACKAGE_FILE" "$PACKAGE_SOURCE"
    elif command -v curl &> /dev/null; then
        curl -L -o "/tmp/$PACKAGE_FILE" "$PACKAGE_SOURCE"
    else
        echo "❌ 未找到 wget 或 curl，无法下载文件"
        exit 1
    fi
    PACKAGE_PATH="/tmp/$PACKAGE_FILE"
else
    PACKAGE_PATH="$PACKAGE_SOURCE"
    if [ ! -f "$PACKAGE_PATH" ]; then
        echo "❌ 文件不存在: $PACKAGE_PATH"
        exit 1
    fi
fi

echo "✅ 发布包就绪: $PACKAGE_PATH"

# 2. 创建用户
echo "👤 2. 创建服务用户..."
if id "$SERVICE_USER" &>/dev/null; then
    echo "   用户 $SERVICE_USER 已存在"
else
    useradd --system --home-dir "$DEPLOY_DIR" --shell /bin/bash "$SERVICE_USER"
    echo "✅ 已创建用户: $SERVICE_USER"
fi

# 3. 创建部署目录
echo "📁 3. 创建部署目录..."
mkdir -p "$DEPLOY_DIR"
chown "$SERVICE_USER:$SERVICE_USER" "$DEPLOY_DIR"
echo "✅ 部署目录就绪: $DEPLOY_DIR"

# 4. 解压部署文件
echo "📦 4. 解压部署文件..."
cd "$DEPLOY_DIR"
if [[ $PACKAGE_PATH == *.tar.gz ]]; then
    tar -xzf "$PACKAGE_PATH" --strip-components=1
elif [[ $PACKAGE_PATH == *.zip ]]; then
    unzip -j "$PACKAGE_PATH"
else
    echo "❌ 不支持的文件格式: $PACKAGE_PATH"
    exit 1
fi

echo "✅ 文件解压完成"

# 5. 设置权限
echo "🔒 5. 设置文件权限..."
chown -R "$SERVICE_USER:$SERVICE_USER" "$DEPLOY_DIR"
chmod +x "$DEPLOY_DIR/gover"
echo "✅ 权限设置完成"

# 6. 创建配置文件
echo "⚙️ 6. 创建配置文件..."
if [ ! -f "$DEPLOY_DIR/config.yaml" ]; then
    cp "$DEPLOY_DIR/config.yaml.example" "$DEPLOY_DIR/config.yaml"
    echo "✅ 已创建配置文件"
    echo "⚠️ 请编辑 $DEPLOY_DIR/config.yaml 设置您的项目路径和认证信息"
else
    echo "   配置文件已存在，跳过"
fi

# 7. 创建 systemd 服务
echo "🔧 7. 创建 systemd 服务..."
cat > /etc/systemd/system/gover.service << EOF
[Unit]
Description=Gover Git Version Manager
Documentation=https://github.com/your-repo/gover
After=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
WorkingDirectory=$DEPLOY_DIR
ExecStart=$DEPLOY_DIR/gover
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=gover

# 安全设置
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$DEPLOY_DIR

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
echo "✅ systemd 服务已创建"

# 8. 启用和启动服务
echo "🚀 8. 启用并启动服务..."
systemctl enable gover
echo "✅ 服务已启用（开机自启）"

# 9. 测试配置
echo "🧪 9. 测试应用..."
sudo -u "$SERVICE_USER" "$DEPLOY_DIR/gover" --version
echo "✅ 应用测试通过"

# 10. 清理
if [[ $PACKAGE_SOURCE == http* ]] && [ -f "/tmp/$PACKAGE_FILE" ]; then
    rm "/tmp/$PACKAGE_FILE"
    echo "🧹 已清理临时文件"
fi

echo
echo "🎉 部署完成！"
echo "================"
echo
echo "📁 部署目录: $DEPLOY_DIR"
echo "👤 运行用户: $SERVICE_USER"
echo "⚙️ 配置文件: $DEPLOY_DIR/config.yaml"
echo
echo "🔧 管理命令:"
echo "   启动服务: sudo systemctl start gover"
echo "   停止服务: sudo systemctl stop gover"
echo "   重启服务: sudo systemctl restart gover"
echo "   查看状态: sudo systemctl status gover"
echo "   查看日志: sudo journalctl -u gover -f"
echo
echo "⚠️ 重要提醒:"
echo "1. 请编辑配置文件设置您的项目路径: sudo vim $DEPLOY_DIR/config.yaml"
echo "2. 确保项目目录对 $SERVICE_USER 用户可访问"
echo "3. 如需修复 Git 权限: sudo -u $SERVICE_USER $DEPLOY_DIR/gover --fix-git"
echo "4. 修改配置后重启服务: sudo systemctl restart gover"
echo
echo "🌐 服务将在配置的端口启动（默认 8088）"
echo "🔗 访问地址: http://your-server-ip:8088" 