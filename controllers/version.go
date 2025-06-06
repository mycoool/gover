package controllers

import (
	"bytes"
	"fmt"
	"gover/models"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
)

// TagInfo 存储标签信息
type TagInfo struct {
	Name        string
	Checked     bool
	CreatedTime string
	Message     string
	CommitHash  string
}

// ProjectInfo 项目信息
type ProjectInfo struct {
	Name        string
	Path        string
	Description string
	Tags        []TagInfo
	Current     bool
}

// VersionController 版本管理控制器
type VersionController struct {
	web.Controller
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
	timeCmd := exec.Command("git", "log", "-1", "--format=%ci", tagName)
	timeCmd.Dir = projectPath
	var timeOut bytes.Buffer
	timeCmd.Stdout = &timeOut

	createdTime := "未知时间"
	if err := timeCmd.Run(); err == nil {
		if timeStr := strings.TrimSpace(timeOut.String()); timeStr != "" {
			if t, err := time.Parse("2006-01-02 15:04:05 -0700", timeStr); err == nil {
				createdTime = t.Format("2006-01-02 15:04")
			}
		}
	}

	// 获取标签消息
	msgCmd := exec.Command("git", "tag", "-l", "--format=%(contents)", tagName)
	msgCmd.Dir = projectPath
	var msgOut bytes.Buffer
	msgCmd.Stdout = &msgOut

	message := "无备注"
	if err := msgCmd.Run(); err == nil {
		if msg := strings.TrimSpace(msgOut.String()); msg != "" {
			message = msg
		}
	}

	// 获取提交hash
	hashCmd := exec.Command("git", "rev-list", "-n", "1", tagName)
	hashCmd.Dir = projectPath
	var hashOut bytes.Buffer
	hashCmd.Stdout = &hashOut

	commitHash := ""
	if err := hashCmd.Run(); err == nil {
		if hash := strings.TrimSpace(hashOut.String()); len(hash) >= 7 {
			commitHash = hash[:7]
		}
	}

	return createdTime, message, commitHash
}

// getTags 获取指定项目的所有Git标签
func (c *VersionController) getTags(projectPath string) ([]TagInfo, error) {
	cmd := exec.Command("git", "tag", "-l")
	cmd.Dir = projectPath
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	tags := strings.Split(strings.TrimSpace(out.String()), "\n")
	var tagInfos []TagInfo

	// 获取当前检出的标签
	currentTagCmd := exec.Command("git", "describe", "--tags")
	currentTagCmd.Dir = projectPath
	var currentTagOut bytes.Buffer
	currentTagCmd.Stdout = &currentTagOut
	currentTagErr := currentTagCmd.Run()
	currentTag := ""
	if currentTagErr == nil {
		currentTag = strings.TrimSpace(currentTagOut.String())
	}

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			// 获取标签详细信息
			createdTime, message, commitHash := c.getTagDetails(projectPath, tag)

			tagInfos = append(tagInfos, TagInfo{
				Name:        tag,
				Checked:     tag == currentTag,
				CreatedTime: createdTime,
				Message:     message,
				CommitHash:  commitHash,
			})
		}
	}

	// 按版本号排序（降序，最新版本在前）
	sort.Slice(tagInfos, func(i, j int) bool {
		return compareVersions(tagInfos[i].Name, tagInfos[j].Name) > 0
	})

	return tagInfos, nil
}

// checkoutTag 检出指定标签（回滚功能）
func (c *VersionController) checkoutTag(projectPath, tag string) error {
	// 先获取最新代码
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git fetch failed: %v", err)
	}

	// 检出指定标签
	cmd = exec.Command("git", "checkout", tag)
	cmd.Dir = projectPath
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout failed: %v, stderr: %s", err, stderr.String())
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
		tags, err := c.getTags(project.Path)
		if err != nil {
			// 如果获取失败，显示空标签列表但保留项目信息
			tags = []TagInfo{}
		}

		projectInfo := ProjectInfo{
			Name:        project.Name,
			Path:        project.Path,
			Description: project.Description,
			Tags:        tags,
			Current:     project.Name == selectedProject,
		}

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

// Checkout 执行版本回滚
func (c *VersionController) Checkout() {
	// 检查认证
	RequireAuth(&c.Controller)

	tag := c.GetString("tag")
	projectName := c.GetString("project")

	if tag == "" || projectName == "" {
		c.Data["Error"] = "标签和项目参数不能为空"
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

	err := c.checkoutTag(project.Path, tag)
	if err != nil {
		c.Data["Error"] = fmt.Sprintf("项目 %s 回滚到版本 %s 失败: %v", projectName, tag, err)
	} else {
		c.Data["Success"] = fmt.Sprintf("项目 %s 成功回滚到版本 %s", projectName, tag)
	}

	// 重定向回主页面，保持当前项目选中状态
	c.Redirect("/?project="+projectName, 302)
}
