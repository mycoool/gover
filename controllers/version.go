package controllers

import (
	"bytes"
	"fmt"
	"gover/models"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/server/web"
)

// 全局调试模式
var DebugMode bool

// 缓存相关
var (
	projectCache = make(map[string]*ProjectCacheItem)
	cacheMutex   sync.RWMutex
	cacheExpiry  = 5 * time.Minute // 缓存过期时间
)

// ProjectCacheItem 项目缓存项
type ProjectCacheItem struct {
	ProjectInfo ProjectInfo
	UpdateTime  time.Time
	Updating    bool // 是否正在更新
}

// 性能配置
var (
	SkipFetch     = false // 是否跳过 fetch 操作
	FastMode      = false // 快速模式：只获取基本信息
	MaxConcurrent = 3     // 最大并发数
)

// TagInfo 存储标签信息
type TagInfo struct {
	Name        string
	Checked     bool
	CreatedTime string
	Message     string
	CommitHash  string
	IsRemote    bool // 是否为远程标签
}

// BranchInfo 存储分支信息
type BranchInfo struct {
	Name       string
	Checked    bool
	IsRemote   bool
	LastCommit string
	CommitHash string
	CommitTime string
}

// ProjectInfo 项目信息
type ProjectInfo struct {
	Name          string
	Path          string
	Description   string
	Tags          []TagInfo
	Branches      []BranchInfo
	Current       bool
	CurrentBranch string // 当前分支名
	CurrentTag    string // 当前标签名
	WorkingMode   string // "branch" 或 "tag"
}

// VersionController 版本控制器
type VersionController struct {
	web.Controller
}

// isProjectCacheValid 检查项目缓存是否有效
func isProjectCacheValid(projectPath string) bool {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	cache, exists := projectCache[projectPath]
	if !exists {
		return false
	}

	return time.Since(cache.UpdateTime) < cacheExpiry
}

// getProjectFromCache 从缓存获取项目信息
func getProjectFromCache(projectPath string) (ProjectInfo, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	cache, exists := projectCache[projectPath]
	if !exists || time.Since(cache.UpdateTime) >= cacheExpiry {
		return ProjectInfo{}, false
	}

	return cache.ProjectInfo, true
}

// setProjectCache 设置项目缓存
func setProjectCache(projectPath string, projectInfo ProjectInfo) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	projectCache[projectPath] = &ProjectCacheItem{
		ProjectInfo: projectInfo,
		UpdateTime:  time.Now(),
		Updating:    false,
	}
}

// markProjectUpdating 标记项目正在更新
func markProjectUpdating(projectPath string, updating bool) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if cache, exists := projectCache[projectPath]; exists {
		cache.Updating = updating
	} else if updating {
		projectCache[projectPath] = &ProjectCacheItem{
			Updating:   true,
			UpdateTime: time.Now(),
		}
	}
}

// isProjectUpdating 检查项目是否正在更新
func isProjectUpdating(projectPath string) bool {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	cache, exists := projectCache[projectPath]
	return exists && cache.Updating
}

// updateProjectAsync 异步更新项目信息
func (c *VersionController) updateProjectAsync(project models.Project) {
	go func() {
		if isProjectUpdating(project.Path) {
			return // 已经在更新中
		}

		markProjectUpdating(project.Path, true)
		defer markProjectUpdating(project.Path, false)

		if DebugMode {
			fmt.Printf("🔄 异步更新项目: %s\n", project.Name)
		}

		// 获取完整的项目信息
		projectInfo := c.buildProjectInfo(project, false) // false = 完整模式
		setProjectCache(project.Path, projectInfo)

		if DebugMode {
			fmt.Printf("✅ 异步更新完成: %s\n", project.Name)
		}
	}()
}

// parseVersion 解析版本号为数字数组，用于排序
func parseVersion(version string) []int {
	// 移除 v 前缀
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")

	// 使用正则表达式提取数字部分
	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindAllString(version, -1)

	var parts []int
	for _, match := range matches {
		if num, err := strconv.Atoi(match); err == nil {
			parts = append(parts, num)
		}
	}

	// 确保至少有3个部分，不足的用0补充
	for len(parts) < 3 {
		parts = append(parts, 0)
	}

	return parts
}

