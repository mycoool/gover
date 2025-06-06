package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gover/controllers"
	"gover/models"

	"github.com/beego/beego/v2/server/web"
)

// 版本信息，在构建时通过 ldflags 注入
var (
	Version   = "dev"     // 版本号
	BuildTime = "unknown" // 构建时间
	GitCommit = "unknown" // Git 提交哈希
)

func init() {
	// Beego 配置将通过 app.conf 文件自动加载
}

// clearSessionFiles 清除 Session 相关文件
func clearSessionFiles() {
	// 清除可能的 Session 临时文件
	tempDirs := []string{
		"/tmp",
		os.TempDir(),
		"./tmp",
		"./sessions",
	}

	for _, dir := range tempDirs {
		if _, err := os.Stat(dir); err == nil {
			// 查找并删除 Session 文件
			pattern := filepath.Join(dir, "session_*")
			files, _ := filepath.Glob(pattern)
			for _, file := range files {
				if err := os.Remove(file); err != nil {
					fmt.Printf("警告: 删除文件失败 %s: %v\n", file, err)
				}
			}

			// 查找并删除 gorilla session 文件
			pattern = filepath.Join(dir, "gorilla_*")
			files, _ = filepath.Glob(pattern)
			for _, file := range files {
				if err := os.Remove(file); err != nil {
					fmt.Printf("警告: 删除文件失败 %s: %v\n", file, err)
				}
			}
		}
	}
}

func main() {
	// 解析命令行参数
	clearSessions := flag.Bool("clear-sessions", false, "清除所有 Session 文件并退出")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	// 如果指定了显示版本参数
	if *showVersion {
		fmt.Printf("🚀 Gover - Git 版本管理工具\n")
		fmt.Printf("📋 版本: %s\n", Version)
		fmt.Printf("🕐 构建时间: %s\n", BuildTime)
		os.Exit(0)
	}

	// 如果指定了清除 Session 参数
	if *clearSessions {
		fmt.Printf("🧹 正在清除 Session 数据...\n")
		clearSessionFiles()
		controllers.ResetSessionStore()
		fmt.Printf("✅ 所有 Session 数据已清除\n")
		fmt.Printf("💡 所有用户需要重新登录\n")
		os.Exit(0)
	}

	// 立即输出程序信息，覆盖 Beego 的配置警告
	fmt.Printf("\n🚀 Gover %s - Git 版本管理工具启动中...\n", Version)
	fmt.Printf("📝 使用 YAML 配置文件 (config.yaml)\n")

	// 初始化配置（会自动创建 app.conf 文件）
	models.InitConfig()

	// 设置路由
	web.Router("/", &controllers.VersionController{}, "get,post:Index")
	web.Router("/checkout", &controllers.VersionController{}, "post:Checkout")
	web.Router("/login", &controllers.AuthController{}, "get,post:Login")
	web.Router("/logout", &controllers.AuthController{}, "get:Logout")

	// 从 YAML 配置覆盖端口和主机设置
	web.BConfig.Listen.HTTPPort = models.AppConfig.Server.Port
	web.BConfig.Listen.HTTPAddr = models.AppConfig.Server.Host

	// 启动服务
	fmt.Printf("✅ 配置加载完成\n")
	fmt.Printf("📡 服务地址: http://%s:%d\n", models.AppConfig.Server.Host, models.AppConfig.Server.Port)
	fmt.Printf("👤 用户名: %s\n", models.AppConfig.Auth.Username)
	fmt.Printf("🔐 密码: %s\n", models.AppConfig.Auth.Password)
	fmt.Printf("📁 管理 %d 个项目\n", len(models.AppConfig.GetEnabledProjects()))
	fmt.Printf("🌟 服务启动中...\n\n")

	web.Run()
}
