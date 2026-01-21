# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

COS File Monitor & Uploader is a high-performance Go application that monitors local file system changes and automatically uploads modified files to Tencent Cloud COS (Object Storage Service). It supports:
- Multi-project configurations with separate COS buckets
- Real-time file monitoring with millisecond-level detection (fsnotify)
- Concurrent uploads via a worker pool
- Automatic retry mechanism (3 retries)
- DingTalk webhook notifications for failures
- Structured JSON logging to both stdout and files

## Architecture

The application follows an **event-driven, modular architecture** with clean separation of concerns:

```
User Input (config.yaml)
         ↓
[Config Module] → Validates and provides project configurations
         ↓
[Main] → Orchestrates lifecycle, spawns watchers and uploader
    ├─ [Watcher Module] → Monitors files via fsnotify, emits events
    │                     (per project, for multiple directories)
    ├─ [Uploader Module] → Manages upload queue and worker pool
    │                      (receives events, uploads to COS)
    ├─ [Logger Module] → JSON structured logging to stdout + file
    ├─ [Alert Module] → Sends DingTalk webhooks on failures
    └─ [Version] → Version info injected via ldflags
```

### Key Data Flows

1. **Startup**: `main` → load config → create watchers & uploader → start services
2. **File Change**: fsnotify event → watcher filter → queue task → worker processes
3. **Upload Failure**: worker retries 3x → if all fail → alert module sends webhook

### Module Responsibilities

- **config**: YAML parsing, validation, default value assignment, log_path configuration. Located in `config/`
- **watcher**: fsnotify integration, event filtering, multi-directory support, recursive directory monitoring. Located in `watcher/`
- **uploader**: Task queue (1000 buffered channels), worker pool (goroutines), COS SDK integration. Located in `uploader/`
- **logger**: Flexible logging with customizable file paths, dual output (stdout + file), structured format. Located in `logger/`
- **alert**: HTTP POST to DingTalk webhooks. Located in `alert/`
- **main**: Lifecycle orchestration, signal handling (SIGINT/SIGTERM), goroutine coordination via WaitGroup, config-driven logging initialization

## Development Commands

### Building

```bash
# Simple build (current OS)
go build -o cos-uploader

# Cross-platform builds
GOOS=linux GOARCH=amd64 go build -ldflags="-X main.Version=v1.0.0" -o cos-uploader-linux-amd64
GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.Version=v1.0.0" -o cos-uploader-darwin-arm64
GOOS=windows GOARCH=amd64 go build -ldflags="-X main.Version=v1.0.0" -o cos-uploader-windows-amd64.exe

# Using GoReleaser (automated multi-platform)
goreleaser release --snapshot --skip-publish --rm-dist
```

### Testing

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./config -v
go test ./watcher -v
go test ./uploader -v

# Run with race detector (concurrent safety)
go test -race ./...

# Run single test
go test -run TestNewLogger ./logger -v

# Test coverage
go test ./config -cover
```

### Code Quality

```bash
# Format code
go fmt ./...

# Lint
go vet ./...

# Run all checks
go build ./... && go test ./... && go vet ./...
```

### Running

```bash
# Show version
./cos-uploader --version

# Run with default config.yaml
./cos-uploader

# Run with custom config
./cos-uploader -config /path/to/config.yaml

# With environment config path
export COS_UPLOADER_CONFIG=/etc/cos-uploader/config.yaml
./cos-uploader -config $COS_UPLOADER_CONFIG
```

### Local Development Setup

```bash
# Install dependencies
go mod download

# Create test config
cp example-config.yaml config.yaml
# Edit config.yaml with your COS credentials and test paths

# Build and test
go build -o cos-uploader
./cos-uploader -config config.yaml

