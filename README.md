# COS File Monitor & Uploader

A high-performance Go-based file monitoring and automatic upload tool for Tencent Cloud COS (Object Storage Service).

## Features

- **Real-time File Monitoring**: Millisecond-level detection of file changes using fsnotify
- **Multi-project Support**: Configure and manage multiple projects with separate COS buckets
- **Multi-directory Monitoring**: Each project can monitor multiple local directories
- **Recursive Directory Monitoring**: Automatically monitors all subdirectories
- **Concurrent Upload**: Configurable worker pool for parallel file uploads (default: 5 workers)
- **Automatic Retry**: Retry failed uploads up to 3 times with exponential backoff
- **Configurable Logging**: Customizable log file path via configuration
- **DingTalk Alert**: Send failure notifications via DingTalk robot
- **Comprehensive Logging**: Dual output to stdout and log files with structured format
- **Graceful Shutdown**: Handle OS signals for clean application termination

## Quick Start

### Installation

```bash
cd cos-uploader
go mod download
go build -o cos-uploader
```

### Configuration

Create a `config.yaml` file:

```yaml
# 日志文件路径（可选，不配置时默认为 logs/cos-uploader.log）
log_path: /opt/cos-uploader/logs/cos-uploader.log

projects:
  - name: project1
    directories:
      - /path/to/local/dir1
      - /path/to/local/dir2
    cos:
      secret_id: your-secret-id
      secret_key: your-secret-key
      region: ap-shanghai
      bucket: my-bucket
      path_prefix: uploads/project1/
    watcher:
      events:
        - create
        - write
      pool_size: 5
    alert:
      dingtalk_webhook: https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN
      enabled: true

  - name: project2
    directories:
      - /path/to/another/dir
    cos:
      secret_id: your-secret-id
      secret_key: your-secret-key
      region: ap-shanghai
      bucket: my-bucket
      path_prefix: uploads/project2/
    watcher:
      events:
        - create
        - write
        - remove
      pool_size: 3
    alert:
      dingtalk_webhook: https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN
      enabled: true
```

### Usage

```bash
# Run with default config.yaml
./cos-uploader

# Run with custom config file
./cos-uploader -config /path/to/config.yaml

# Display version
./cos-uploader --version
```

## Configuration Parameters

### Global Level
- **log_path** (optional): Path to the log file. Can be relative or absolute path. Default: `logs/cos-uploader.log`

### Project Level
- **name** (required): Project name for identification
- **directories** (required): List of local directories to monitor (will recursively watch subdirectories)
- **cos**: COS bucket configuration
- **watcher**: File monitoring configuration
- **alert**: Alert notification configuration

### COS Configuration
- **secret_id** (required): Tencent Cloud COS API Secret ID
- **secret_key** (required): Tencent Cloud COS API Secret Key
- **region** (optional): COS region, default: ap-shanghai
- **bucket** (required): COS bucket name
- **path_prefix** (required): Path prefix for remote files

### Watcher Configuration
- **events** (optional): File events to monitor. Options: `create`, `write`, `remove`, `rename`, `chmod`. Default: `[create, write]`
- **pool_size** (optional): Number of concurrent upload workers. Default: 5

### Alert Configuration
- **dingtalk_webhook** (optional): DingTalk robot webhook URL
- **enabled** (optional): Enable/disable alert notifications. Default: false

## Logging

The application supports flexible logging configuration:

### Application Logs
Configured via `log_path` in `config.yaml`:
```yaml
log_path: /opt/cos-uploader/logs/cos-uploader.log
```

- **Default path**: `logs/cos-uploader.log` (relative to working directory)
- **Absolute path**: `/opt/cos-uploader/logs/cos-uploader.log` (full path)
- **Supports environment paths**: `~/logs/app.log` (will be expanded)

### Console Output
- Real-time output to stdout (INFO level and above)

### Troubleshooting Logs
When running with LaunchAgent on macOS, additional logs are available:
- **stdout**: `~/Library/Logs/cos-uploader/stdout.log`
- **stderr**: `~/Library/Logs/cos-uploader/stderr.log`

