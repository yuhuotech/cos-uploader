# COS 文件监控上传工具

一个高性能的 Go 应用，实时监控本地文件系统变化并自动上传修改的文件到腾讯云对象存储（COS）。

## ✨ 特性

- **实时文件监控**：使用 fsnotify 实现毫秒级文件变化检测
- **多项目支持**：配置和管理多个项目，每个项目拥有独立的 COS 桶
- **多目录监控**：每个项目可监控多个本地目录
- **递归目录监控**：自动监控所有子目录
- **并发上传**：可配置的工作池实现并行上传（默认 5 个工作线程）
- **自动重试**：失败的上传最多重试 3 次，具备指数退避机制
- **灵活日志配置**：通过配置文件自定义日志文件路径
- **钉钉告警**：上传失败时通过钉钉机器人推送通知
- **结构化日志**：支持同时输出到标准输出和日志文件
- **优雅停机**：正确处理操作系统信号实现干净的应用关闭

## 📦 安装

### 系统要求

- Go 1.21 或更高版本
- 可访问腾讯云 COS API 的网络连接
- （可选）钉钉工作空间用于告警
- （可选）systemd (Linux) 或 LaunchAgent (macOS) 用于后台运行

### 快速安装

```bash
cd cos-uploader
go mod download
go build -o cos-uploader
```

### 从源码编译

更多编译选项参见 [BUILD_GUIDE.md](./docs/BUILD_GUIDE.md)。

## 🚀 快速开始

### 1. 创建配置文件

创建 `config.yaml` 文件：

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

### 2. 启动运行

```bash
# 使用默认的 config.yaml 运行
./cos-uploader

# 使用自定义配置文件
./cos-uploader -config /path/to/config.yaml

# 查看版本
./cos-uploader --version
```

### 3. 查看日志

```bash
# 实时查看日志
tail -f /opt/cos-uploader/logs/cos-uploader.log
```

## 📖 详细配置

### 全局配置

| 配置项 | 说明 | 默认值 | 必需 |
|--------|------|--------|------|
| `log_path` | 日志文件路径（相对或绝对路径） | `logs/cos-uploader.log` | 否 |

### 项目配置

| 配置项 | 说明 | 必需 |
|--------|------|------|
| `name` | 项目名称，用于识别 | 是 |
| `directories` | 要监控的本地目录列表（将递归监控子目录） | 是 |
| `cos` | COS 桶配置 | 是 |
| `watcher` | 文件监控配置 | 是 |
| `alert` | 告警通知配置 | 否 |

### COS 配置

| 配置项 | 说明 | 默认值 | 必需 |
|--------|------|--------|------|
| `secret_id` | 腾讯云 COS API Secret ID | - | 是 |
| `secret_key` | 腾讯云 COS API Secret Key | - | 是 |
| `region` | COS 地域 | `ap-shanghai` | 否 |
| `bucket` | COS 桶名称 | - | 是 |
| `path_prefix` | 远程文件路径前缀 | - | 是 |

### 监控配置

| 配置项 | 说明 | 默认值 | 必需 |
|--------|------|--------|------|
| `events` | 要监控的文件事件 | `[create, write]` | 否 |
| `pool_size` | 并发上传工作线程数 | `5` | 否 |

**支持的事件类型**：`create`、`write`、`remove`、`rename`、`chmod`

### 告警配置

| 配置项 | 说明 | 默认值 | 必需 |
|--------|------|--------|------|
| `dingtalk_webhook` | 钉钉机器人 webhook URL | - | 否 |
| `enabled` | 是否启用告警通知 | `false` | 否 |

## 🔧 使用指南

### 推荐目录结构

```
/opt/cos-uploader/
├── cos-uploader           # 应用程序
├── config.yaml            # 配置文件
└── logs/
    └── cos-uploader.log   # 应用日志
```

此结构将所有应用文件集中在一处，便于管理。

### macOS 部署（LaunchAgent）

完整的 macOS 设置指南请参见 [MACOS_BACKGROUND_SETUP.md](./docs/MACOS_BACKGROUND_SETUP.md)。

快速设置：

