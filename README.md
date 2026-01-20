# COS File Monitor & Uploader

A high-performance Go-based file monitoring and automatic upload tool for Tencent Cloud COS (Object Storage Service).

## Features

- **Real-time File Monitoring**: Millisecond-level detection of file changes using fsnotify
- **Multi-project Support**: Configure and manage multiple projects with separate COS buckets
- **Multi-directory Monitoring**: Each project can monitor multiple local directories
- **Concurrent Upload**: Configurable worker pool for parallel file uploads (default: 5 workers)
- **Automatic Retry**: Retry failed uploads up to 3 times with exponential backoff
- **DingTalk Alert**: Send failure notifications via DingTalk robot
- **Comprehensive Logging**: Dual output to stdout and log files with structured JSON format
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
```

## Configuration Parameters

### Project Level
- **name** (required): Project name for identification
- **directories** (required): List of local directories to monitor
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
- **events** (optional): File events to monitor. Options: create, write, remove, rename, chmod. Default: [create, write]
- **pool_size** (optional): Number of concurrent upload workers. Default: 5

### Alert Configuration
- **dingtalk_webhook** (optional): DingTalk robot webhook URL
- **enabled** (optional): Enable/disable alert notifications

## Logs

Logs are output to:
- **stdout**: Real-time console output (INFO level and above)
- **logs/cos-uploader.log**: Persistent file log (DEBUG level and above)

## Architecture

```
File Change Event
      ↓
   fsnotify
      ↓
Event Filtering (by type)
      ↓
Upload Queue (buffered: 100 tasks)
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

- **config**: Configuration management and validation
- **logger**: Structured logging to stdout and file
- **watcher**: File system monitoring using fsnotify
- **uploader**: COS upload engine with worker pool and retry logic
- **alert**: DingTalk notification integration

## System Requirements

- Go 1.21 or higher
- Network access to Tencent Cloud COS API
- (Optional) DingTalk workspace for alerts

## Dependencies

- fsnotify (v1.9.0): File system notifications
- cos-go-sdk-v5 (v0.7.72): Tencent Cloud COS SDK
- zap (v1.27.1): Structured logging
- yaml.v3 (v3.0.1): YAML parsing

## Error Handling

The application includes robust error handling:
- Failed uploads automatically retry up to 3 times
- Failed retries are logged with detailed error messages
- Graceful shutdown on system signals (SIGINT, SIGTERM)
- All operations logged for debugging and monitoring

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

## Troubleshooting

### Files not uploading
1. Check logs: `tail -f logs/cos-uploader.log`
2. Verify COS credentials in config.yaml
3. Ensure monitored directories exist and are accessible
4. Check network connectivity to Tencent Cloud

### High CPU usage
- Reduce pool_size if monitoring large directories with many files
- Increase event filtering if not needed events are being processed

## License

MIT

## Support

For issues, questions, or contributions, please refer to the project repository.
