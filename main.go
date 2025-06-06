package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
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
	debugMode := flag.Bool("debug", false, "启用调试模式，显示详细的项目诊断信息")
	fixGitPermissions := flag.Bool("fix-git", false, "修复所有项目的 Git 权限问题并退出")
	fastMode := flag.Bool("fast", false, "启用快速模式，减少 Git 操作以提高响应速度")
	skipFetch := flag.Bool("skip-fetch", false, "跳过 Git fetch 操作，使用本地数据")
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

	// 如果指定了修复 Git 权限参数
	if *fixGitPermissions {
		fmt.Printf("🔧 正在修复 Git 权限问题...\n")

		// 先加载配置
		models.InitConfig()

		successCount := 0
		for _, project := range models.AppConfig.GetEnabledProjects() {
			fmt.Printf("📁 处理项目: %s (%s)\n", project.Name, project.Path)

			// 检查项目路径是否存在
			if _, err := os.Stat(project.Path); os.IsNotExist(err) {
				fmt.Printf("   ❌ 路径不存在: %s\n", project.Path)
				continue
			}

			// 添加安全目录配置
			cmd := exec.Command("git", "config", "--global", "--add", "safe.directory", project.Path)
			if err := cmd.Run(); err != nil {
				fmt.Printf("   ❌ 配置失败: %v\n", err)

				// 尝试其他方法
				fmt.Printf("   🔄 尝试手动修复，请运行:\n")
				fmt.Printf("      git config --global --add safe.directory %s\n", project.Path)
			} else {
				fmt.Printf("   ✅ 已添加安全目录配置\n")
				successCount++
			}
		}

		fmt.Printf("\n🎉 处理完成！成功修复 %d 个项目\n", successCount)
		fmt.Printf("💡 现在可以正常运行 gover 了\n")
		os.Exit(0)
	}

	// 立即输出程序信息，覆盖 Beego 的配置警告
	fmt.Printf("\n🚀 Gover %s - Git 版本管理工具启动中...\n", Version)
	fmt.Printf("📝 使用 YAML 配置文件 (config.yaml)\n")

	// 初始化配置（会自动创建 app.conf 文件）
	models.InitConfig()

	// 设置嵌入的模板文件
	if err := setupEmbeddedTemplates(); err != nil {
		fmt.Printf("❌ 模板设置失败: %v\n", err)
		os.Exit(1)
	}

	// 设置调试模式
	controllers.DebugMode = *debugMode
	if *debugMode {
		fmt.Printf("🐛 调试模式已启用\n")
	}

	// 设置性能模式
	controllers.FastMode = *fastMode
	controllers.SkipFetch = *skipFetch
	if *fastMode {
		fmt.Printf("⚡ 快速模式已启用\n")
	}
	if *skipFetch {
		fmt.Printf("📡 跳过 fetch 操作\n")
	}

	// 设置路由
	web.Router("/", &controllers.VersionController{}, "get,post:Index")
	web.Router("/checkout", &controllers.VersionController{}, "post:Checkout")
	web.Router("/refresh", &controllers.VersionController{}, "post:RefreshProject")
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

	// 检查项目权限并提供修复建议
	for _, project := range models.AppConfig.GetEnabledProjects() {
		if _, err := os.Stat(project.Path); err != nil {
			fmt.Printf("⚠️ 项目路径不存在: %s\n", project.Path)
		}
	}

	fmt.Printf("\n💡 提示: 如果遇到 Git 权限问题，可以运行以下命令修复:\n")
	for _, project := range models.AppConfig.GetEnabledProjects() {
		fmt.Printf("   git config --global --add safe.directory %s\n", project.Path)
	}

	fmt.Printf("\n🌟 服务启动中...\n\n")

	web.Run()
}
