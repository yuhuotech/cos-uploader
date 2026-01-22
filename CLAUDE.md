# 开发者文档

本文件为使用 Claude Code 在本项目中工作提供指导。

## 项目简介

COS 文件监控上传工具是一个高性能的 Go 应用，监控本地文件系统变化并自动上传修改的文件到腾讯云对象存储（COS）。主要功能包括：

- 多项目配置支持，每个项目有独立的 COS 桶
- 使用 fsnotify 实现毫秒级实时文件监控
- 基于工作线程池的并发上传
- 自动重试机制（最多 3 次）
- DingTalk webhook 失败通知
- 结构化 JSON 日志记录

## 架构设计

### 整体架构

应用采用**事件驱动、模块化架构**，各模块职责分离清晰：

```
用户配置 (config.yaml)
         ↓
[配置模块] → 验证和提供项目配置
         ↓
[主程序] → 编排生命周期、启动监控和上传服务
    ├─ [监控模块] → 通过 fsnotify 监控文件、发送事件
    │              （每个项目、多个目录）
    ├─ [上传模块] → 管理上传队列和工作线程池
    │              （接收事件、上传到 COS）
    ├─ [日志模块] → JSON 结构化日志，输出到 stdout 和文件
    ├─ [告警模块] → 失败时发送 DingTalk webhook
    └─ [版本信息] → 通过 ldflags 注入版本号
```

### 数据流向

1. **启动流程**：`main` → 加载配置 → 创建监控和上传服务 → 启动
2. **文件变化**：fsnotify 事件 → 监控器过滤 → 队列任务 → 工作线程处理
3. **上传失败**：工作线程重试 3 次 → 失败则发送 DingTalk 告警

### 并发模型

#### 1. 监控器 → 事件通道
- `watcher.go`：`Start()` 启动 goroutine，将事件发送到缓冲通道（100）
- 按类型过滤事件（create、write、remove、rename、chmod）
- **重要**：使用 `done` 通道实现优雅关闭；必须先关闭 `done` 再关闭事件通道
- **安全**：Mutex 保护 `closed` 标志，防止双重关闭 panic

#### 2. 主事件循环
- `main.go`：对每个项目的 `watcher.Events()` 进行 range 循环（独立 goroutine）
- 发送 `UploadTask` 到上传器队列
- 通过 `WaitGroup` 协调有序关闭

#### 3. 上传工作线程池
- `uploader/uploader.go`：固定大小的 goroutine 池（默认 5，可配置）
- `WorkerPool.worker()`：无限循环消费任务通道
- **重试逻辑**：错误时，如果 `Retry < 3` 则重新加入队列；否则记录失败
- **告警集成**：失败上传（3 次重试后）应触发告警模块

#### 4. 优雅关闭
- `main.go`：捕获 SIGINT/SIGTERM
- 关闭所有监控器 → 等待 `watcherGroup` → 停止上传器
- 上传器关闭 `done` 通道 → 工作线程退出 → `WaitGroup.Wait()` 返回

## 模块详解

### Config 模块
**文件**：`config/config.go`

- **结构**：`Config`（`ProjectConfig` 数组），每个项目包含 COS、监控、告警配置
- **LoadConfig()**：读取 YAML、验证、设置默认值（region="ap-shanghai"、pool_size=5、events=[create,write]）
- **验证**：确保每个项目的 name、directories、bucket、凭证非空
- **测试覆盖率**：92.9%，包括错误情况（缺少字段、空项目）

### Watcher 模块
**文件**：`watcher/watcher.go`

- **结构**：`Watcher` 包装 `fsnotify.Watcher`，管理目录、事件类型、通道
- **NewWatcher()**：初始化 fsnotify、添加目录
- **Start()**：Goroutine 循环消费 fsnotify 事件，通过 `shouldWatch()` 过滤，发送到 `eventsChan`
- **Close()**：Mutex 保护防止双重关闭；关闭 `done` 信号、等待 100ms、关闭 watcher
- **支持的事件**：create、write、remove、rename、chmod
- **测试覆盖率**：47.3%，包括 `-race` 并发安全测试

### Uploader 模块
**文件**：`uploader/queue.go`、`uploader/uploader.go`

- **Queue**：简单的基于通道的任务队列
- **Uploader**：管理 COS 客户端（每个项目）、编排工作线程池、处理重试
- **WorkerPool**：固定大小的 goroutine 池、工作循环和重试逻辑
- **COS 集成**：使用 `tencentyun/cos-go-sdk-v5`，通过 SecretID/SecretKey 认证
- **重试机制**：3 次尝试、每次失败都记录、第 3 次失败后发送 DingTalk 告警
- **测试覆盖率**：6.4%（基础队列/线程池测试）；COS 操作手动测试

### Logger 模块
**文件**：`logger/logger.go`