## Directory Structure (Recommended)

```
/opt/cos-uploader/
├── cos-uploader           # Application binary
├── config.yaml            # Configuration file
└── logs/
    └── cos-uploader.log   # Application logs
```

This structure keeps all application files in one place for easy management.

## macOS Installation

For a complete setup guide on macOS with LaunchAgent, see [MACOS_BACKGROUND_SETUP.md](./MACOS_BACKGROUND_SETUP.md).

Quick setup:
```bash
chmod +x setup-macos.sh
./setup-macos.sh
```

## Linux Installation

See [BUILD_GUIDE.md](./BUILD_GUIDE.md) for Linux/systemd setup instructions.

## Architecture

```
File Change Event
      ↓
   fsnotify
      ↓
Event Filtering (by type)
      ↓
Upload Queue (buffered: 1000 tasks)
      ↓
Worker Pool (configurable concurrent workers)
      ↓
COS Upload API
      ↓
Success / Retry (up to 3 times)
      ↓
DingTalk Alert (on failure)
```

## Module Structure

- **config**: Configuration management, validation, and YAML parsing
- **logger**: Flexible structured logging to stdout and configurable file paths
- **watcher**: File system monitoring using fsnotify with recursive directory support
- **uploader**: COS upload engine with worker pool, retry logic, and full upload capability
- **alert**: DingTalk notification integration for upload failures
- **main**: Application orchestration, signal handling, and lifecycle management

## System Requirements

- Go 1.21 or higher
- Network access to Tencent Cloud COS API
- (Optional) DingTalk workspace for alerts
- (Optional) systemd (Linux) or LaunchAgent (macOS) for background running

## Dependencies

- **fsnotify** (v1.9.0): File system notifications
- **cos-go-sdk-v5** (v0.7.72): Tencent Cloud COS SDK
- **zap** (v1.27.1): Structured logging (used for logger foundation)
- **yaml.v3** (v3.0.1): YAML parsing

## Error Handling

The application includes robust error handling:
- Failed uploads automatically retry up to 3 times
- Failed retries are logged with detailed error messages
- Graceful shutdown on system signals (SIGINT, SIGTERM)
- All operations logged for debugging and monitoring
- Automatic recovery from temporary network failures

## Performance Tuning

### Increasing Upload Concurrency
Adjust `pool_size` in watcher configuration:
```yaml
watcher:
  pool_size: 10  # Increase to 10 concurrent uploads
```

### Monitoring Multiple Large Directories
Consider creating separate projects to parallelize monitoring:
```yaml
projects:
  - name: project1
    directories:
      - /data/dir1
  - name: project2
    directories:
      - /data/dir2
```

### Customizing Log Location
```yaml
log_path: /var/log/cos-uploader/app.log
```

## Troubleshooting

### Files not uploading
1. Check logs: `tail -f /opt/cos-uploader/logs/cos-uploader.log`
2. Verify COS credentials in `config.yaml`
3. Ensure monitored directories exist and are accessible
4. Check network connectivity to Tencent Cloud
5. Verify file events match the configured `events` list

### High CPU usage
- Reduce `pool_size` if monitoring large directories with many files
- Increase event filtering if not needed events are being processed
- Check for excessive file churn in monitored directories

### Application crashes or frequent restarts
1. Review application logs for errors
2. Check LaunchAgent stderr logs (macOS): `tail -f ~/Library/Logs/cos-uploader/stderr.log`
3. Verify configuration file syntax with: `./cos-uploader -config config.yaml`
4. Check system resources (disk space, memory)

## Documentation

- [MACOS_BACKGROUND_SETUP.md](./MACOS_BACKGROUND_SETUP.md) - macOS LaunchAgent setup and configuration
- [BUILD_GUIDE.md](./BUILD_GUIDE.md) - Building from source and cross-compilation
- [CLAUDE.md](./CLAUDE.md) - Development guide and architecture details

## License

MIT

## Support

For issues, questions, or contributions, please refer to the project repository.