// compareVersions 比较两个版本号，返回 -1, 0, 1
func compareVersions(v1, v2 string) int {
	parts1 := parseVersion(v1)
	parts2 := parseVersion(v2)

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	// 补齐较短的版本号
	for len(parts1) < maxLen {
		parts1 = append(parts1, 0)
	}
	for len(parts2) < maxLen {
		parts2 = append(parts2, 0)
	}

	for i := 0; i < maxLen; i++ {
		if parts1[i] < parts2[i] {
			return -1
		} else if parts1[i] > parts2[i] {
			return 1
		}
	}

	return 0
}

// getTagDetails 获取标签的详细信息
func (c *VersionController) getTagDetails(projectPath, tagName string) (string, string, string) {
	// 获取标签创建时间
	createdTime := "未知时间"
	if timeStr, err := c.executeGitCommand(projectPath, "log", "-1", "--format=%ci", tagName); err == nil {
		if timeStr != "" {
			if t, err := time.Parse("2006-01-02 15:04:05 -0700", timeStr); err == nil {
				createdTime = t.Format("2006-01-02 15:04")
			}
		}
	} else if DebugMode {
		fmt.Printf("⚠️ 获取标签 %s 时间失败: %v\n", tagName, err)
	}

	// 获取标签消息
	message := "无备注"
	if msg, err := c.executeGitCommand(projectPath, "tag", "-l", "--format=%(contents)", tagName); err == nil {
		if msg != "" {
			message = msg
		}
	} else if DebugMode {
		fmt.Printf("⚠️ 获取标签 %s 消息失败: %v\n", tagName, err)
	}

	// 获取提交hash
	commitHash := ""
	if hash, err := c.executeGitCommand(projectPath, "rev-list", "-n", "1", tagName); err == nil {
		if len(hash) >= 7 {
			commitHash = hash[:7]
		}
	} else if DebugMode {
		fmt.Printf("⚠️ 获取标签 %s 哈希失败: %v\n", tagName, err)
	}

	return createdTime, message, commitHash
}

// getCurrentWorkingMode 获取当前工作模式和状态
func (c *VersionController) getCurrentWorkingMode(projectPath string) (string, string, string) {
	if DebugMode {
		fmt.Printf("🔍 获取项目 %s 的当前工作模式...\n", projectPath)
	}

	// 检查是否在分支上
	if branchName, err := c.executeGitCommand(projectPath, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		branchName = strings.TrimSpace(branchName)
		if branchName != "HEAD" && branchName != "" {
			if DebugMode {
				fmt.Printf("✅ 当前在分支: %s\n", branchName)
			}
			return "branch", branchName, ""
		}
	}

	// 检查是否在标签上
	if tagName, err := c.executeGitCommand(projectPath, "describe", "--exact-match", "--tags"); err == nil {
		tagName = strings.TrimSpace(tagName)
		if tagName != "" {
			if DebugMode {
				fmt.Printf("✅ 当前在标签: %s\n", tagName)
			}
			return "tag", "", tagName
		}
	}

	// 尝试获取最近的标签
	if tagName, err := c.executeGitCommand(projectPath, "describe", "--tags"); err == nil {
		tagName = strings.TrimSpace(tagName)
		if tagName != "" {
			if DebugMode {
				fmt.Printf("⚠️ 当前在游离状态，最近标签: %s\n", tagName)
			}
			return "detached", "", tagName
		}
	}

	if DebugMode {
		fmt.Printf("⚠️ 无法确定当前工作模式\n")
	}
	return "unknown", "", ""
}