# Check logs
tail -f logs/cos-uploader.log
```

## Critical Concurrency Patterns

The application uses Go's concurrency primitives extensively. Key patterns to understand:

### 1. Watcher → Event Channel
- `watcher.go`: `Start()` spawns goroutine that sends events to buffered channel (100)
- Events filtered by type (create, write, remove, rename, chmod)
- **Important**: Uses `done` channel for graceful shutdown; must close `done` before closing event channel
- **Safety**: Mutex protects `closed` flag to prevent double-close panic

### 2. Main Event Loop
- `main.go`: Range loop over `watcher.Events()` for each project (separate goroutine per project)
- Posts `UploadTask` to uploader's queue
- Coordinates with `WaitGroup` for orderly shutdown

### 3. Uploader Worker Pool
- `uploader/uploader.go`: Fixed-size goroutine pool (default 5, configurable)
- `WorkerPool.worker()`: Infinite loop consuming from task channel
- **Retry Logic**: On error, re-queues task if `Retry < 3`; otherwise logs failure
- **Alert Integration**: Failed uploads (after 3 retries) should trigger alert module

### 4. Graceful Shutdown
- `main.go`: Catches SIGINT/SIGTERM
- Closes all watchers → awaits `watcherGroup` → stops uploader
- Uploader closes `done` channel → workers exit → `WaitGroup.Wait()` returns

## Module Deep Dives

### Config Module
**File**: `config/config.go`

- Struct: `Config` (array of `ProjectConfig`), each with COS, Watcher, Alert configs
- `LoadConfig()`: Reads YAML, validates, sets defaults (region="ap-shanghai", pool_size=5, events=[create,write])
- **Validation**: Ensures non-empty name, directories, bucket, credentials for each project
- **Testing**: 92.9% coverage; includes error cases (missing fields, empty projects)

### Watcher Module
**File**: `watcher/watcher.go`

- Struct: `Watcher` wraps `fsnotify.Watcher`, manages directories, event types, channels
- `NewWatcher()`: Initializes fsnotify, adds directories (simplified: only adds top level)
- `Start()`: Goroutine loop consuming fsnotify events, filters by `shouldWatch()`, sends to `eventsChan`
- `Close()`: Mutex-protected to prevent double-close; closes `done` signal, waits 100ms, closes watcher
- **Events Supported**: create, write, remove, rename, chmod
- **Testing**: 47.3% coverage; includes goroutine safety tests via `-race` flag

### Uploader Module
**Files**: `uploader/queue.go`, `uploader/uploader.go`

- `Queue`: Simple channel-based task queue
- `Uploader`: Manages COS clients (per project), orchestrates worker pool, handles retries
- `WorkerPool`: Fixed-size goroutine pool, worker loop with retry logic
- **COS Integration**: Uses `tencentyun/cos-go-sdk-v5`, authenticates via SecretID/SecretKey
- **Retry**: 3 attempts, logs after each failure, sends alert to DingTalk on final failure
- **Testing**: 6.4% coverage (basic queue/pool tests); COS operations tested manually

### Logger Module
**File**: `logger/logger.go`

- Flexible structured logging with customizable file paths (NEW in v1.0.1)
- Dual output: stdout (INFO+) and file (DEBUG+)
- Two initialization methods:
  - `NewLogger()`: Uses default path `logs/cos-uploader.log` (backward compatible)
  - `NewLoggerWithPath(logPath string)`: Uses custom path from config (NEW)
- Automatically creates log directories as needed
- Supports both absolute paths (`/opt/cos-uploader/logs/app.log`) and relative paths (`logs/app.log`)
- **Testing**: 80% coverage; verifies directory/file creation and custom path handling

### Alert Module
**File**: `alert/alert.go`

- `SendAlert()`: HTTP POST to DingTalk webhook with JSON message
- `SendUploadFailureAlert()`: Convenience method, formats upload error details
- Logs to logger if webhook not configured (graceful degradation)
- **Testing**: 30.4% coverage; basic initialization and message format tests

## Gotchas & Common Issues

### 1. Event Channel Closure (FIXED in v1.0.1)
- **Issue**: Watcher's `Close()` method did not close `eventsChan`, causing main event loop to hang indefinitely
- **Symptom**: Application would not shut down gracefully; LaunchAgent on macOS would repeatedly restart it
- **Fix**: Added `close(w.eventsChan)` in watcher `Close()` method to allow main loop's range statement to terminate
- **Lesson**: Always close send channels from the sender side when multiple goroutines are involved

### 2. Race Conditions
- **Issue**: Original implementation had potential race condition during shutdown
- **Fix**: Added `sync.Mutex` with `closed` flag; waits 100ms for goroutine to exit before closing resources
- **Lesson**: Always use mutex for state flags accessed from multiple goroutines

### 3. Event Deduplication
- **Current**: No deduplication; rapid writes may generate multiple events per file
- **Consideration**: High-frequency file writes (e.g., log files) may cause upload spam
- **Future Enhancement**: Add event aggregation/batching

### 4. Symlinks and Directory Traversal
- **Status**: RESOLVED in v1.0.0+
- **Implementation**: `addRecursive()` now uses `filepath.Walk()` to monitor all subdirectories
- **Behavior**: Recursively monitors all nested directories automatically

### 5. COS SDK Error Handling
- **Issue**: SDK may return non-EOF errors without proper context
- **Current**: Wraps errors with `fmt.Errorf` for context
- **Testing**: Limited COS integration tests (requires credentials, network)

### 6. Configurable Log Paths (NEW in v1.0.1)
- **Feature**: Applications now support custom log file paths via `log_path` in config.yaml
- **Implementation**: `logger.NewLoggerWithPath()` creates logger at specified path
- **Benefit**: Centralized logging directory, easier maintenance
- **Example**: `log_path: /opt/cos-uploader/logs/cos-uploader.log`

### 7. Version Injection
- **Build**: Compile with `-ldflags="-X main.Version=v1.0.0"`
- **Default**: If not injected, `Version = "dev"` (see `version.go`)
- **CLI**: Use `./cos-uploader --version` to verify

## Testing Strategy

- **Unit Tests**: Each module has `_test.go` with focused tests
- **Integration**: `main.go` tested via full lifecycle with test configs
- **Concurrency**: Run with `go test -race ./...` to catch data races
- **Config Validation**: Tests cover happy path + validation failures
- **Coverage**: Target >80% for core logic (config, logger); lower for optional features

### Adding New Tests

1. Create `*_test.go` in same package
2. Use table-driven tests for multiple scenarios
3. Always test error paths
4. Use `t.TempDir()` for file operations
5. Run with `-race` flag

## Deployment & Operations

### Recommended Directory Structure

```
/opt/cos-uploader/
├── cos-uploader           # Application binary
├── config.yaml            # Configuration file (includes log_path)
└── logs/
    └── cos-uploader.log   # Application logs
