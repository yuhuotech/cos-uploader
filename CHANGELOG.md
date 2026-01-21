# Changelog

All notable changes to this project will be documented in this file.

## [1.0.1] - 2026-01-21

### 🔧 Bug Fixes

- **Fixed frequent application restarts on macOS**
  - Root cause: `eventsChan` was not being closed in `watcher.Close()` method
  - Impact: Application would hang during graceful shutdown, causing LaunchAgent to repeatedly restart it
  - Solution: Added proper channel closure in watcher module
  - Files: `watcher/watcher.go`

### ✨ New Features

- **Configurable log file path via YAML**
  - Add `log_path` field to config.yaml for custom log locations
  - Supports both relative and absolute paths
  - Default: `logs/cos-uploader.log` if not configured
  - Files: `config/config.go`, `logger/logger.go`, `main.go`

- **Enhanced logger module**
  - New `NewLoggerWithPath(logPath string)` function for custom paths
  - Maintains backward compatibility with `NewLogger()`
  - Automatic log directory creation
  - Files: `logger/logger.go`

### 📁 Directory Structure Improvements

- **Standardized installation directory**
  - Recommended: `/opt/cos-uploader/`
  - Structure:
    ```
    /opt/cos-uploader/
    ├── cos-uploader           # Application binary
    ├── config.yaml            # Configuration file
    └── logs/
        └── cos-uploader.log   # Application logs
    ```
  - All application files centralized in one location
  - Improved maintainability and management

### 📚 Documentation Updates

- **MACOS_BACKGROUND_SETUP.md**
  - Complete rewrite for new `/opt/cos-uploader/` directory structure
  - Added detailed log configuration guide
  - Improved troubleshooting section
  - Updated setup script to use new paths
  - Added log directory and configuration explanations

- **README.md**
  - Added `log_path` configuration documentation
  - Added directory structure section
  - Enhanced logging section with path configuration
  - Improved troubleshooting for application crashes

- **BUILD_GUIDE.md** (to be updated)
  - Will reflect new build paths and installation locations

### 🔨 Implementation Details

#### Changes to Config Module
```go
type Config struct {
    Projects []ProjectConfig `yaml:"projects"`
    LogPath  string          `yaml:"log_path"` // NEW
}
```

#### Changes to Logger Module
```go
// NEW: Support custom log paths
func NewLoggerWithPath(logPath string) *Logger {
    logDir := filepath.Dir(logPath)
    os.MkdirAll(logDir, 0755)
    // ... create log file
}

// EXISTING: Maintained for backward compatibility
func NewLogger() *Logger {
    return NewLoggerWithPath("logs/cos-uploader.log")
}
```

#### Changes to Main Module
```go
// NEW: Load config first, then initialize logger with config's log path
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

#### Changes to Watcher Module
```go
// FIXED: Properly close eventsChan on shutdown
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

    // NEW: Close eventsChan to allow main loop to exit
    close(w.eventsChan)

    return w.watcher.Close()
}
```

### 🚀 Migration Guide

Users upgrading from v1.0.0 to v1.0.1:

1. **Update configuration file** - Add `log_path` at the top:
   ```yaml
   log_path: /opt/cos-uploader/logs/cos-uploader.log

   projects:
     - name: ...
   ```

2. **Optional: Migrate to new directory structure**
   ```bash
   mkdir -p /opt/cos-uploader
   cp cos-uploader /opt/cos-uploader/
   cp config.yaml /opt/cos-uploader/
   mkdir -p /opt/cos-uploader/logs
   ```

3. **Update LaunchAgent (macOS only)**
   ```xml
   <string>/opt/cos-uploader/cos-uploader</string>
   <string>/opt/cos-uploader/config.yaml</string>
   <string>/opt/cos-uploader</string>  <!-- WorkingDirectory -->
   ```

4. **Reload application**
   ```bash
   launchctl stop com.hmw.cos-uploader
   launchctl start com.hmw.cos-uploader
   ```

### ⚠️ Known Issues

- None at this time

### 📋 Testing Notes

- ✅ Application tested on macOS with LaunchAgent
- ✅ Tested with custom log paths (both relative and absolute)
- ✅ Verified graceful shutdown works correctly
- ✅ Confirmed no more frequent restarts
- ✅ File monitoring and uploading working normally

### 🙏 Credits

- Fixed by: Development team
- Tested on: macOS 12.x+
- Go version: 1.21+

---

## [1.0.0] - 2026-01-20

### Initial Release

- Real-time file monitoring using fsnotify
- Multi-project support with separate COS buckets
- Concurrent file upload with worker pool
- Automatic retry mechanism (3 attempts)
- DingTalk webhook notifications for failures
- Structured JSON logging to stdout and files
- Comprehensive error handling
- Graceful shutdown support

### Features
- Recursive directory monitoring
- Configurable upload concurrency
- Event-based filtering (create, write, remove, rename, chmod)
- Full upload capability with progress tracking
- Index-based upload deduplication
- Cross-platform support (Linux, macOS, Windows)
- GitHub Actions CI/CD pipeline
- GoReleaser automated builds