// getBranches 获取所有分支信息
func (c *VersionController) getBranches(projectPath string) ([]BranchInfo, error) {
	if DebugMode {
		fmt.Printf("🔍 正在获取项目 %s 的分支信息...\n", projectPath)
	}

	// 先执行 fetch 获取最新的远程分支信息
	if _, err := c.executeGitCommand(projectPath, "fetch", "--all"); err != nil {
		if DebugMode {
			fmt.Printf("⚠️ fetch 远程分支失败: %v\n", err)
		}
	}

	// 获取所有分支（本地和远程）
	branchOutput, err := c.executeGitCommand(projectPath, "branch", "-a", "-v")
	if err != nil {
		return nil, fmt.Errorf("获取分支列表失败: %v", err)
	}

	// 获取当前分支
	currentBranch := ""
	if branch, err := c.executeGitCommand(projectPath, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		currentBranch = branch
	}

	var branches []BranchInfo
	lines := strings.Split(branchOutput, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析分支信息
		var branchInfo BranchInfo

		// 检查是否为当前分支
		if strings.HasPrefix(line, "* ") {
			branchInfo.Checked = true
			line = strings.TrimPrefix(line, "* ")
		} else if strings.HasPrefix(line, "  ") {
			line = strings.TrimPrefix(line, "  ")
		}

		// 分割分支名和提交信息
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		branchName := parts[0]
		commitHash := parts[1]

		// 检查是否为远程分支
		if strings.HasPrefix(branchName, "remotes/") {
			branchInfo.IsRemote = true
			// 去掉 remotes/ 前缀但保留 origin/ 等
			branchName = strings.TrimPrefix(branchName, "remotes/")
		}

		// 跳过 HEAD 指针
		if strings.Contains(branchName, "HEAD ->") {
			continue
		}

		branchInfo.Name = branchName
		branchInfo.CommitHash = commitHash

		// 获取最后一次提交的时间和信息
		if commitTime, err := c.executeGitCommand(projectPath, "log", "-1", "--format=%ci", commitHash); err == nil {
			if t, err := time.Parse("2006-01-02 15:04:05 -0700", commitTime); err == nil {
				branchInfo.CommitTime = t.Format("2006-01-02 15:04")
			}
		}

		if commitMsg, err := c.executeGitCommand(projectPath, "log", "-1", "--format=%s", commitHash); err == nil {
			if len(commitMsg) > 50 {
				commitMsg = commitMsg[:50] + "..."
			}
			branchInfo.LastCommit = commitMsg
		}

		// 设置当前分支标记
		if branchName == currentBranch || (branchInfo.IsRemote && strings.HasSuffix(branchName, "/"+currentBranch)) {
			branchInfo.Checked = true
		}

		branches = append(branches, branchInfo)
	}

	if DebugMode {
		fmt.Printf("✅ 获取到 %d 个分支\n", len(branches))
	}

	return branches, nil
}

// buildProjectInfo 构建项目信息（支持快速模式和完整模式）
func (c *VersionController) buildProjectInfo(project models.Project, fastMode bool) ProjectInfo {
	projectInfo := ProjectInfo{
		Name:        project.Name,
		Path:        project.Path,
		Description: project.Description,
		Current:     false, // 稍后在调用处设置
	}

	// 获取当前工作模式和状态
	workingMode, currentBranch, currentTag := c.getCurrentWorkingMode(project.Path)
	projectInfo.WorkingMode = workingMode
	projectInfo.CurrentBranch = currentBranch
	projectInfo.CurrentTag = currentTag

	if fastMode {
		// 快速模式：只获取基本信息，不获取详细标签和分支信息
		projectInfo.Description = fmt.Sprintf("Git 项目 (%s)", workingMode)
		if workingMode == "branch" {
			projectInfo.Description += fmt.Sprintf("，当前分支: %s", currentBranch)
		} else if workingMode == "tag" {
			projectInfo.Description += fmt.Sprintf("，当前标签: %s", currentTag)
		}
		return projectInfo
	}

	// 完整模式：获取详细信息
	tags, _ := c.getTagsFast(project.Path)
	branches, _ := c.getBranchesFast(project.Path)

	projectInfo.Tags = tags
	projectInfo.Branches = branches

	// 更新描述
	description := project.Description
	if len(tags) > 0 || len(branches) > 0 {
		description = fmt.Sprintf("Git 项目，%d 个标签，%d 个分支", len(tags), len(branches))
		if workingMode == "branch" {
			description += fmt.Sprintf("，当前分支: %s", currentBranch)
		} else if workingMode == "tag" {
			description += fmt.Sprintf("，当前标签: %s", currentTag)
		}
	}
	projectInfo.Description = description

	return projectInfo
}