```

This structure centralizes all application files for easy management.

### Systemd Service (Linux)
**File**: `cos-uploader.service`

- Type: `simple` (foreground process, no daemonization)
- User: `ubuntu` (update as needed)
- ExecStart: `/opt/cos-uploader/cos-uploader -config /opt/cos-uploader/config.yaml`
- WorkingDirectory: `/opt/cos-uploader`
- Restart: `always` with 10-second backoff
- Logs: Managed by application (see config.yaml `log_path`), also captured by systemd journal
- Installation: `sudo cp cos-uploader.service /etc/systemd/system/` → `sudo systemctl daemon-reload` → `sudo systemctl enable --now cos-uploader`

### GitHub Actions
**File**: `.github/workflows/build-release.yml`

- **Trigger**: Manual workflow dispatch with version input
- **Platforms**: Builds 6 variants (linux/darwin/windows × amd64/arm64)
- **Steps**: Checkout → Setup Go → Test → Build → Archive → Upload Artifacts → Create Release
- **Status**: Runs in ~15-20 minutes for all platforms
- **Gate**: All tests must pass before creating release

### Local Release Building
**File**: `.goreleaser.yml`

- Automates multi-platform builds, checksums, GitHub Release creation
- Use for local testing: `goreleaser release --snapshot --skip-publish --rm-dist`
- Requires git tags: `git tag -a v1.0.0 -m "Release"`

## Key Dependencies

- `github.com/fsnotify/fsnotify` (v1.9.0): File system event monitoring
- `github.com/tencentyun/cos-go-sdk-v5` (v0.7.72): Tencent COS API client
- `go.uber.org/zap` (v1.27.1): Structured logging
- `gopkg.in/yaml.v3` (v3.0.1): YAML parsing

All dependencies should be stable; breaking changes require major version bump.

## Future Enhancements to Consider

1. **Recursive Directory Monitoring**: Currently only watches top-level directories
2. **Event Aggregation**: Batch multiple rapid events to reduce COS load
3. **Metrics Export**: Prometheus metrics for monitoring upload success/failure rates
4. **Configuration Hot-Reload**: Update config without restart
5. **Database Audit Log**: Persistent record of all uploads
6. **Web UI**: Status dashboard and configuration management
7. **Bandwidth Throttling**: Rate-limit uploads to avoid network saturation

## Documentation Files

- `README.md`: User-facing feature overview, quick start, and configuration guide
- `MACOS_BACKGROUND_SETUP.md`: Complete macOS LaunchAgent setup and troubleshooting guide
- `BUILD_GUIDE.md`: Local build instructions, cross-compilation, GoReleaser usage
- `CHANGELOG.md`: Version history, features, bug fixes, and migration guides
- `GITHUB_WORKFLOW_GUIDE.md`: GitHub Actions workflow details
- `RELEASE_WORKFLOW_README.md`: Complete release process guide
- `CLAUDE.md`: This file - development guide and architecture for Claude Code
- `docs/plans/2026-01-20-cos-uploader-implementation.md`: Implementation architecture and task breakdown

### Recent Documentation Updates (v1.0.1)

- **MACOS_BACKGROUND_SETUP.md**: Completely rewritten for `/opt/cos-uploader/` directory structure and log configuration
- **README.md**: Added `log_path` configuration documentation and directory structure guide
- **CLAUDE.md**: Updated with v1.0.1 changes, including eventsChan fix and log configuration
- **CHANGELOG.md**: Created to track version history and migration guides
