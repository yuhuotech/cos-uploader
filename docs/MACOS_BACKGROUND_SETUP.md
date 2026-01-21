# macOS 后台运行指南

在 macOS 上使用 **LaunchAgent** 让 cos-uploader 在后台长期运行。

## 什么是 LaunchAgent？

LaunchAgent 是 macOS 的任务调度系统，用于：
- 在用户登录时自动启动程序
- 程序异常退出时自动重启
- 管理程序的日志输出
- 按需启动和停止

## 目录结构

推荐使用以下标准目录结构，所有文件集中在 `/opt/cos-uploader` 目录下：

```
/opt/cos-uploader/
├── cos-uploader           # 应用二进制文件
├── config.yaml            # 配置文件
└── logs/
    └── cos-uploader.log   # 应用日志
```

这样的结构便于管理和维护，所有文件都在同一个地方。

## 快速安装（推荐）

### 步骤1：编译应用

```bash
cd /path/to/cos-uploader
go build -o cos-uploader .
```

### 步骤2：创建安装目录并复制文件

```bash
# 创建 /opt/cos-uploader 目录
mkdir -p /opt/cos-uploader

# 复制二进制文件
cp cos-uploader /opt/cos-uploader/

# 复制配置文件（修改配置后再复制）
cp config.yaml /opt/cos-uploader/

# 给二进制文件执行权限
chmod +x /opt/cos-uploader/cos-uploader

# 创建日志目录
mkdir -p /opt/cos-uploader/logs
```

### 步骤3：配置文件中添加日志路径

编辑 `/opt/cos-uploader/config.yaml`，在顶部添加：

```yaml
# 日志文件路径（可以使用相对路径或绝对路径）
log_path: /opt/cos-uploader/logs/cos-uploader.log

projects:
    - name: your-project-name
      # ... 其他配置
```

### 步骤4：创建并配置 LaunchAgent

编辑 `~/Library/LaunchAgents/com.hmw.cos-uploader.plist`：

```bash
nano ~/Library/LaunchAgents/com.hmw.cos-uploader.plist
```

粘贴以下内容：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.hmw.cos-uploader</string>

    <key>ProgramArguments</key>
    <array>
        <string>/opt/cos-uploader/cos-uploader</string>
        <string>-config</string>
        <string>/opt/cos-uploader/config.yaml</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>ThrottleInterval</key>
    <integer>10</integer>

    <key>StandardOutPath</key>
    <string>/Users/YOUR_USERNAME/Library/Logs/cos-uploader/stdout.log</string>

    <key>StandardErrorPath</key>
    <string>/Users/YOUR_USERNAME/Library/Logs/cos-uploader/stderr.log</string>

    <key>WorkingDirectory</key>
    <string>/opt/cos-uploader</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
</dict>
</plist>
```

⚠️ **注意**：将 `YOUR_USERNAME` 替换为你的实际用户名。

### 步骤5：创建日志目录并加载 LaunchAgent

```bash
# 创建 LaunchAgent 的标准输出/错误日志目录
mkdir -p ~/Library/Logs/cos-uploader

# 加载 LaunchAgent
launchctl load ~/Library/LaunchAgents/com.hmw.cos-uploader.plist

# 验证加载
launchctl list | grep cos-uploader
```

---

## 配置说明

| 字段 | 说明 |
|------|------|
| Label | 唯一标识，用来管理 LaunchAgent |
| ProgramArguments | 程序路径和启动参数 |
| RunAtLoad | true 表示登录时自动启动 |
| KeepAlive | true 表示异常退出时自动重启 |
| ThrottleInterval | 两次启动之间的最小间隔（秒） |
| StandardOutPath | LaunchAgent 的 stdout 日志位置 |
| StandardErrorPath | LaunchAgent 的 stderr 日志位置 |
| WorkingDirectory | 程序运行目录（应与程序路径同目录） |

---

## 日志配置

应用支持两种日志配置方式：

### 方式1：通过 config.yaml 配置应用日志（推荐）

在 `config.yaml` 顶部添加：

```yaml
log_path: /opt/cos-uploader/logs/cos-uploader.log
```

这样应用的日志会写入到指定路径。如果不配置，默认为 `logs/cos-uploader.log`（相对路径）。

### 方式2：LaunchAgent 的标准输出/错误日志

LaunchAgent 的 `StandardOutPath` 和 `StandardErrorPath` 是独立的日志，用于捕获应用启动过程中的错误或标准输出。建议保留这个配置以便排查问题。

### 查看日志

```bash
# 查看应用日志（config.yaml 中配置的路径）
tail -f /opt/cos-uploader/logs/cos-uploader.log

