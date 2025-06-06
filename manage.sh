#!/bin/bash

# Gover - Git 版本管理工具 - 管理脚本

case "$1" in
    "start")
        echo "🚀 启动 Gover..."
        ./gover
        ;;
    "clear-sessions")
        echo "🧹 清除所有 Session..."
        ./gover -clear-sessions
        echo "✅ 操作完成"
        ;;
    "build")
        echo "🔨 构建项目..."
        go build -o gover .
        echo "✅ 构建完成"
        ;;
    "restart")
        echo "🔄 重启服务..."
        pkill -f gover 2>/dev/null || true
        sleep 1
        ./gover &
        echo "✅ 服务已重启"
        ;;
    "stop")
        echo "⏹️ 停止服务..."
        pkill -f gover
        echo "✅ 服务已停止"
        ;;
    "status")
        if pgrep -f gover > /dev/null; then
            echo "✅ 服务正在运行"
            echo "进程信息:"
            ps aux | grep gover | grep -v grep
        else
            echo "❌ 服务未运行"
        fi
        ;;
    *)
        echo "🛠️  Gover - Git 版本管理工具 - 管理脚本"
        echo ""
        echo "用法: $0 {start|stop|restart|status|build|clear-sessions}"
        echo ""
        echo "命令说明:"
        echo "  start          启动服务"
        echo "  stop           停止服务"
        echo "  restart        重启服务"
        echo "  status         查看服务状态"
        echo "  build          构建项目"
        echo "  clear-sessions 清除所有 Session"
        echo ""
        exit 1
        ;;
esac 