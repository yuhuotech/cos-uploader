# 本地构建指南

## 快速构建

### 简单构建（当前系统）

```bash
go build -o cos-uploader
```

### 查看版本

```bash
./cos-uploader --version
```

### 跨平台构建（手动）

#### Linux 64-bit
```bash
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=v1.0.0" -o cos-uploader-linux-amd64
```

#### macOS Intel
```bash
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.Version=v1.0.0" -o cos-uploader-darwin-amd64
```

#### macOS Apple Silicon
```bash
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.Version=v1.0.0" -o cos-uploader-darwin-arm64
```

#### Windows 64-bit
```bash
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.Version=v1.0.0" -o cos-uploader-windows-amd64.exe
```

#### Windows ARM64
```bash
GOOS=windows GOARCH=arm64 go build -ldflags="-s -w -X main.Version=v1.0.0" -o cos-uploader-windows-arm64.exe
```

#### Linux ARM64
```bash
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.Version=v1.0.0" -o cos-uploader-linux-arm64
```

## 使用 GoReleaser 构建

GoReleaser 是一个强大的发布工具，可以自动化多平台构建和打包。

### 安装 GoReleaser

#### macOS
```bash
brew install goreleaser
```

#### Linux
```bash
curl -sL https://git.io/goreleaser | bash
```

#### 其他系统
访问 [https://goreleaser.com/install/](https://goreleaser.com/install/)

### 快速使用

#### 1. 快照构建（本地测试）

不需要 git tag，直接构建当前代码：

```bash
goreleaser release --snapshot --skip-publish --rm-dist
```

这会生成所有平台的可执行文件在 `dist/` 目录：

```
dist/
├── cos-uploader-v0.0.0-next-linux-amd64/
├── cos-uploader-v0.0.0-next-linux-arm64/
├── cos-uploader-v0.0.0-next-darwin-amd64/
├── cos-uploader-v0.0.0-next-darwin-arm64/
├── cos-uploader-v0.0.0-next-windows-amd64/
├── cos-uploader-v0.0.0-next-windows-arm64/
└── checksums.txt
```

#### 2. 制作发布版本

首先创建 git tag：

```bash
# 创建版本标签
git tag -a v1.0.0 -m "Release version 1.0.0"

# 推送标签到远程
git push origin v1.0.0
```

然后构建发布版本：

```bash
goreleaser release --clean
```

这会：
1. 构建所有平台的可执行文件
2. 创建压缩包
3. 计算校验和
4. 创建 GitHub Release（需要配置 GITHUB_TOKEN）
5. 上传所有文件

#### 3. 本地构建而不发布

```bash
goreleaser build --snapshot --skip-publish
```

### GoReleaser 配置文件说明

配置文件位于: `.goreleaser.yml`

关键配置项：

```yaml
builds:              # 构建配置
  goos:              # 目标操作系统
  goarch:            # 目标架构
  ldflags:           # 链接标志（用于注入版本信息）

archives:            # 打包配置
  format:            # 压缩格式（tar.gz 或 zip）

release:             # GitHub Release 配置
  github:            # GitHub 配置
    owner:           # GitHub 用户名（需要修改）
    name:            # 仓库名称
```

### 环境变量设置

#### 设置 GitHub Token（用于发布 Release）

##### Linux/macOS
```bash
export GITHUB_TOKEN=your_github_token_here
goreleaser release
```

##### Windows PowerShell
```powershell
$env:GITHUB_TOKEN='your_github_token_here'
goreleaser.exe release
```

### 常见问题

#### Q: GoReleaser 找不到 git tag

A: 确保你已经创建并推送了 git tag：
```bash
git tag -a v1.0.0 -m "Release"
git push origin v1.0.0
```

#### Q: 如何只构建特定平台？

A: 使用 `-os` 和 `-arch` 参数：
```bash
goreleaser build --snapshot -os linux -arch amd64
```

#### Q: 如何修改 GoReleaser 的行为？

A: 编辑 `.goreleaser.yml` 配置文件，然后重新运行。

#### Q: 构建 Windows ARM64 失败？

A: Windows ARM64 支持需要 Go 1.21+。如果还有问题，可以在 `.goreleaser.yml` 中跳过此平台：
```yaml
overrides:
  - goos: windows
    goarch: arm64
    skip: true
```

### 输出文件

构建完成后，`dist/` 目录会包含：

- `cos-uploader-vX.X.X-OS-ARCH.tar.gz` - Linux/macOS 压缩包
- `cos-uploader-vX.X.X-windows-ARCH.zip` - Windows 压缩包
- `checksums.txt` - 所有文件的 SHA256 校验和
- `release.notes` - Release 说明（如果发布到 GitHub）

### 使用建议

1. **开发阶段**: 使用 `go build` 快速构建
2. **测试阶段**: 使用 `goreleaser build --snapshot` 构建所有平台
3. **发布阶段**:
   - 本地完整测试通过
   - 创建 git tag
   - 使用 GitHub Actions workflow 自动构建（推荐）
   - 或使用 `goreleaser release` 本地发布

### 脚本示例

#### 快速构建所有平台（shell 脚本）

创建 `build-all.sh`:

```bash
#!/bin/bash

VERSION="v1.0.0"
TARGETS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
  "windows/arm64"
)

mkdir -p builds

for target in "${TARGETS[@]}"; do
  os=$(echo $target | cut -d'/' -f1)
  arch=$(echo $target | cut -d'/' -f2)

  output="cos-uploader-${VERSION}-${os}-${arch}"
  if [ "$os" = "windows" ]; then
    output="${output}.exe"
  fi

  echo "Building $output..."
  GOOS=$os GOARCH=$arch go build -ldflags="-X main.Version=${VERSION}" -o "builds/${output}"
done

echo "Done! Builds are in builds/ directory"
```

运行：
```bash
chmod +x build-all.sh
./build-all.sh
```

### 相关链接

- GoReleaser 文档: https://goreleaser.com/
- Go 构建参考: https://golang.org/cmd/go/
- GitHub Actions 文档: https://docs.github.com/en/actions