# 查看 LaunchAgent 的标准输出日志
tail -f ~/Library/Logs/cos-uploader/stdout.log

# 查看 LaunchAgent 的错误日志
tail -f ~/Library/Logs/cos-uploader/stderr.log
```

---

## 常用命令

```bash
# 启动程序
launchctl start com.hmw.cos-uploader

# 停止程序
launchctl stop com.hmw.cos-uploader

# 重启程序
launchctl stop com.hmw.cos-uploader && launchctl start com.hmw.cos-uploader

# 查看运行状态
launchctl list com.hmw.cos-uploader

# 重新加载配置（修改 config.yaml 后）
launchctl unload ~/Library/LaunchAgents/com.hmw.cos-uploader.plist
launchctl load ~/Library/LaunchAgents/com.hmw.cos-uploader.plist

# 卸载 LaunchAgent
launchctl unload ~/Library/LaunchAgents/com.hmw.cos-uploader.plist

# 查看进程
ps aux | grep cos-uploader | grep -v grep
```

---

## 快速设置脚本

保存为 `setup-macos.sh`：

```bash
#!/bin/bash

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${YELLOW}=== COS Uploader macOS 安装脚本 ===${NC}\n"

# 1. 编译
echo -e "${YELLOW}[1/6] 编译程序...${NC}"
go build -o cos-uploader . || { echo -e "${RED}编译失败${NC}"; exit 1; }
echo -e "${GREEN}✓ 编译完成${NC}\n"

# 2. 创建安装目录
echo -e "${YELLOW}[2/6] 创建 /opt/cos-uploader 目录...${NC}"
mkdir -p /opt/cos-uploader
mkdir -p /opt/cos-uploader/logs
echo -e "${GREEN}✓ 目录创建完成${NC}\n"

# 3. 复制文件
echo -e "${YELLOW}[3/6] 复制文件...${NC}"
cp cos-uploader /opt/cos-uploader/
cp config.yaml /opt/cos-uploader/
chmod +x /opt/cos-uploader/cos-uploader
echo -e "${GREEN}✓ 文件复制完成${NC}\n"

# 4. 创建 LaunchAgent 配置
echo -e "${YELLOW}[4/6] 创建 LaunchAgent 配置...${NC}"
USERNAME=$(whoami)
mkdir -p ~/Library/LaunchAgents
mkdir -p ~/Library/Logs/cos-uploader

cat > ~/Library/LaunchAgents/com.hmw.cos-uploader.plist << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.hmw.cos-uploader</string>

    <key>ProgramArguments</key>
    <array>
        <string>/opt/cos-uploader/cos-uploader</string>
        <string>-config</string>
        <string>/opt/cos-uploader/config.yaml</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>ThrottleInterval</key>
    <integer>10</integer>

    <key>StandardOutPath</key>
    <string>\$HOME/Library/Logs/cos-uploader/stdout.log</string>

    <key>StandardErrorPath</key>
    <string>\$HOME/Library/Logs/cos-uploader/stderr.log</string>

    <key>WorkingDirectory</key>
    <string>/opt/cos-uploader</string>
</dict>
</plist>
EOF

echo -e "${GREEN}✓ LaunchAgent 配置已创建${NC}\n"

# 5. 加载 LaunchAgent
echo -e "${YELLOW}[5/6] 加载 LaunchAgent...${NC}"

# 如果已存在则先卸载
if launchctl list com.hmw.cos-uploader &>/dev/null; then
    echo "检测到已存在的 LaunchAgent，正在卸载..."
    launchctl unload ~/Library/LaunchAgents/com.hmw.cos-uploader.plist 2>/dev/null || true
    sleep 1
fi

launchctl load ~/Library/LaunchAgents/com.hmw.cos-uploader.plist

if launchctl list com.hmw.cos-uploader &>/dev/null; then
    echo -e "${GREEN}✓ LaunchAgent 已加载${NC}\n"
else
    echo -e "${RED}✗ LaunchAgent 加载失败${NC}\n"
    exit 1
fi

# 6. 验证
echo -e "${YELLOW}[6/6] 验证安装...${NC}"
sleep 2
if ps aux | grep -v grep | grep "/opt/cos-uploader/cos-uploader" > /dev/null; then
    echo -e "${GREEN}✓ 应用已启动${NC}\n"
else
    echo -e "${YELLOW}⚠ 应用未检测到，请检查日志${NC}\n"
fi

