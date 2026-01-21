# GitHub Workflow 使用指南

## Build Release Workflow

这个 workflow 用于自动化构建和发布多平台可执行文件到 GitHub Release。

### 支持的平台和架构

| 操作系统 | 架构 | 文件名 | 文件大小 |
|---------|------|--------|---------|
| Linux | amd64 | `cos-uploader-vX.X.X-linux_amd64.tar.gz` | ~10MB |
| Linux | arm64 | `cos-uploader-vX.X.X-linux_arm64.tar.gz` | ~10MB |
| macOS | amd64 (Intel) | `cos-uploader-vX.X.X-darwin_amd64.tar.gz` | ~10MB |
| macOS | arm64 (Apple Silicon) | `cos-uploader-vX.X.X-darwin_arm64.tar.gz` | ~10MB |
| Windows | amd64 | `cos-uploader-vX.X.X-windows_amd64.zip` | ~10MB |
| Windows | arm64 | `cos-uploader-vX.X.X-windows_arm64.zip` | ~10MB |

### 使用方法

#### 1. 通过 GitHub Web 界面触发

1. 打开项目的 GitHub 页面
2. 点击 **Actions** 选项卡
3. 在左侧选择 **Build Release** workflow
4. 点击 **Run workflow** 按钮
5. 填入参数：
   - **Release version**: 版本号，格式为 `vX.X.X`（例如：`v1.0.0`）
   - **Create as draft release**: 勾选则创建为草稿版本（可选）
   - **Mark as prerelease**: 勾选则标记为预发布版本（可选）
6. 点击 **Run workflow** 开始构建

#### 2. 通过 GitHub CLI 触发

```bash
# 安装 GitHub CLI (https://cli.github.com/)
gh workflow run build-release.yml \
  -f version=v1.0.0 \
  -f draft=false \
  -f prerelease=false
```

#### 3. 使用 curl 触发（需要 Personal Access Token）

```bash
curl -X POST \
  -H "Authorization: token YOUR_GITHUB_TOKEN" \
  -H "Content-Type: application/json" \
  https://api.github.com/repos/YOUR_USERNAME/cos-uploader/actions/workflows/build-release.yml/dispatches \
  -d '{"ref":"main","inputs":{"version":"v1.0.0","draft":"false","prerelease":"false"}}'
```

### Workflow 执行过程

1. **检出代码** (Checkout code)
   - 获取最新的源代码

2. **设置 Go 环境** (Set up Go)
   - 安装 Go 1.21
   - 启用 Go 模块缓存加速

3. **运行测试** (Run tests)
   - 执行 `go test ./...` 确保所有测试通过
   - 如果测试失败，workflow 会停止

4. **构建可执行文件** (Build)
   - 为每个平台和架构编译二进制文件
   - 使用 ldflags 注入版本信息
   - Linux/macOS: 生成 `cos-uploader`
   - Windows: 生成 `cos-uploader.exe`

5. **打包文件** (Create archive)
   - Linux/macOS: 使用 tar.gz 压缩
   - Windows: 使用 ZIP 压缩

6. **上传工件** (Upload artifacts)
   - 将编译好的文件上传为临时工件
   - 保留期为 7 天

7. **创建 Release** (Create GitHub Release)
   - 为指定版本创建 GitHub Release
   - 上传所有平台的可执行文件
   - 自动生成下载说明

8. **通知** (Notify Build Status)
   - 在 Workflow 摘要中显示构建状态

### 下载发布的文件

构建完成后，可以通过以下方式下载文件：

#### 1. 从 GitHub Release 页面下载

1. 打开项目的 GitHub 页面
2. 点击右侧的 **Releases**
3. 找到对应版本
4. 在 **Assets** 部分下载所需的文件

#### 2. 使用命令行下载

```bash
# 下载 Linux amd64 版本
wget https://github.com/YOUR_USERNAME/cos-uploader/releases/download/v1.0.0/cos-uploader-v1.0.0-linux_amd64.tar.gz

# 或使用 curl
curl -L -o cos-uploader-v1.0.0-linux_amd64.tar.gz \
  https://github.com/YOUR_USERNAME/cos-uploader/releases/download/v1.0.0/cos-uploader-v1.0.0-linux_amd64.tar.gz
```

### 安装发布的可执行文件

#### Linux/macOS

```bash
# 下载文件
tar -xzf cos-uploader-v1.0.0-linux_amd64.tar.gz

# 赋予执行权限
chmod +x cos-uploader

# 移到系统路径（可选）
sudo mv cos-uploader /usr/local/bin/

# 验证版本
./cos-uploader --version
# 输出: COS Uploader v1.0.0
```

#### Windows

```powershell
# 解压文件
Expand-Archive cos-uploader-v1.0.0-windows_amd64.zip

# 进入目录
cd cos-uploader

# 运行程序
.\cos-uploader.exe -config config.yaml

# 查看版本
.\cos-uploader.exe --version
# 输出: COS Uploader v1.0.0
```

### 版本号规范

建议使用语义化版本号 (Semantic Versioning)：

- **v1.0.0**: 主版本.副版本.补丁版本
  - **主版本**: 不兼容的 API 更改
  - **副版本**: 向后兼容的功能新增
  - **补丁版本**: 向后兼容的问题修复

例如：
- v1.0.0 - 初始发布
- v1.1.0 - 新增功能
- v1.1.1 - 修复 Bug
- v2.0.0 - 重大更新（不兼容）

### 常见问题

#### Q: 如何修改支持的平台？

A: 编辑 `.github/workflows/build-release.yml` 中的 `strategy.matrix.include` 部分，调整或添加新的平台配置。

#### Q: 如何跳过测试直接构建？

A: 在 Workflow 文件中注释掉 "Run tests" 步骤（不推荐）。

#### Q: 如何添加 armv7 架构支持？

A: 在 matrix.include 中添加新的配置：

```yaml
- os: linux
  arch: armv7
  runner: ubuntu-latest
  target: linux_armv7
```

#### Q: 构建失败了怎么办？

A:
1. 检查 GitHub Actions 的日志
2. 确保代码中的所有测试都通过
3. 检查是否有依赖问题
4. 查看 Go 版本兼容性

#### Q: 如何预览 Release 内容？

A: 在触发 workflow 时勾选 "Create as draft release"，这样可以先预览 Release 内容，然后手动发布。

### Workflow 文件位置

- `.github/workflows/build-release.yml` - 主要的构建和发布 workflow

### 设置要求

1. **Git 权限**: 需要能够创建 Release
2. **Go 环境**: 项目已配置使用 Go 1.21
3. **GitHub Actions**: 项目已启用 GitHub Actions

### 性能指标

- 单个构建耗时: ~3-5 分钟
- 总计 6 个平台/架构组合: ~15-20 分钟
- 构建的可执行文件大小: ~10MB（压缩后）

### 许可证和开源

此 workflow 配置基于 GitHub Actions 最佳实践，可自由修改和使用。

### 更新日志

#### v1.0.0 (2026-01-20)
- 初始版本
- 支持 6 个平台/架构组合
- 自动化测试和构建
- 自动生成 GitHub Release
