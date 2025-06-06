package controllers

import (
	"crypto/sha256"
	"fmt"
	"gover/models"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/gorilla/sessions"
)

var (
	// Session store - 将在 init 中初始化
	store *sessions.CookieStore
)

func init() {
	// 延迟初始化 session store，等待配置加载
}

// ResetSessionStore 重置 Session Store（用于命令行清除功能）
func ResetSessionStore() {
	store = nil
}

// AuthController 认证控制器
type AuthController struct {
	web.Controller
}

// initStore 初始化 session store
func initStore() {
	if store == nil {
		store = sessions.NewCookieStore([]byte(models.AppConfig.Security.SessionSecret))
	}
}

// Login 显示登录页面或处理登录请求
func (c *AuthController) Login() {
	initStore()

	if c.Ctx.Request.Method == "GET" {
		// 检查是否已经登录
		if c.isLoggedIn() {
			c.Redirect("/", 302)
			return
		}

		// 获取重定向参数
		redirect := c.GetString("redirect", "")

		c.Data["Title"] = "登录 - " + models.AppConfig.UI.Title
		c.Data["Redirect"] = redirect
		c.TplName = "auth/login.html"
		return
	}

	// POST 请求处理登录
	username := c.GetString("username")
	password := c.GetString("password")
	remember := c.GetString("remember") == "on"

	if c.validateCredentials(username, password) {
		// 登录成功，创建 session
		session, _ := store.Get(c.Ctx.Request, "gogo-session")
		session.Values["authenticated"] = true
		session.Values["username"] = username
		session.Values["login_time"] = time.Now().Unix()
		session.Values["config_hash"] = c.getConfigHash() // 添加配置哈希

		// 设置 session 过期时间
		if remember {
			session.Options.MaxAge = models.AppConfig.Security.RememberMeDays * 24 * 3600
		} else {
			session.Options.MaxAge = models.AppConfig.Security.SessionTimeout
		}

		if err := session.Save(c.Ctx.Request, c.Ctx.ResponseWriter); err != nil {
			c.Data["Error"] = "会话保存失败"
			c.Data["Username"] = username
			c.Data["Title"] = "登录 - " + models.AppConfig.UI.Title
			c.Data["Redirect"] = c.GetString("redirect", "")
			c.TplName = "auth/login.html"
			return
		}

		// 重定向到原来要访问的页面或首页
		redirect := c.GetString("redirect", "/")
		c.Redirect(redirect, 302)
	} else {
		// 登录失败
		redirect := c.GetString("redirect", "")
		c.Data["Error"] = "用户名或密码错误"
		c.Data["Username"] = username
		c.Data["Title"] = "登录 - " + models.AppConfig.UI.Title
		c.Data["Redirect"] = redirect
		c.TplName = "auth/login.html"
	}
}

// Logout 退出登录
func (c *AuthController) Logout() {
	initStore()
	session, _ := store.Get(c.Ctx.Request, "gogo-session")
	session.Values["authenticated"] = false
	session.Options.MaxAge = -1 // 删除 session
	if err := session.Save(c.Ctx.Request, c.Ctx.ResponseWriter); err != nil {
		// 记录错误但不阻止退出流程
		fmt.Printf("退出时保存会话失败: %v\n", err)
	}

	c.Redirect("/login", 302)
}

// validateCredentials 验证用户凭据
func (c *AuthController) validateCredentials(username, password string) bool {
	// 使用配置文件中的用户名和密码
	expectedUsername := models.AppConfig.Auth.Username
	expectedPassword := models.AppConfig.Auth.Password

	// 对密码进行哈希比较（更安全）
	hashedPassword := c.hashPassword(password)
	expectedHashedPassword := c.hashPassword(expectedPassword)

	return username == expectedUsername && hashedPassword == expectedHashedPassword
}

// hashPassword 对密码进行哈希处理
func (c *AuthController) hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password + "gogo-salt"))
	return fmt.Sprintf("%x", hash)
}

// getConfigHash 获取配置哈希值，用于检测配置变更
func (c *AuthController) getConfigHash() string {
	// 将关键配置信息组合成字符串进行哈希
	configData := fmt.Sprintf("%s:%s:%s",
		models.AppConfig.Auth.Username,
		models.AppConfig.Auth.Password,
		models.AppConfig.Security.SessionSecret)
	hash := sha256.Sum256([]byte(configData))
	return fmt.Sprintf("%x", hash)
}

// invalidateSession 使当前 session 失效
func (c *AuthController) invalidateSession() {
	initStore()
	session, err := store.Get(c.Ctx.Request, "gogo-session")
	if err != nil {
		return
	}

	// 清除 session 数据
	session.Values["authenticated"] = false
	session.Options.MaxAge = -1
	if err := session.Save(c.Ctx.Request, c.Ctx.ResponseWriter); err != nil {
		// 记录错误但不阻止失效流程
		fmt.Printf("会话失效时保存失败: %v\n", err)
	}
}

// isLoggedIn 检查用户是否已登录
func (c *AuthController) isLoggedIn() bool {
	initStore()
	session, err := store.Get(c.Ctx.Request, "gogo-session")
	if err != nil {
		return false
	}

	authenticated, ok := session.Values["authenticated"].(bool)
	if !ok || !authenticated {
		return false
	}

	// 检查 session 是否过期
	loginTime, ok := session.Values["login_time"].(int64)
	if !ok {
		return false
	}

	// 如果超过配置的超时时间，认为已过期
	if time.Now().Unix()-loginTime > int64(models.AppConfig.Security.SessionTimeout) {
		return false
	}

	// 检查配置是否发生变更
	sessionConfigHash, ok := session.Values["config_hash"].(string)
	if !ok {
		// 如果没有配置哈希，说明是旧版本的 session，需要重新登录
		return false
	}

	currentConfigHash := c.getConfigHash()
	if sessionConfigHash != currentConfigHash {
		// 配置已变更，session 失效
		c.invalidateSession()
		return false
	}

	return true
}

// RequireAuth 中间件：要求用户登录
func RequireAuth(c *web.Controller) {
	authCtrl := &AuthController{Controller: *c}
	if !authCtrl.isLoggedIn() {
		// 保存当前请求的 URL，登录后重定向
		currentURL := c.Ctx.Request.URL.String()
		c.Redirect("/login?redirect="+currentURL, 302)
		return
	}
}