# 显示完成信息
echo -e "${GREEN}=== 安装完成 ===${NC}"
echo ""
echo "应用信息："
echo "  二进制文件: /opt/cos-uploader/cos-uploader"
echo "  配置文件:   /opt/cos-uploader/config.yaml"
echo "  应用日志:   /opt/cos-uploader/logs/cos-uploader.log"
echo "  LaunchAgent 日志: ~/Library/Logs/cos-uploader/"
echo ""
echo "常用命令："
echo "  启动:     launchctl start com.hmw.cos-uploader"
echo "  停止:     launchctl stop com.hmw.cos-uploader"
echo "  查看状态: launchctl list com.hmw.cos-uploader"
echo "  查看日志: tail -f /opt/cos-uploader/logs/cos-uploader.log"
echo "  重启:     launchctl stop com.hmw.cos-uploader && launchctl start com.hmw.cos-uploader"
echo ""
echo -e "${GREEN}程序已在后台运行！${NC}"
```

使用脚本：

```bash
chmod +x setup-macos.sh
./setup-macos.sh
```

---

## 监控运行状态

```bash
# 查看进程
ps aux | grep cos-uploader | grep -v grep

# 查看 LaunchAgent 状态
launchctl list com.hmw.cos-uploader

# 实时查看应用日志
tail -f /opt/cos-uploader/logs/cos-uploader.log

# 查看最近 50 行日志
tail -50 /opt/cos-uploader/logs/cos-uploader.log
```

---

## 故障排除

### 问题1：应用无法启动

**症状**：`launchctl list` 显示非零状态或应用经常重启

**排查步骤**：

```bash
# 1. 查看 LaunchAgent 的错误日志
tail -f ~/Library/Logs/cos-uploader/stderr.log

# 2. 手动运行应用测试
/opt/cos-uploader/cos-uploader -config /opt/cos-uploader/config.yaml

# 3. 检查配置文件
cat /opt/cos-uploader/config.yaml

# 4. 查看应用日志
tail -f /opt/cos-uploader/logs/cos-uploader.log
```

### 问题2：配置文件路径不正确

**错误**：`Failed to load config`

**解决**：
```bash
# 确保配置文件存在
ls -la /opt/cos-uploader/config.yaml

# 确保路径在 plist 文件中正确
cat ~/Library/LaunchAgents/com.hmw.cos-uploader.plist | grep config.yaml
```

### 问题3：日志文件权限问题

**错误**：`Permission denied` 写入日志

**解决**：
```bash
# 检查目录权限
ls -la /opt/cos-uploader/logs/

# 重新创建目录并设置正确权限
mkdir -p /opt/cos-uploader/logs
chmod 755 /opt/cos-uploader/logs
```

### 问题4：应用频繁重启

**症状**：应用每隔几秒就重启一次

**原因**：通常是由于应用崩溃或配置错误

**排查**：
```bash
# 1. 停止应用
launchctl stop com.hmw.cos-uploader

# 2. 查看错误日志
tail -100 ~/Library/Logs/cos-uploader/stderr.log

# 3. 手动运行查看详细错误
/opt/cos-uploader/cos-uploader -config /opt/cos-uploader/config.yaml

# 4. 修复问题后重启
launchctl start com.hmw.cos-uploader
```

---

## 完整工作流示例

```bash
# 1. 在项目目录中
cd /path/to/cos-uploader

# 2. 编辑配置文件，确保 log_path 已配置
nano config.yaml

# 3. 运行安装脚本
chmod +x setup-macos.sh
./setup-macos.sh

# 4. 验证安装成功
launchctl list com.hmw.cos-uploader
ps aux | grep cos-uploader

# 5. 查看实时日志
tail -f /opt/cos-uploader/logs/cos-uploader.log

# 程序现在会在后台运行，开机自启
```

---

## 卸载

```bash
# 停止程序
launchctl stop com.hmw.cos-uploader

# 卸载 LaunchAgent
launchctl unload ~/Library/LaunchAgents/com.hmw.cos-uploader.plist

# 删除配置文件
rm ~/Library/LaunchAgents/com.hmw.cos-uploader.plist

# 删除应用文件（可选）
rm -rf /opt/cos-uploader

# 删除日志（可选）
rm -rf ~/Library/Logs/cos-uploader
```

---

## 开机自启

LaunchAgent 已默认配置 `RunAtLoad=true`，所以：
- ✅ 用户登录时自动启动
- ✅ 程序意外退出时自动重启
- ✅ macOS 重启后会自动启动（用户需要登录）

---

## 与 systemd 对比（Linux 用户参考）

macOS LaunchAgent ≈ Linux systemd

| 功能 | LaunchAgent | systemd |
|------|------------|---------|
| 配置文件位置 | ~/Library/LaunchAgents/ | /etc/systemd/system/ |
| 启动命令 | launchctl start | systemctl start |
| 查看状态 | launchctl list | systemctl status |
| 自动重启 | KeepAlive | Restart=always |
| 开机自启 | RunAtLoad | enable |