// getTagsFast 快速获取标签信息（减少 fetch 调用）
func (c *VersionController) getTagsFast(projectPath string) ([]TagInfo, error) {
	if DebugMode {
		fmt.Printf("🔍 正在快速获取项目 %s 的标签...\n", projectPath)
	}

	// 在快速模式下，只在必要时才 fetch
	if !SkipFetch {
		// 使用超时的 fetch，避免长时间等待
		done := make(chan bool, 1)
		go func() {
			c.executeGitCommand(projectPath, "fetch", "--tags")
			done <- true
		}()

		select {
		case <-done:
			// fetch 完成
		case <-time.After(3 * time.Second):
			// 超时，继续使用本地标签
			if DebugMode {
				fmt.Printf("⚠️ fetch 超时，使用本地标签\n")
			}
		}
	}

	// 获取所有标签（本地优先）
	tagOutput, err := c.executeGitCommand(projectPath, "tag", "-l", "--sort=-version:refname")
	if err != nil {
		return nil, fmt.Errorf("获取标签列表失败: %v", err)
	}

	tags := strings.Split(tagOutput, "\n")
	var tagInfos []TagInfo

	// 获取当前状态以设置选中标签
	workingMode, _, currentTag := c.getCurrentWorkingMode(projectPath)

	// 限制处理的标签数量以提高性能
	maxTags := 20
	count := 0

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			count++
			if count > maxTags {
				// 只处理前20个标签以提高性能
				break
			}

			// 在快速模式下简化标签详细信息获取
			var createdTime, message, commitHash string
			if FastMode {
				// 快速模式：只获取基本信息
				createdTime = "N/A"
				message = "使用快速模式"
				commitHash = "N/A"
			} else {
				// 完整模式：获取详细信息
				createdTime, message, commitHash = c.getTagDetails(projectPath, tag)
			}

			// 确保标签选中状态的正确性
			isChecked := (workingMode == "tag" || workingMode == "detached") && tag == currentTag

			tagInfos = append(tagInfos, TagInfo{
				Name:        tag,
				Checked:     isChecked,
				CreatedTime: createdTime,
				Message:     message,
				CommitHash:  commitHash,
				IsRemote:    false,
			})
		}
	}

	if DebugMode {
		fmt.Printf("✅ 获取到 %d 个标签\n", len(tagInfos))
	}

	return tagInfos, nil
}

// getBranchesFast 快速获取分支信息（减少 fetch 调用）
func (c *VersionController) getBranchesFast(projectPath string) ([]BranchInfo, error) {
	if DebugMode {
		fmt.Printf("🔍 正在快速获取项目 %s 的分支信息...\n", projectPath)
	}

	// 在快速模式下，只在必要时才 fetch
	if !SkipFetch {
		// 使用超时的 fetch，避免长时间等待
		done := make(chan bool, 1)
		go func() {
			c.executeGitCommand(projectPath, "fetch", "--all")
			done <- true
		}()

		select {
		case <-done:
			// fetch 完成
		case <-time.After(3 * time.Second):
			// 超时，继续使用本地分支
			if DebugMode {
				fmt.Printf("⚠️ fetch 分支超时，使用本地分支\n")
			}
		}
	}

	// 获取分支信息（本地和远程）
	branchOutput, err := c.executeGitCommand(projectPath, "branch", "-a", "-v")
	if err != nil {
		return nil, fmt.Errorf("获取分支列表失败: %v", err)
	}

	// 获取当前分支和工作模式
	workingMode, currentBranch, _ := c.getCurrentWorkingMode(projectPath)

	var branches []BranchInfo
	lines := strings.Split(branchOutput, "\n")

	// 限制处理的分支数量以提高性能
	maxBranches := 15
	count := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		count++
		if count > maxBranches {
			break
		}

		// 解析分支信息
		var branchInfo BranchInfo

		// 检查是否为当前分支
		if strings.HasPrefix(line, "* ") {
			branchInfo.Checked = true
			line = strings.TrimPrefix(line, "* ")
		} else if strings.HasPrefix(line, "  ") {
			line = strings.TrimPrefix(line, "  ")
		}

		// 分割分支名和提交信息
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		branchName := parts[0]
		commitHash := parts[1]

		// 检查是否为远程分支
		if strings.HasPrefix(branchName, "remotes/") {
			branchInfo.IsRemote = true
			branchName = strings.TrimPrefix(branchName, "remotes/")
		}

		// 跳过 HEAD 指针
		if strings.Contains(branchName, "HEAD ->") {
			continue
		}

		branchInfo.Name = branchName
		branchInfo.CommitHash = commitHash

		// 在快速模式下简化提交信息获取
		if FastMode {
			branchInfo.CommitTime = "N/A"
			branchInfo.LastCommit = "使用快速模式"
		} else {
			// 获取最后一次提交的时间和信息
			if commitTime, err := c.executeGitCommand(projectPath, "log", "-1", "--format=%ci", commitHash); err == nil {
				if t, err := time.Parse("2006-01-02 15:04:05 -0700", commitTime); err == nil {
					branchInfo.CommitTime = t.Format("2006-01-02 15:04")
				}
			}

			if commitMsg, err := c.executeGitCommand(projectPath, "log", "-1", "--format=%s", commitHash); err == nil {
				if len(commitMsg) > 50 {
					commitMsg = commitMsg[:50] + "..."
				}
				branchInfo.LastCommit = commitMsg
			}
		}

		// 设置当前分支标记 - 只有在分支模式下才标记分支为选中
		if workingMode == "branch" && (branchName == currentBranch || (branchInfo.IsRemote && strings.HasSuffix(branchName, "/"+currentBranch))) {
			branchInfo.Checked = true
		}

		branches = append(branches, branchInfo)
	}

	if DebugMode {
		fmt.Printf("✅ 获取到 %d 个分支\n", len(branches))
	}

	return branches, nil
}

