# 变更日志

项目的所有重要变更都记录在此文件中。

## [1.0.1] - 2026-01-21

### 🔧 错误修复

- **修复 macOS 上应用频繁重启的问题**
  - 根本原因：`watcher.Close()` 方法未关闭 `eventsChan`
  - 影响：应用在优雅关闭时挂起，导致 LaunchAgent 反复重启
  - 解决方案：在 watcher 模块中添加正确的通道关闭
  - 文件：`watcher/watcher.go`

### ✨ 新增特性

- **通过 YAML 配置自定义日志文件路径**
  - 在 config.yaml 中添加 `log_path` 字段以指定自定义日志位置
  - 支持相对路径和绝对路径
  - 默认：未配置时使用 `logs/cos-uploader.log`
  - 文件：`config/config.go`、`logger/logger.go`、`main.go`

- **增强日志模块**
  - 新增 `NewLoggerWithPath(logPath string)` 函数用于自定义路径
  - 保持与 `NewLogger()` 的向后兼容性
  - 自动创建日志目录
  - 文件：`logger/logger.go`

### 📁 目录结构改进

- **标准化安装目录**
  - 推荐：`/opt/cos-uploader/`
  - 目录结构：
    ```
    /opt/cos-uploader/
    ├── cos-uploader           # 应用程序
    ├── config.yaml            # 配置文件
    └── logs/
        └── cos-uploader.log   # 应用日志
    ```
  - 所有应用文件集中在一处
  - 提高可维护性和管理便利性

### 📚 文档更新

- **MACOS_BACKGROUND_SETUP.md**
  - 完全重写以适配新的 `/opt/cos-uploader/` 目录结构
  - 添加详细的日志配置指南
  - 改进故障排查部分
  - 更新安装脚本使用新路径
  - 添加日志目录和配置说明

- **README.md**
  - 添加 `log_path` 配置文档
  - 添加目录结构章节
  - 增强日志配置部分及路径配置说明
  - 改进应用崩溃问题的故障排查

- **BUILD_GUIDE.md**（待更新）
  - 将反映新的构建路径和安装位置

### 🔨 实现细节

#### Config 模块的变更
```go
type Config struct {
    Projects []ProjectConfig `yaml:"projects"`
    LogPath  string          `yaml:"log_path"` // 新增
}
```

#### Logger 模块的变更
```go
// 新增：支持自定义日志路径
func NewLoggerWithPath(logPath string) *Logger {
    logDir := filepath.Dir(logPath)
    os.MkdirAll(logDir, 0755)
    // ... 创建日志文件
}

// 现有：为向后兼容性保留
func NewLogger() *Logger {
    return NewLoggerWithPath("logs/cos-uploader.log")
}
```

#### Main 模块的变更
```go
// 新增：先加载配置，然后用配置的日志路径初始化日志
cfg, err := config.LoadConfig(*configPath)
if err != nil {
    println("Failed to load config:", err)
    os.Exit(1)
}

var log *logger.Logger
if cfg.LogPath != "" {
    log = logger.NewLoggerWithPath(cfg.LogPath)
} else {
    log = logger.NewLogger()
}
```

#### Watcher 模块的变更
```go
// 修复：在关闭时正确关闭 eventsChan
func (w *Watcher) Close() error {
    w.mu.Lock()
    if w.closed {
        w.mu.Unlock()
        return nil
    }
    w.closed = true
    w.mu.Unlock()

    close(w.done)
    time.Sleep(100 * time.Millisecond)

    // 新增：关闭 eventsChan 以允许主循环退出
    close(w.eventsChan)

    return w.watcher.Close()
}
```

### 🚀 迁移指南

从 v1.0.0 升级到 v1.0.1 的用户：

1. **更新配置文件** - 在最前面添加 `log_path`：
   ```yaml
   log_path: /opt/cos-uploader/logs/cos-uploader.log

   projects:
     - name: ...
   ```

2. **可选：迁移到新的目录结构**
   ```bash
   mkdir -p /opt/cos-uploader
   cp cos-uploader /opt/cos-uploader/
   cp config.yaml /opt/cos-uploader/
   mkdir -p /opt/cos-uploader/logs
   ```

3. **更新 LaunchAgent（仅 macOS）**
   ```xml
   <string>/opt/cos-uploader/cos-uploader</string>
   <string>/opt/cos-uploader/config.yaml</string>
   <string>/opt/cos-uploader</string>  <!-- WorkingDirectory -->
   ```

4. **重新加载应用**
   ```bash
   launchctl stop com.hmw.cos-uploader
   launchctl start com.hmw.cos-uploader
   ```

### ⚠️ 已知问题

- 暂无

### 📋 测试备注

- ✅ 在 macOS 上使用 LaunchAgent 进行了完整测试
- ✅ 使用自定义日志路径（相对和绝对）进行了测试
- ✅ 验证了优雅关闭工作正常
- ✅ 确认不再出现频繁重启
- ✅ 文件监控和上传功能正常运行

### 🙏 致谢

- 修复者：开发团队
- 测试环境：macOS 12.x+
- Go 版本：1.21+

---

## [1.0.0] - 2026-01-20

### 🎉 首次发布

- 使用 fsnotify 实现实时文件监控
- 多项目支持，每个项目拥有独立的 COS 桶
- 并发文件上传与工作线程池
- 自动重试机制（3 次尝试）
- DingTalk webhook 失败通知
- 结构化 JSON 日志记录到 stdout 和文件

### ✨ 特性

- 递归目录监控
- 可配置的上传并发数
- 基于事件的过滤（create、write、remove、rename、chmod）
- 完整的上传能力和进度跟踪
- 基于索引的上传去重
- 跨平台支持（Linux、macOS、Windows）
- GitHub Actions CI/CD 流水线
- GoReleaser 自动化构建