```bash
chmod +x setup-macos.sh
./setup-macos.sh
```

### Linux 部署（systemd）

详见 [BUILD_GUIDE.md](./docs/BUILD_GUIDE.md) 中的 Linux/systemd 设置说明。

## 📊 架构说明

```
文件变化事件
      ↓
   fsnotify
      ↓
事件过滤（按类型）
      ↓
上传队列（缓冲：1000 个任务）
      ↓
工作线程池（可配置并发数）
      ↓
COS 上传 API
      ↓
成功 / 重试（最多 3 次）
      ↓
钉钉告警（失败时）
```

### 模块结构

- **config**：配置管理、验证和 YAML 解析
- **logger**：灵活的结构化日志记录，支持输出到标准输出和自定义文件路径
- **watcher**：使用 fsnotify 进行文件系统监控，支持递归目录监控
- **uploader**：COS 上传引擎，包括工作线程池、重试逻辑和完整的上传能力
- **alert**：钉钉通知集成，用于上传失败时的告警
- **main**：应用程序编排、信号处理和生命周期管理

## 📝 日志配置

应用支持灵活的日志配置：

### 应用日志

通过 `config.yaml` 中的 `log_path` 配置：

```yaml
log_path: /opt/cos-uploader/logs/cos-uploader.log
```

- **默认路径**：`logs/cos-uploader.log`（相对于工作目录）
- **绝对路径**：`/opt/cos-uploader/logs/cos-uploader.log`（完整路径）

### 控制台输出

- 实时输出到标准输出（INFO 级别及以上）

### 故障排查日志

在 macOS 的 LaunchAgent 中运行时，可查看以下日志：

- **stdout**：`~/Library/Logs/cos-uploader/stdout.log`
- **stderr**：`~/Library/Logs/cos-uploader/stderr.log`

## 🔍 问题排查

### 文件未上传

1. 查看日志：`tail -f /opt/cos-uploader/logs/cos-uploader.log`
2. 验证 `config.yaml` 中的 COS 凭证
3. 确保监控目录存在且可访问
4. 检查网络连接到腾讯云
5. 验证文件事件与配置的 `events` 列表匹配

### CPU 占用率高

- 减少 `pool_size` 如果监控大型目录时 CPU 占用过高
- 增加事件过滤以排除不需要的事件
- 检查监控目录中是否有过多文件变化

### 应用频繁崩溃或重启

1. 查看应用日志以获取错误信息
2. 检查 LaunchAgent stderr 日志（macOS）：`tail -f ~/Library/Logs/cos-uploader/stderr.log`
3. 验证配置文件语法：`./cos-uploader -config config.yaml`
4. 检查系统资源（磁盘空间、内存）

## 🛠️ 性能优化

### 增加上传并发数

调整监控配置中的 `pool_size`：

```yaml
watcher:
  pool_size: 10  # 增加到 10 个并发上传
```

### 监控多个大型目录

考虑创建多个项目来并行化监控：

```yaml
projects:
  - name: project1
    directories:
      - /data/dir1
  - name: project2
    directories:
      - /data/dir2
```

### 自定义日志位置

```yaml
log_path: /var/log/cos-uploader/app.log
```

## 📄 文档

- [MACOS_BACKGROUND_SETUP.md](./docs/MACOS_BACKGROUND_SETUP.md) - macOS LaunchAgent 设置和配置指南
- [BUILD_GUIDE.md](./docs/BUILD_GUIDE.md) - 从源码编译和跨平台编译指南
- [CLAUDE.md](./CLAUDE.md) - 开发者文档和架构详解
- [CHANGELOG.md](./CHANGELOG.md) - 版本历史和变更记录

## 🔗 关键依赖

- **fsnotify** (v1.9.0)：文件系统事件监控
- **cos-go-sdk-v5** (v0.7.72)：腾讯云 COS API 客户端
- **zap** (v1.27.1)：结构化日志库
- **yaml.v3** (v3.0.1)：YAML 解析库

## 📄 许可证

MIT

## 💬 支持与反馈

如有问题或建议，请参考项目仓库进行反馈。
