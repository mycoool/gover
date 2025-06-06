package models

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Project 项目配置
type Project struct {
	Name        string `yaml:"name"`
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
	Enabled     bool   `yaml:"enabled"`
}

// UIConfig 界面配置
type UIConfig struct {
	Title    string `yaml:"title"`
	Theme    string `yaml:"theme"`
	Language string `yaml:"language"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	EnableAuth     bool   `yaml:"enable_auth"`
	SessionTimeout int    `yaml:"session_timeout"`
	SessionSecret  string `yaml:"session_secret"`
	RememberMeDays int    `yaml:"remember_me_days"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// Config 完整配置结构
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Auth     AuthConfig     `yaml:"auth"`
	Projects []Project      `yaml:"projects"`
	UI       UIConfig       `yaml:"ui"`
	Security SecurityConfig `yaml:"security"`
	Logging  LoggingConfig  `yaml:"logging"`
}

var AppConfig *Config

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	AppConfig = &config
	return &config, nil
}

// GetEnabledProjects 获取启用的项目列表
func (c *Config) GetEnabledProjects() []Project {
	var enabledProjects []Project
	for _, project := range c.Projects {
		if project.Enabled {
			enabledProjects = append(enabledProjects, project)
		}
	}
	return enabledProjects
}

// GetProjectByName 根据名称获取项目
func (c *Config) GetProjectByName(name string) *Project {
	for _, project := range c.Projects {
		if project.Name == name && project.Enabled {
			return &project
		}
	}
	return nil
}

// InitConfig 初始化配置，如果配置文件不存在则创建默认配置
func InitConfig() {
	config, err := LoadConfig("config.yaml")
	if err != nil {
		log.Printf("加载配置文件失败: %v", err)
		log.Println("使用默认配置...")

		// 创建默认配置
		config = &Config{
			Server: ServerConfig{
				Port: 8080,
				Host: "0.0.0.0",
			},
			Auth: AuthConfig{
				Username: "admin",
				Password: "password",
			},
			Projects: []Project{
				{
					Name:        "当前项目",
					Path:        "/www/wwwroot/gogo",
					Description: "当前版本管理工具项目",
					Enabled:     true,
				},
			},
			UI: UIConfig{
				Title:    "Git 版本管理工具",
				Theme:    "default",
				Language: "zh-CN",
			},
			Security: SecurityConfig{
				EnableAuth:     true,
				SessionTimeout: 3600,
				SessionSecret:  "gogo-version-manager-secret-key-2024",
				RememberMeDays: 7,
			},
			Logging: LoggingConfig{
				Level: "info",
				File:  "logs/app.log",
			},
		}
		AppConfig = config
	}

	log.Printf("配置加载成功，共有 %d 个项目", len(config.Projects))

	// 创建 Beego 配置文件
	createBeegoConfig()
}

// createBeegoConfig 创建 Beego 配置文件
func createBeegoConfig() {
	configDir := "conf"
	configFile := filepath.Join(configDir, "app.conf")

	// 检查 conf 目录是否存在，不存在则创建
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			log.Printf("创建 conf 目录失败: %v", err)
			return
		}
		log.Printf("已创建 conf 目录")
	}

	// 检查 app.conf 是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// 创建 Beego 配置内容
		configContent := fmt.Sprintf(`# Beego 配置文件 (自动生成)
# 本文件由 config.yaml 自动生成，请勿手动编辑

appname = gogo
httpport = %d
runmode = prod
autorender = true
copyrequestbody = true
enabledocs = false

# Session 配置
sessionon = false

# 视图配置
viewspath = views

# 静态文件配置
enablestatic = false

# 日志配置
[logs]
level = %s
`, AppConfig.Server.Port, AppConfig.Logging.Level)

		// 写入文件
		err := ioutil.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			log.Printf("创建 app.conf 文件失败: %v", err)
			return
		}

		log.Printf("已创建 Beego 配置文件: %s", configFile)
	}
}
