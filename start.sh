#!/bin/bash

echo "🚀 正在启动 Gover - Git 版本管理工具..."
echo "📝 配置文件: config.yaml"
echo "📁 支持多项目管理"
echo ""

# 构建项目
echo "🔨 构建项目..."
go build -o gover .

if [ $? -eq 0 ]; then
    echo "✅ 构建成功"
    echo ""
    
    # 启动服务
    echo "🌟 启动服务..."
    ./gover
else
    echo "❌ 构建失败"
    exit 1
fi 