// debugProjectInfo 输出项目的调试信息
func (c *VersionController) debugProjectInfo(project models.Project) {
	fmt.Printf("\n🔧 项目诊断信息:\n")
	fmt.Printf("   名称: %s\n", project.Name)
	fmt.Printf("   路径: %s\n", project.Path)
	fmt.Printf("   描述: %s\n", project.Description)
	fmt.Printf("   启用: %v\n", project.Enabled)

	// 检查路径是否存在
	if stat, err := os.Stat(project.Path); err == nil {
		fmt.Printf("   路径状态: ✅ 存在 (%s)\n", func() string {
			if stat.IsDir() {
				return "目录"
			}
			return "文件"
		}())
	} else {
		fmt.Printf("   路径状态: ❌ 不存在 (%v)\n", err)
		return
	}

	// 检查 .git 目录
	gitDir := filepath.Join(project.Path, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		fmt.Printf("   Git 仓库: ✅ 有效\n")
	} else {
		fmt.Printf("   Git 仓库: ❌ 无效 (.git 目录不存在)\n")
		return
	}

	// 测试 git 命令
	if _, err := c.executeGitCommand(project.Path, "status", "--porcelain"); err == nil {
		fmt.Printf("   Git 命令: ✅ 正常\n")
	} else {
		fmt.Printf("   Git 命令: ❌ 失败 (%v)\n", err)
	}

	// 快速获取标签数量
	if tagOutput, err := c.executeGitCommand(project.Path, "tag", "-l"); err == nil {
		tagCount := len(strings.Fields(tagOutput))
		fmt.Printf("   标签数量: %d\n", tagCount)
	} else {
		fmt.Printf("   标签获取: ❌ 失败 (%v)\n", err)
	}
	fmt.Printf("\n")
}

// fixGitOwnership 修复 Git 仓库权限问题
func (c *VersionController) fixGitOwnership(projectPath string) error {
	if DebugMode {
		fmt.Printf("🔧 尝试修复 Git 权限: %s\n", projectPath)
	}

	// 方法1: 尝试添加全局安全目录配置
	if err := c.tryGlobalSafeDirectory(projectPath); err == nil {
		if DebugMode {
			fmt.Printf("✅ 全局配置成功\n")
		}
		return nil
	}

	// 方法2: 尝试添加系统级安全目录配置
	if err := c.trySystemSafeDirectory(projectPath); err == nil {
		if DebugMode {
			fmt.Printf("✅ 系统配置成功\n")
		}
		return nil
	}

	// 方法3: 尝试本地仓库配置
	if err := c.tryLocalSafeDirectory(projectPath); err == nil {
		if DebugMode {
			fmt.Printf("✅ 本地配置成功\n")
		}
		return nil
	}

	// 方法4: 设置 HOME 环境变量后重试
	if err := c.tryWithHomeSet(projectPath); err == nil {
		if DebugMode {
			fmt.Printf("✅ 设置 HOME 后成功\n")
		}
		return nil
	}

	return fmt.Errorf("所有权限修复方法都失败了")
}

// tryGlobalSafeDirectory 尝试添加全局安全目录
func (c *VersionController) tryGlobalSafeDirectory(projectPath string) error {
	cmd := exec.Command("git", "config", "--global", "--add", "safe.directory", projectPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if DebugMode {
			fmt.Printf("⚠️ 全局配置失败: %v, 输出: %s\n", err, stderr.String())
		}
		return err
	}
	return nil
}

