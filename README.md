# Gover - Git 版本管理工具

Gover 是一个基于 Beego 框架开发的 Git 版本管理工具，可以方便地查看和回滚到不同的 Git 标签版本。

## 功能特性

- 📋 **智能版本列表**: 显示所有可用的 Git 标签版本，按语义化版本号排序
- 🔍 **当前版本**: 突出显示当前活跃的版本
- 🔄 **版本回滚**: 一键回滚到指定版本
- 📁 **多项目支持**: 通过 YAML 配置管理多个项目
- 🎨 **现代界面**: 响应式设计，支持移动设备
- 🔐 **安全认证**: Session 认证，支持页面登录和记住我功能
- ⚙️ **配置灵活**: YAML 配置文件，支持自定义各种参数
- 📅 **版本详情**: 显示版本创建时间、标签备注和提交哈希
- 🔄 **智能排序**: 版本号按语义化规则排序（v0.0.1 < v0.0.12）

## 技术栈

- **后端**: Go + Beego v2 框架
- **前端**: HTML + CSS (响应式设计)
- **版本控制**: Git

## 安装和运行

### 方式1: 下载预编译版本（推荐）

从 [GitHub Releases](https://github.com/mycoool/gover/releases) 页面下载对应平台的最新版本：

- **Linux AMD64**: `gover-{version}-linux-amd64.tar.gz`
- **Linux ARM64**: `gover-{version}-linux-arm64.tar.gz`
- **macOS Intel**: `gover-{version}-darwin-amd64.tar.gz`
- **macOS Apple Silicon**: `gover-{version}-darwin-arm64.tar.gz`
- **Windows AMD64**: `gover-{version}-windows-amd64.exe.zip`
- **Windows ARM64**: `gover-{version}-windows-arm64.exe.zip`

**安装步骤:**
1. 下载对应平台的压缩包
2. 解压到目标目录
3. 根据需要修改 `config.yaml` 配置
4. 运行 `./gover` (Linux/macOS) 或 `gover.exe` (Windows)

### 方式2: 从源码构建

#### 前提条件

- Go 1.21 或更高版本
- Git 已安装并配置
- 当前目录是一个 Git 仓库

#### 快速启动

1. 构建并运行：
   ```bash
   ./start.sh
   ```

2. 或者手动运行：
   ```bash
   go build -o gover .
   ./gover
   ```

3. 多平台构建：
   ```bash
   chmod +x build.sh
   ./build.sh
   ```

4. 访问 http://localhost:8080

> **注意**: 启动时可能会看到 Beego 框架的配置文件警告信息，这是正常的时序问题。系统会自动创建必要的配置文件。详见 [NOTICE.md](NOTICE.md)

### 命令行选项

```bash
# 查看帮助
./gover --help

# 清除所有 Session（强制所有用户重新登录）
./gover -clear-sessions
```

### 管理脚本

为了方便管理，项目提供了 `manage.sh` 脚本：

```bash
# 查看帮助
./manage.sh

# 启动服务
./manage.sh start

# 停止服务
./manage.sh stop

# 重启服务
./manage.sh restart

# 查看服务状态
./manage.sh status

# 构建项目
./manage.sh build

# 清除所有 Session
./manage.sh clear-sessions
```

### 认证信息

- **登录方式**: 页面登录（不再是弹窗认证）
- **用户名**: admin (可在 config.yaml 中修改)
- **密码**: password (可在 config.yaml 中修改)
- **记住我**: 支持7天免登录（可配置）
- **安全特性**: Session 认证 + 密码哈希加密

## 配置

### YAML 配置文件 (config.yaml)

项目使用 `config.yaml` 进行配置，支持以下功能：

#### 服务器配置
```yaml
server:
  port: 8080        # 服务端口
  host: "0.0.0.0"   # 监听地址
```

#### 认证配置
```yaml
auth:
  username: "admin"    # 登录用户名
  password: "password" # 登录密码
```

#### 项目配置
```yaml
projects:
  - name: "项目名称"
    path: "/项目/路径"
    description: "项目描述"
    enabled: true      # 是否启用
```

#### 界面配置
```yaml
ui:
  title: "Git 版本管理工具"
  theme: "default"
  language: "zh-CN"
```

#### 安全配置
```yaml
security:
  enable_auth: true                                    # 是否启用认证
  session_timeout: 3600                               # 会话超时时间(秒)
  session_secret: "gover-secret-key"   # Session 密钥
  remember_me_days: 7                                  # 记住我功能天数
```

### 配置说明

所有配置都通过 `config.yaml` 文件进行管理。系统会自动：

- 创建 `conf` 目录（如果不存在）
- 生成 `conf/app.conf` 文件（如果不存在）
- 从 `config.yaml` 同步基本配置到 `app.conf`

> 📝 `conf/app.conf` 文件是自动生成的，请勿手动编辑

## 项目结构

```
gogo/
├── main.go                 # 主入口文件
├── config.yaml            # YAML 配置文件 (支持多项目)
├── models/
│   └── config.go          # 配置文件模型和加载逻辑
├── controllers/
│   └── version.go         # 版本管理控制器
├── views/
│   └── version/
│       └── index.html     # 版本管理页面模板
├── start.sh              # 启动脚本
└── README.md            # 项目说明
```

## 使用说明

1. **选择项目**: 在项目选择器中点击要管理的项目
2. **查看版本列表**: 选中项目后可看到该项目所有可用的 Git 标签版本
3. **识别当前版本**: 当前活跃的版本会以绿色背景显示
4. **版本回滚**: 点击任意版本旁的"回滚到此版本"按钮
5. **确认操作**: 系统会要求确认回滚操作
6. **操作反馈**: 成功或失败的消息会在页面顶部显示

### 多项目管理

- 通过修改 `config.yaml` 添加、删除或禁用项目
- 每个项目都有独立的版本管理
- 支持项目描述和路径显示
- 可以随时启用/禁用项目

### 版本信息显示

- **版本号排序**: 支持语义化版本号排序（v0.0.1, v0.0.2, ..., v0.0.12, v0.1.0）
- **创建时间**: 显示每个版本的创建时间
- **标签备注**: 显示 Git 标签的备注信息
- **提交哈希**: 显示对应的提交哈希值（前7位）
- **当前版本**: 高亮显示当前活跃的版本

## 注意事项

- 确保目标目录是一个有效的 Git 仓库
- 回滚操作会改变工作目录的代码状态
- 建议在生产环境使用前先在测试环境验证
- 如果没有 Git 标签，页面会显示"暂无可用版本标签"

## 安全提醒

- 请及时修改默认的认证密码
- 建议在生产环境中使用更强的认证机制
- 对于重要的生产系统，建议添加更多的安全检查

## 故障排除

如果遇到问题：

1. 检查 Git 仓库状态：`git status`
2. 确认有可用的标签：`git tag -l`
3. 检查目录权限
4. 查看应用日志输出

## 📚 文档

- [功能特性详情](FEATURES.md)
- [安全升级说明](SECURITY.md)
- [Session 管理测试](SESSION_TEST.md)
- [启动警告说明](NOTICE.md)
- [GitHub Actions 自动化指南](GITHUB_ACTIONS.md)

## 🚀 自动化构建

本项目已配置 GitHub Actions 自动化工作流：

- **持续集成**: 推送代码时自动测试和构建
- **自动发布**: 推送标签时自动构建多平台版本并创建 Release
- **代码质量检查**: 自动运行格式化、静态分析和安全扫描

### 创建新版本发布

```bash
# 创建标签
git tag -a v1.0.0 -m "Release v1.0.0"

# 推送标签触发自动发布
git push origin v1.0.0
```

详细说明请参考 [GitHub Actions 自动化指南](GITHUB_ACTIONS.md)。 