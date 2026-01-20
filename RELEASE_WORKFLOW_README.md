# GitHub Actions Release Workflow 完整指南

## 项目概览

本项目已配置完整的 GitHub Actions workflow，支持一键构建多平台可执行文件并自动发布到 GitHub Release。

## 文件清单

### 核心配置文件

| 文件 | 说明 |
|------|------|
| `.github/workflows/build-release.yml` | GitHub Actions workflow 配置 |
| `.goreleaser.yml` | GoReleaser 本地构建配置 |
| `version.go` | 版本信息文件 |

### 文档文件

| 文件 | 说明 |
|------|------|
| `RELEASE_WORKFLOW_README.md` | 本文件 |
| `GITHUB_WORKFLOW_GUIDE.md` | GitHub Actions 使用详细指南 |
| `BUILD_GUIDE.md` | 本地构建指南 |
| `README.md` | 项目使用文档 |

## 快速开始

### 方式 1: 通过 GitHub Web 界面（推荐）

1. **打开项目**
   - 访问项目的 GitHub 页面

2. **触发 Workflow**
   - 点击 **Actions** 选项卡
   - 选择 **Build Release** workflow
   - 点击 **Run workflow** 按钮

3. **填入版本信息**
   - **Release version**: 输入版本号（格式: `vX.X.X`，例如 `v1.0.0`）
   - **Create as draft release**: 可选，勾选则创建草稿
   - **Mark as prerelease**: 可选，勾选则标记为预发布

4. **点击 Run workflow 开始构建**

5. **等待完成**
   - 整个构建过程约 15-20 分钟
   - 完成后 Release 会自动发布到 GitHub Release 页面

### 方式 2: 使用 GitHub CLI

```bash
# 需要安装 GitHub CLI (https://cli.github.com/)
gh workflow run build-release.yml \
  -f version=v1.0.0 \
  -f draft=false \
  -f prerelease=false
```

### 方式 3: 本地构建（GoReleaser）

```bash
# 快照构建（测试用）
goreleaser release --snapshot --skip-publish --rm-dist

# 发布构建（需要 git tag）
git tag -a v1.0.0 -m "Release v1.0.0"
goreleaser release --clean
```

## 支持的平台

构建包括以下 6 个平台/架构组合：

```
✅ Linux x86_64       → cos-uploader-vX.X.X-linux_amd64.tar.gz
✅ Linux ARM64        → cos-uploader-vX.X.X-linux_arm64.tar.gz
✅ macOS Intel        → cos-uploader-vX.X.X-darwin_amd64.tar.gz
✅ macOS Apple Silicon → cos-uploader-vX.X.X-darwin_arm64.tar.gz
✅ Windows x86_64     → cos-uploader-vX.X.X-windows_amd64.zip
✅ Windows ARM64      → cos-uploader-vX.X.X-windows_arm64.zip
```

## Workflow 工作流程

```
┌─────────────────────────────────────┐
│  用户点击 "Run workflow" 或 CLI      │
└──────────┬──────────────────────────┘
           │
           ▼
┌─────────────────────────────────────┐
│  为 6 个平台并行构建                 │
│  ├─ 检出代码                        │
│  ├─ 设置 Go 环境 (1.21)             │
│  ├─ 运行测试                        │
│  ├─ 编译二进制文件                  │
│  ├─ 打包（tar.gz/zip）             │
│  └─ 上传为临时工件                  │
└──────────┬──────────────────────────┘
           │
           ▼
┌─────────────────────────────────────┐
│  创建 GitHub Release                │
│  ├─ 下载所有工件                    │
│  ├─ 生成安装说明                    │
│  ├─ 上传所有文件                    │
│  └─ 发布 Release                    │
└──────────┬──────────────────────────┘
           │
           ▼
┌─────────────────────────────────────┐
│  完成！用户可从 Release 页面下载    │
└─────────────────────────────────────┘
```

## 构建产物

### Linux/macOS 产物

```
cos-uploader-v1.0.0-linux_amd64.tar.gz
├── cos-uploader (可执行文件)
├── LICENSE
├── README.md
├── example-config.yaml
└── GITHUB_WORKFLOW_GUIDE.md
```

### Windows 产物

```
cos-uploader-v1.0.0-windows_amd64.zip
├── cos-uploader.exe (可执行文件)
├── LICENSE
├── README.md
├── example-config.yaml
└── GITHUB_WORKFLOW_GUIDE.md
```

## 安装已发布的版本

### Linux/macOS