// trySystemSafeDirectory 尝试添加系统级安全目录
func (c *VersionController) trySystemSafeDirectory(projectPath string) error {
	cmd := exec.Command("git", "config", "--system", "--add", "safe.directory", projectPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if DebugMode {
			fmt.Printf("⚠️ 系统配置失败: %v, 输出: %s\n", err, stderr.String())
		}
		return err
	}
	return nil
}

// tryLocalSafeDirectory 尝试在本地仓库配置
func (c *VersionController) tryLocalSafeDirectory(projectPath string) error {
	cmd := exec.Command("git", "config", "--add", "safe.directory", projectPath)
	cmd.Dir = projectPath
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if DebugMode {
			fmt.Printf("⚠️ 本地配置失败: %v, 输出: %s\n", err, stderr.String())
		}
		return err
	}
	return nil
}

// tryWithHomeSet 设置 HOME 环境变量后重试
func (c *VersionController) tryWithHomeSet(projectPath string) error {
	// 尝试设置一个临时的 HOME 目录
	tmpHome := "/tmp"
	if _, err := os.Stat("/tmp"); os.IsNotExist(err) {
		tmpHome = "."
	}

	cmd := exec.Command("git", "config", "--global", "--add", "safe.directory", projectPath)
	cmd.Env = append(os.Environ(), "HOME="+tmpHome)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if DebugMode {
			fmt.Printf("⚠️ 设置 HOME 后配置失败: %v, 输出: %s\n", err, stderr.String())
		}
		return err
	}
	return nil
}

// executeGitCommand 执行 Git 命令的通用方法，自动处理权限问题
func (c *VersionController) executeGitCommand(projectPath string, args ...string) (string, error) {
	// 方法1: 直接尝试执行命令
	output, err := c.tryGitCommand(projectPath, args...)
	if err == nil {
		return output, nil
	}

	// 检查是否是权限问题
	if !strings.Contains(err.Error(), "dubious ownership") {
		return "", err
	}

	if DebugMode {
		fmt.Printf("🔧 检测到权限问题，尝试修复...\n")
	}

	// 方法2: 使用环境变量绕过权限检查
	output, err = c.tryGitCommandWithEnvBypass(projectPath, args...)
	if err == nil {
		if DebugMode {
			fmt.Printf("✅ 环境变量绕过成功\n")
		}
		return output, nil
	}

	// 方法3: 尝试修复权限后重试
	if fixErr := c.fixGitOwnership(projectPath); fixErr != nil {
		return "", fmt.Errorf("权限修复失败: %v", fixErr)
	}

	// 重试命令
	output, err = c.tryGitCommand(projectPath, args...)
	if err != nil {
		return "", fmt.Errorf("修复权限后仍然失败: %v", err)
	}

	return output, nil
}

// tryGitCommand 尝试执行 Git 命令
func (c *VersionController) tryGitCommand(projectPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = projectPath

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("命令执行失败: %v, 输出: %s", err, stderr.String())
	}

	return strings.TrimSpace(out.String()), nil
}

// tryGitCommandWithEnvBypass 使用环境变量绕过权限检查
func (c *VersionController) tryGitCommandWithEnvBypass(projectPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = projectPath

	// 设置环境变量绕过权限检查
	env := os.Environ()

	// 方案1: 通过环境变量设置安全目录 (Git 2.35.2+)
	env = append(env, "GIT_CONFIG_COUNT=1")
	env = append(env, "GIT_CONFIG_KEY_0=safe.directory")
	env = append(env, "GIT_CONFIG_VALUE_0=*")

	// 方案2: 忽略配置文件
	env = append(env, "GIT_CONFIG_GLOBAL=/dev/null") // 忽略全局配置
	env = append(env, "GIT_CONFIG_SYSTEM=/dev/null") // 忽略系统配置

	// 方案3: 确保 HOME 环境变量存在
	hasHome := false
	for _, e := range env {
		if strings.HasPrefix(e, "HOME=") {
			hasHome = true
			break
		}
	}
	if !hasHome {
		env = append(env, "HOME=/tmp")
	}

	cmd.Env = env

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if DebugMode {
			fmt.Printf("⚠️ 环境变量绕过失败: %v, 输出: %s\n", err, stderr.String())
		}
		return "", fmt.Errorf("环境变量绕过失败: %v, 输出: %s", err, stderr.String())
	}

	return strings.TrimSpace(out.String()), nil
}