- 灵活的结构化日志，支持自定义文件路径（v1.0.1 新增）
- 双输出：stdout（INFO+）和文件（DEBUG+）
- 两种初始化方式：
  - `NewLogger()`：使用默认路径 `logs/cos-uploader.log`（向后兼容）
  - `NewLoggerWithPath(logPath string)`：使用配置中的自定义路径（新增）
- 自动创建日志目录
- 支持绝对路径（`/opt/cos-uploader/logs/app.log`）和相对路径（`logs/app.log`）
- **测试覆盖率**：80%，验证目录/文件创建和自定义路径处理

### Alert 模块
**文件**：`alert/alert.go`

- **SendAlert()**：HTTP POST 到 DingTalk webhook，发送 JSON 消息
- **SendUploadFailureAlert()**：便利方法，格式化上传错误详情
- 如未配置 webhook，则记录到日志（优雅降级）
- **测试覆盖率**：30.4%，基础初始化和消息格式测试

## 开发环境

### 环境搭建

```bash
# 克隆仓库
git clone <repo-url>
cd cos-uploader

# 下载依赖
go mod download

# 创建测试配置
cp example-config.yaml config.yaml
# 编辑 config.yaml，填入你的 COS 凭证和测试路径

# 构建
go build -o cos-uploader

# 运行
./cos-uploader -config config.yaml
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./config -v
go test ./watcher -v
go test ./uploader -v

# 运行并发安全检测
go test -race ./...

# 运行单个测试
go test -run TestNewLogger ./logger -v

# 查看测试覆盖率
go test ./config -cover
```

### 代码规范

```bash
# 格式化代码
go fmt ./...

# 静态检查
go vet ./...

# 完整检查
go build ./... && go test ./... && go vet ./...
```

## 核心实现

### 构建命令

```bash
# 简单构建（当前 OS）
go build -o cos-uploader

# 跨平台构建
GOOS=linux GOARCH=amd64 go build -ldflags="-X main.Version=v1.0.0" -o cos-uploader-linux-amd64
GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.Version=v1.0.0" -o cos-uploader-darwin-arm64
GOOS=windows GOARCH=amd64 go build -ldflags="-X main.Version=v1.0.0" -o cos-uploader-windows-amd64.exe

# 使用 GoReleaser（自动多平台）
goreleaser release --snapshot --skip-publish --rm-dist
```

### 运行应用

```bash
# 显示版本
./cos-uploader --version

# 使用默认配置运行
./cos-uploader

# 使用自定义配置
./cos-uploader -config /path/to/config.yaml

# 使用环境变量配置路径
export COS_UPLOADER_CONFIG=/etc/cos-uploader/config.yaml
./cos-uploader -config $COS_UPLOADER_CONFIG
```

### 已知问题与陷阱

#### 1. 事件通道关闭问题（v1.0.1 已修复）
- **问题**：Watcher 的 `Close()` 方法未关闭 `eventsChan`，导致主事件循环无限挂起
- **症状**：应用无法优雅关闭；macOS LaunchAgent 会反复重启
- **修复**：在 watcher `Close()` 方法中添加 `close(w.eventsChan)`
- **教训**：多 goroutine 场景下，发送端必须关闭通道

#### 2. 竞态条件
- **问题**：原实现在关闭时存在潜在竞态条件
- **修复**：添加 `sync.Mutex` 和 `closed` 标志；等待 100ms 让 goroutine 退出后再关闭资源
- **教训**：多 goroutine 访问的状态标志必须用 mutex 保护

#### 3. 事件去重
- **当前**：无去重；快速写入可能为单个文件生成多个事件
- **考虑**：高频文件写入（如日志文件）可能导致上传频繁
- **未来增强**：添加事件聚合/批处理

#### 4. 符号链接和目录遍历
- **状态**：v1.0.0+ 已解决
- **实现**：`addRecursive()` 使用 `filepath.Walk()` 监控所有子目录
- **行为**：自动递归监控所有嵌套目录

#### 5. COS SDK 错误处理
- **问题**：SDK 可能返回无上下文的非 EOF 错误
- **当前**：用 `fmt.Errorf` 包装错误以添加上下文
- **测试**：COS 集成测试有限（需要凭证、网络）

#### 6. 可配置日志路径（v1.0.1 新增）
- **特性**：应用现支持通过 `log_path` 配置自定义日志路径
- **实现**：`logger.NewLoggerWithPath()` 在指定路径创建日志
- **优点**：集中化日志目录、更易维护
- **示例**：`log_path: /opt/cos-uploader/logs/cos-uploader.log`

#### 7. 版本注入
- **构建**：编译时使用 `-ldflags="-X main.Version=v1.0.0"`
- **默认值**：未注入时 `Version = "dev"`（见 `version.go`）
- **CLI**：使用 `./cos-uploader --version` 验证

## 测试策略

