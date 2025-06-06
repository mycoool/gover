package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/beego/beego/v2/server/web"
)

// 嵌入模板文件
//
//go:embed views
var embeddedTemplates embed.FS

// setupEmbeddedTemplates 设置嵌入的模板文件
func setupEmbeddedTemplates() error {
	// 检查是否存在本地 views 目录
	if _, err := os.Stat("views"); err == nil {
		// 如果本地存在 views 目录，使用本地模板（开发模式）
		fmt.Printf("📁 使用本地模板文件 (开发模式)\n")
		return nil
	}

	// 使用嵌入的模板文件
	fmt.Printf("📦 使用嵌入模板文件 (生产模式)\n")

	// 创建临时目录来提取模板文件
	tempDir := os.TempDir()
	viewsPath := filepath.Join(tempDir, "gover_views")

	// 清理旧的临时模板文件
	if err := os.RemoveAll(viewsPath); err != nil {
		fmt.Printf("⚠️ 清理旧模板失败: %v\n", err)
	}

	// 提取嵌入的模板文件到临时目录
	err := extractEmbeddedFiles(embeddedTemplates, "views", viewsPath)
	if err != nil {
		return fmt.Errorf("提取模板文件失败: %v", err)
	}

	// 设置 Beego 模板路径
	web.BConfig.WebConfig.ViewsPath = viewsPath
	fmt.Printf("📂 模板文件路径: %s\n", viewsPath)

	return nil
}

// extractEmbeddedFiles 提取嵌入的文件到指定目录
func extractEmbeddedFiles(embeddedFS embed.FS, sourceDir, targetDir string) error {
	return fs.WalkDir(embeddedFS, sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 计算目标路径
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(targetDir, relPath)

		if d.IsDir() {
			// 创建目录
			return os.MkdirAll(targetPath, 0755)
		}

		// 读取嵌入的文件内容
		content, err := embeddedFS.ReadFile(path)
		if err != nil {
			return err
		}

		// 确保目标目录存在
		targetDirPath := filepath.Dir(targetPath)
		if err := os.MkdirAll(targetDirPath, 0755); err != nil {
			return err
		}

		// 写入文件
		return os.WriteFile(targetPath, content, 0644)
	})
}
