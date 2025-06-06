# GitHub Actions 自动化指南

## 🚀 自动化工作流概览

本项目配置了两个主要的 GitHub Actions 工作流：

### 1. 🔄 测试工作流 (`.github/workflows/test.yml`)

**触发条件:**
- 推送到 `main`、`master`、`develop` 分支
- 针对这些分支的拉取请求

**执行任务:**
- ✅ 代码格式检查 (`go fmt`)
- 🔍 静态分析 (`go vet`)
- 🏗️ 编译构建
- 📊 代码质量检查 (`golangci-lint`)
- 🔒 安全扫描 (`gosec`)
- 📦 上传开发版本

### 2. 🎯 发布工作流 (`.github/workflows/release.yml`)

**触发条件:**
- 推送以 `v` 开头的标签 (如 `v1.0.0`)

**执行任务:**
- 🏗️ 多平台构建 (Linux, macOS, Windows)
- 📦 创建发布包
- 🚀 自动创建 GitHub Release
- 📝 生成发布说明

## 📋 支持的构建平台

| 操作系统 | 架构 | 文件名格式 |
|----------|------|------------|
| Linux | AMD64 | `gover-{version}-linux-amd64.tar.gz` |
| Linux | ARM64 | `gover-{version}-linux-arm64.tar.gz` |
| macOS | Intel | `gover-{version}-darwin-amd64.tar.gz` |
| macOS | Apple Silicon | `gover-{version}-darwin-arm64.tar.gz` |
| Windows | AMD64 | `gover-{version}-windows-amd64.exe.zip` |
| Windows | ARM64 | `gover-{version}-windows-arm64.exe.zip` |

## 🔖 发布新版本

### 步骤1: 准备发布

1. **更新版本信息**
   - 确保代码已准备好发布
   - 更新 README.md 中的更新日志（如果需要）

2. **本地测试**
   ```bash
   # 构建测试
   go build -o gover .
   
   # 版本信息测试
   ./gover -version
   
   # 功能测试
   ./gover
   ```

### 步骤2: 创建标签

```bash
# 创建标签
git tag -a v1.0.0 -m "Release v1.0.0"

# 推送标签到远程仓库
git push origin v1.0.0
```

### 步骤3: 自动发布

- GitHub Actions 会自动触发
- 构建多平台二进制文件
- 创建 GitHub Release
- 上传所有构建产物

## 📁 版本信息注入

构建时会自动注入以下信息：

```go
// 这些变量在构建时通过 ldflags 注入
var (
    Version   = "v1.0.0"                    // 来自 git tag
    BuildTime = "2024-12-06 15:30:00 UTC"   // 构建时间
    GitCommit = "abc1234"                    // Git 提交哈希
)
```

**查看版本信息:**
```bash
./gover -version
```

## 🛠️ 本地构建脚本

如果需要本地构建发布版本，可以使用：

```bash
#!/bin/bash
# build.sh

VERSION=$(git describe --tags --always)
BUILD_TIME=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
GIT_COMMIT=$(git rev-parse --short HEAD)

LDFLAGS="-X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}' -X 'main.GitCommit=${GIT_COMMIT}'"

# Linux AMD64
GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o gover-linux-amd64 .

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o gover-darwin-amd64 .

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o gover-windows-amd64.exe .

echo "构建完成！"
```

## 🔍 工作流状态

### 查看构建状态

在 GitHub 仓库页面：
1. 点击 "Actions" 标签
2. 查看最新的工作流运行状态
3. 点击具体运行查看详细日志

### 添加状态徽章

可以在 README.md 中添加构建状态徽章：

```markdown
[![Test](https://github.com/mycoool/gover/actions/workflows/test.yml/badge.svg)](https://github.com/mycoool/gover/actions/workflows/test.yml)
[![Release](https://github.com/mycoool/gover/actions/workflows/release.yml/badge.svg)](https://github.com/mycoool/gover/actions/workflows/release.yml)
```

## 🐛 故障排除

### 构建失败
1. 检查 Go 版本兼容性
2. 确保所有依赖都在 `go.mod` 中
3. 运行 `go mod tidy` 整理依赖

### 发布失败
1. 确保有推送标签的权限
2. 检查标签格式是否正确 (v*.*.*)
3. 确保 GitHub Token 有足够权限

### 测试失败
1. 本地运行 `go fmt ./...`
2. 本地运行 `go vet ./...`
3. 修复代码质量问题

## 📝 自定义发布说明

如果需要自定义发布说明，可以：

1. **预发布**: 创建 draft release
2. **编辑**: 在 GitHub 界面编辑发布说明
3. **发布**: 手动发布

## 🔒 安全考虑

- 工作流使用 `GITHUB_TOKEN`，无需额外配置
- 所有构建在 GitHub 托管的运行器上执行
- 不会访问或存储敏感信息
- 构建产物公开可下载

---

**💡 提示**: 首次使用时，建议先创建一个测试标签 (如 `v0.0.1-test`) 来验证工作流是否正常运行。 