// getTags 获取指定项目的所有Git标签（包括远程标签）
func (c *VersionController) getTags(projectPath string) ([]TagInfo, error) {
	// 首先检查目录是否存在
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径 %s 不存在", projectPath)
	}

	// 检查是否是 Git 仓库
	gitDir := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("目录 %s 不是一个 Git 仓库（缺少 .git 目录）", projectPath)
	}

	if DebugMode {
		fmt.Printf("🔍 正在获取项目 %s 的 Git 标签（包括远程）...\n", projectPath)
	}

	// 使用快速方法获取标签
	tags, err := c.getTagsFast(projectPath)
	if err != nil {
		if DebugMode {
			fmt.Printf("⚠️ 获取标签失败: %v\n", err)
		}
		return []TagInfo{}, err
	}

	// 按版本号排序（降序，最新版本在前）
	sort.Slice(tags, func(i, j int) bool {
		return compareVersions(tags[i].Name, tags[j].Name) > 0
	})

	// 获取当前状态用于调试
	workingMode, currentBranch, currentTag := c.getCurrentWorkingMode(projectPath)
	if DebugMode {
		fmt.Printf("✅ 成功获取项目 %s 的标签信息，当前模式: %s", projectPath, workingMode)
		if workingMode == "branch" {
			fmt.Printf("，当前分支: %s", currentBranch)
		} else if workingMode == "tag" {
			fmt.Printf("，当前标签: %s", currentTag)
		}
		fmt.Printf("\n")
	}

	return tags, nil
}

// checkoutTag 检出指定标签（回滚功能）
func (c *VersionController) checkoutTag(projectPath, tag string) error {
	// 先获取最新代码和标签
	if _, err := c.executeGitCommand(projectPath, "fetch", "--tags"); err != nil {
		return fmt.Errorf("git fetch tags failed: %v", err)
	}

	// 检出指定标签
	if _, err := c.executeGitCommand(projectPath, "checkout", tag); err != nil {
		return fmt.Errorf("git checkout failed: %v", err)
	}

	return nil
}

// checkoutBranch 检出指定分支
func (c *VersionController) checkoutBranch(projectPath, branch string) error {
	// 先获取最新的远程分支信息
	if _, err := c.executeGitCommand(projectPath, "fetch", "--all"); err != nil {
		return fmt.Errorf("git fetch failed: %v", err)
	}

	// 处理远程分支名称
	localBranch := branch
	if strings.HasPrefix(branch, "origin/") {
		localBranch = strings.TrimPrefix(branch, "origin/")
	}

	// 检查本地分支是否存在
	if _, err := c.executeGitCommand(projectPath, "show-ref", "--verify", "--quiet", "refs/heads/"+localBranch); err != nil {
		// 本地分支不存在，创建并跟踪远程分支
		if _, err := c.executeGitCommand(projectPath, "checkout", "-b", localBranch, "origin/"+localBranch); err != nil {
			return fmt.Errorf("创建并检出分支 %s 失败: %v", localBranch, err)
		}
	} else {
		// 本地分支存在，直接切换
		if _, err := c.executeGitCommand(projectPath, "checkout", localBranch); err != nil {
			return fmt.Errorf("切换到分支 %s 失败: %v", localBranch, err)
		}

		// 更新本地分支到最新
		if _, err := c.executeGitCommand(projectPath, "pull", "origin", localBranch); err != nil {
			// pull 失败不是致命错误，只是记录警告
			fmt.Printf("⚠️ 更新分支 %s 失败: %v\n", localBranch, err)
		}
	}

	return nil
}

// Index 显示项目列表和版本管理页面
func (c *VersionController) Index() {
	// 检查认证
	RequireAuth(&c.Controller)

	// 获取当前选中的项目
	selectedProject := c.GetString("project", "")

	// 获取所有启用的项目
	enabledProjects := models.AppConfig.GetEnabledProjects()

	var projectInfos []ProjectInfo
	var currentProjectInfo *ProjectInfo

	for _, project := range enabledProjects {
		// 添加更详细的诊断信息（仅在调试模式下）
		if DebugMode {
			c.debugProjectInfo(project)
		}

		var projectInfo ProjectInfo

		// 检查缓存
		if cachedInfo, found := getProjectFromCache(project.Path); found {
			projectInfo = cachedInfo
			if DebugMode {
				fmt.Printf("📋 项目 %s 使用缓存数据\n", project.Name)
			}
		} else {
			// 缓存未命中，使用快速模式获取基本信息
			projectInfo = c.buildProjectInfo(project, true) // true = 快速模式

			// 异步更新完整信息
			c.updateProjectAsync(project)

			if DebugMode {
				fmt.Printf("📋 项目 %s 使用快速模式，已启动异步更新\n", project.Name)
			}
		}

		// 设置当前项目标记
		projectInfo.Current = project.Name == selectedProject
		projectInfos = append(projectInfos, projectInfo)

		// 如果是选中的项目或者是第一个项目（默认选中）
		if project.Name == selectedProject || (selectedProject == "" && currentProjectInfo == nil) {
			currentProjectInfo = &projectInfo
		}
	}

	c.Data["Projects"] = projectInfos
	c.Data["CurrentProject"] = currentProjectInfo
	c.Data["Title"] = models.AppConfig.UI.Title
	c.TplName = "version/index.html"
}