- **单元测试**：每个模块有对应的 `_test.go` 文件
- **集成测试**：通过完整生命周期测试 `main.go`
- **并发检测**：运行 `go test -race ./...` 捕获数据竞争
- **配置验证**：测试覆盖正常路径和验证失败情况
- **覆盖率目标**：核心逻辑（config、logger）>80%；可选特性较低

### 添加新测试

1. 在同一包内创建 `*_test.go` 文件
2. 使用表驱动测试处理多个场景
3. 必须测试错误路径
4. 文件操作使用 `t.TempDir()`
5. 运行时带上 `-race` 标志

## 部署指南

### 推荐目录结构

```
/opt/cos-uploader/
├── cos-uploader           # 应用程序
├── config.yaml            # 配置文件（包含 log_path）
└── logs/
    └── cos-uploader.log   # 应用日志
```

此结构将所有应用文件集中在一处，便于管理。

### Linux 部署（systemd）
**文件**：`cos-uploader.service`

- **Type**：`simple`（前台进程，无守护化）
- **User**：`ubuntu`（根据需要更新）
- **ExecStart**：`/opt/cos-uploader/cos-uploader -config /opt/cos-uploader/config.yaml`
- **WorkingDirectory**：`/opt/cos-uploader`
- **Restart**：`always`，10 秒回退
- **日志**：由应用管理（见 config.yaml 的 `log_path`），也由 systemd journal 捕获
- **安装**：`sudo cp cos-uploader.service /etc/systemd/system/` → `sudo systemctl daemon-reload` → `sudo systemctl enable --now cos-uploader`

### GitHub Actions
**文件**：`.github/workflows/build-release.yml`

- **触发**：手动工作流分派，输入版本
- **平台**：构建 6 个变体（linux/darwin/windows × amd64/arm64）
- **步骤**：Checkout → Setup Go → Test → Build → Archive → Upload Artifacts → Create Release
- **耗时**：所有平台约 15-20 分钟
- **门控**：创建 release 前必须所有测试通过

### 本地 Release 构建
**文件**：`.goreleaser.yml`

- 自动化多平台构建、校验和、GitHub Release 创建
- 本地测试：`goreleaser release --snapshot --skip-publish --rm-dist`
- 需要 git 标签：`git tag -a v1.0.0 -m "Release"`

## 关键依赖

- `github.com/fsnotify/fsnotify` (v1.9.0)：文件系统事件监控
- `github.com/tencentyun/cos-go-sdk-v5` (v0.7.72)：腾讯云 COS API 客户端
- `go.uber.org/zap` (v1.27.1)：结构化日志
- `gopkg.in/yaml.v3` (v3.0.1)：YAML 解析

所有依赖应保持稳定；重大变更需要主版本号更新。

## 扩展开发

### 添加新功能

1. 评估是否需要新模块或增强现有模块
2. 编写测试（测试驱动开发）
3. 实现功能
4. 确保测试通过和覆盖率达到标准
5. 更新文档

### 添加新的存储后端

1. 在 `uploader/` 中创建新的后端实现
2. 实现统一的上传接口
3. 在 config 中添加后端配置项
4. 添加对应的测试
5. 更新主程序的后端选择逻辑

### 自定义告警方式

1. 在 `alert/` 中扩展 `SendAlert()` 或创建新的告警函数
2. 在 config 中添加新的告警配置项
3. 在上传失败时调用新的告警函数
4. 添加测试覆盖

## 文档文件

- `README.md`：面向用户的特性概述、快速开始和配置指南
- `docs/MACOS_BACKGROUND_SETUP.md`：完整的 macOS LaunchAgent 设置和故障排查指南
- `docs/BUILD_GUIDE.md`：本地构建说明、跨平台编译、GoReleaser 使用
- `CHANGELOG.md`：版本历史、特性、bug 修复和迁移指南
- `docs/GITHUB_WORKFLOW_GUIDE.md`：GitHub Actions 工作流详解
- `docs/RELEASE_WORKFLOW_README.md`：完整的 release 流程指南
- `CLAUDE.md`：本文件 - 开发者文档和架构详解
- `docs/plans/2026-01-20-cos-uploader-implementation.md`：实现架构和任务分解

### v1.0.1 文档更新

- **docs/MACOS_BACKGROUND_SETUP.md**：完全重写以适应新的 `/opt/cos-uploader/` 目录结构和日志配置
- **README.md**：添加 `log_path` 配置文档和目录结构指南
- **CLAUDE.md**：更新 v1.0.1 变更，包括 eventsChan 修复和日志配置
- **CHANGELOG.md**：创建以记录版本历史和迁移指南

## 参考资源

- [Go 官方文档](https://golang.org/doc/)
- [fsnotify 文档](https://github.com/fsnotify/fsnotify)
- [腾讯云 COS SDK](https://github.com/tencentyun/cos-go-sdk-v5)
- [Zap 日志库](https://github.com/uber-go/zap)