```bash
# 下载
wget https://github.com/YOUR_USERNAME/cos-uploader/releases/download/v1.0.0/cos-uploader-v1.0.0-linux_amd64.tar.gz

# 提取
tar -xzf cos-uploader-v1.0.0-linux_amd64.tar.gz

# 赋予执行权限
chmod +x cos-uploader

# 运行
./cos-uploader --version
./cos-uploader -config config.yaml
```

### Windows

```powershell
# 下载并提取
Expand-Archive cos-uploader-v1.0.0-windows_amd64.zip

# 运行
.\cos-uploader.exe --version
.\cos-uploader.exe -config config.yaml
```

## 配置修改

### 修改 Workflow 行为

编辑 `.github/workflows/build-release.yml`:

#### 添加新平台

在 `strategy.matrix.include` 中添加新配置：

```yaml
- os: linux
  arch: riscv64
  runner: ubuntu-latest
  target: linux_riscv64
```

#### 修改 Go 版本

在 `setup-go` 步骤中修改：

```yaml
- uses: actions/setup-go@v4
  with:
    go-version: '1.22'  # 修改版本号
```

#### 修改工件保留时间

在 `upload-artifact` 步骤中修改：

```yaml
retention-days: 30  # 修改保留天数（默认 7）
```

### 修改 GoReleaser 配置

编辑 `.goreleaser.yml`:

#### 修改项目名称

```yaml
project_name: my-project
```

#### 修改输出目录

在 `archives` 部分中修改 `name_template`。

## GitHub Actions 环境配置

### 必要权限

Workflow 需要以下权限：

```yaml
permissions:
  contents: write  # 创建 Release
```

这已在 `.github/workflows/build-release.yml` 中配置。

### 环境变量

如需设置环境变量，在 Workflow 中添加：

```yaml
env:
  MY_VAR: value
```

## 常见问题解答

### Q: 构建失败怎么办？

**A:**
1. 检查 GitHub Actions 日志
2. 确保所有测试通过：`go test ./...`
3. 检查 Go 版本兼容性
4. 查看具体错误信息

### Q: 如何添加新的依赖？

**A:**
```bash
go get -u github.com/new/package
go mod tidy
git add go.mod go.sum
git commit -m "chore: add new dependency"
```

### Q: 如何跳过某个平台的构建？

**A:** 在 `.github/workflows/build-release.yml` 中注释或移除相应的 matrix 配置。

### Q: Release 可以修改吗？

**A:** 可以。使用 "Create as draft release" 选项预览 Release，然后在 GitHub 上手动编辑和发布。

### Q: 如何支持更多的架构（如 armv7）？

**A:** 在 matrix.include 中添加新配置，确保 runner 支持该架构的构建。

### Q: 构建成功但找不到文件？

**A:** 检查 GitHub Release 页面中的 **Assets** 部分，所有文件都在那里。

### Q: 如何从发布的版本回滚？

**A:** 删除 GitHub Release 和对应的 git tag，然后重新运行 Workflow。

## 性能指标

- **单个平台构建**: ~2-3 分钟
- **总计 6 个平台**: ~15-20 分钟（并行）
- **文件大小**: 每个可执行文件 ~10MB（压缩后）
- **网络传输**: ~60MB（所有平台总计）

## 最佳实践

### 版本号管理

使用语义化版本 (Semantic Versioning):

```
v[主版本].[副版本].[补丁版本]
  ↓        ↓        ↓
 API      功能     Bug修复
```

例如：
- v1.0.0 - 初始发布
- v1.1.0 - 新增功能
- v1.1.1 - 修复 Bug
- v2.0.0 - 重大更新（不兼容）

### 发布前检查清单

- ✅ 所有测试通过
- ✅ 代码审核完成
- ✅ CHANGELOG 已更新
- ✅ 文档已更新
- ✅ 版本号符合规范
- ✅ Git 无未提交的更改

### 发布流程

1. 完成所有开发和测试
2. 在 GitHub 创建 Release 草稿
3. 运行 Workflow 构建
4. 验证发布内容
5. 发布 Release

## 相关资源

- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [GoReleaser 文档](https://goreleaser.com/)
- [Go 构建参考](https://golang.org/cmd/go/)
- [语义化版本](https://semver.org/lang/zh-CN/)

## 获取帮助

- 查看 Workflow 日志：Actions → Build Release → 查看具体运行
- 参考 BUILD_GUIDE.md 了解本地构建
- 参考 GITHUB_WORKFLOW_GUIDE.md 了解详细步骤
- 查看项目 README.md 了解功能说明

## 许可证

此 Workflow 配置可自由使用和修改。

---

**最后更新**: 2026-01-20
**版本**: 1.0.0