// Checkout 执行版本回滚或分支切换
func (c *VersionController) Checkout() {
	// 检查认证
	RequireAuth(&c.Controller)

	tag := c.GetString("tag")
	branch := c.GetString("branch")
	projectName := c.GetString("project")

	// 检查参数
	if (tag == "" && branch == "") || projectName == "" {
		c.Data["Error"] = "标签/分支和项目参数不能为空"
		c.Redirect("/", 302)
		return
	}

	if tag != "" && branch != "" {
		c.Data["Error"] = "不能同时指定标签和分支"
		c.Redirect("/", 302)
		return
	}

	// 获取项目信息
	project := models.AppConfig.GetProjectByName(projectName)
	if project == nil {
		c.Data["Error"] = fmt.Sprintf("项目 %s 不存在或未启用", projectName)
		c.Redirect("/", 302)
		return
	}

	var err error
	var successMsg string

	if tag != "" {
		// 标签切换
		err = c.checkoutTag(project.Path, tag)
		if err != nil {
			c.Data["Error"] = fmt.Sprintf("项目 %s 切换到标签 %s 失败: %v", projectName, tag, err)
		} else {
			successMsg = fmt.Sprintf("项目 %s 成功切换到标签 %s", projectName, tag)
		}
	} else if branch != "" {
		// 分支切换
		err = c.checkoutBranch(project.Path, branch)
		if err != nil {
			c.Data["Error"] = fmt.Sprintf("项目 %s 切换到分支 %s 失败: %v", projectName, branch, err)
		} else {
			successMsg = fmt.Sprintf("项目 %s 成功切换到分支 %s", projectName, branch)
		}
	}

	if successMsg != "" {
		c.Data["Success"] = successMsg

		// 切换成功后立即清除缓存并更新项目信息
		cacheMutex.Lock()
		delete(projectCache, project.Path)
		cacheMutex.Unlock()

		// 立即获取最新的项目状态并缓存
		updatedInfo := c.buildProjectInfo(*project, false) // false = 完整模式
		setProjectCache(project.Path, updatedInfo)

		if DebugMode {
			fmt.Printf("✅ 切换后已更新项目 %s 的缓存信息\n", projectName)
		}
	}

	// 重定向回主页面，保持当前项目选中状态
	c.Redirect("/?project="+projectName, 302)
}

// RefreshProject 刷新项目缓存
func (c *VersionController) RefreshProject() {
	// 检查认证
	RequireAuth(&c.Controller)

	projectName := c.GetString("project")
	if projectName == "" {
		c.Data["json"] = map[string]interface{}{
			"success": false,
			"message": "项目参数不能为空",
		}
		c.ServeJSON()
		return
	}

	// 获取项目信息
	project := models.AppConfig.GetProjectByName(projectName)
	if project == nil {
		c.Data["json"] = map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("项目 %s 不存在或未启用", projectName),
		}
		c.ServeJSON()
		return
	}

	// 清除缓存
	cacheMutex.Lock()
	delete(projectCache, project.Path)
	cacheMutex.Unlock()

	// 重新获取项目信息
	projectInfo := c.buildProjectInfo(*project, false) // false = 完整模式
	setProjectCache(project.Path, projectInfo)

	c.Data["json"] = map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("项目 %s 缓存已刷新", projectName),
		"data": map[string]interface{}{
			"tags":     len(projectInfo.Tags),
			"branches": len(projectInfo.Branches),
			"mode":     projectInfo.WorkingMode,
		},
	}
	c.ServeJSON()
}
