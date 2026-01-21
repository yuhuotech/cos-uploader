# macOS 后台运行指南

在 macOS 上使用 **LaunchAgent** 让 cos-uploader 在后台长期运行。

## 什么是 LaunchAgent？

LaunchAgent 是 macOS 的任务调度系统，用于：
- 在用户登录时自动启动程序
- 程序异常退出时自动重启
- 管理程序的日志输出
- 按需启动和停止

## 步骤1：编译应用

```bash
go build -o cos-uploader .
```

## 步骤2：将可执行文件移到合适的位置

```bash
# 方案A：放在用户目录（推荐）
mkdir -p ~/bin
mv cos-uploader ~/bin/

# 或方案B：放在 /usr/local/bin（系统范围）
sudo mv cos-uploader /usr/local/bin/
```

## 步骤3：创建配置文件

假设你的配置文件在 `~/.cos-uploader/config.yaml`，创建 LaunchAgent 配置文件：

```bash
# 使用你喜欢的编辑器创建这个文件
nano ~/Library/LaunchAgents/com.hmw.cos-uploader.plist
```

配置文件内容（用户级，放在~/bin）：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <!-- 唯一标识符，用来管理这个LaunchAgent -->
    <key>Label</key>
    <string>com.hmw.cos-uploader</string>

    <!-- 程序路径 -->
    <key>ProgramArguments</key>
    <array>
        <string>/Users/YOUR_USERNAME/bin/cos-uploader</string>
        <string>-config</string>
        <string>/Users/YOUR_USERNAME/.cos-uploader/config.yaml</string>
    </array>

    <!-- 在用户登录时启动 -->
    <key>RunAtLoad</key>
    <true/>

    <!-- 程序异常退出时自动重启 -->
    <key>KeepAlive</key>
    <true/>

    <!-- 重启间隔（秒） -->
    <key>ThrottleInterval</key>
    <integer>10</integer>

    <!-- 标准输出日志 -->
    <key>StandardOutPath</key>
    <string>/Users/YOUR_USERNAME/Library/Logs/cos-uploader/stdout.log</string>

    <!-- 标准错误日志 -->
    <key>StandardErrorPath</key>
    <string>/Users/YOUR_USERNAME/Library/Logs/cos-uploader/stderr.log</string>

    <!-- 工作目录 -->
    <key>WorkingDirectory</key>
    <string>/Users/YOUR_USERNAME</string>

    <!-- 环境变量（可选） -->
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</key>
    </dict>
</dict>
</plist>
```

### 配置说明

| 字段 | 说明 |
|------|------|
| Label | 唯一标识，用来管理LaunchAgent |
| ProgramArguments | 程序路径和参数 |
| RunAtLoad | true表示登录时自动启动 |
| KeepAlive | true表示异常退出时自动重启 |
| ThrottleInterval | 两次启动之间的最小间隔（秒） |
| StandardOutPath | stdout输出日志位置 |
| StandardErrorPath | stderr输出日志位置 |
| WorkingDirectory | 程序运行目录 |

## 步骤4：创建日志目录

```bash
mkdir -p ~/Library/Logs/cos-uploader
```

## 步骤5：加载 LaunchAgent

```bash
# 加载并立即启动
launchctl load ~/Library/LaunchAgents/com.hmw.cos-uploader.plist

# 验证是否已加载
launchctl list | grep cos-uploader
```

## 常用命令

```bash
# 启动程序
launchctl start com.hmw.cos-uploader

# 停止程序
launchctl stop com.hmw.cos-uploader

# 重启程序
launchctl stop com.hmw.cos-uploader
launchctl start com.hmw.cos-uploader

# 卸载（停止并移除）
launchctl unload ~/Library/LaunchAgents/com.hmw.cos-uploader.plist

# 重新加载配置
launchctl unload ~/Library/LaunchAgents/com.hmw.cos-uploader.plist
launchctl load ~/Library/LaunchAgents/com.hmw.cos-uploader.plist

# 查看运行状态
launchctl list com.hmw.cos-uploader

# 查看日志
tail -f ~/Library/Logs/cos-uploader/stdout.log
tail -f ~/Library/Logs/cos-uploader/stderr.log
```

## 快速设置脚本

保存为 `setup-launchagent.sh`：

```bash
#!/bin/bash

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

USERNAME=$(whoami)
HOME_DIR=$(eval echo ~$USERNAME)
BIN_DIR="$HOME_DIR/bin"
CONFIG_FILE="$HOME_DIR/.cos-uploader/config.yaml"
LOG_DIR="$HOME_DIR/Library/Logs/cos-uploader"
PLIST_FILE="$HOME_DIR/Library/LaunchAgents/com.hmw.cos-uploader.plist"

echo -e "${YELLOW}=== macOS LaunchAgent 设置脚本 ===${NC}\n"

# 1. 编译
echo -e "${YELLOW}[1/5] 编译程序...${NC}"
go build -o cos-uploader . || { echo -e "${RED}编译失败${NC}"; exit 1; }
echo -e "${GREEN}✓ 编译完成${NC}\n"

# 2. 创建bin目录并移动程序
echo -e "${YELLOW}[2/5] 移动程序到 ~/bin...${NC}"
mkdir -p "$BIN_DIR"
mv cos-uploader "$BIN_DIR/"
chmod +x "$BIN_DIR/cos-uploader"
echo -e "${GREEN}✓ 程序已移动到 $BIN_DIR${NC}\n"

# 3. 创建日志目录
echo -e "${YELLOW}[3/5] 创建日志目录...${NC}"
mkdir -p "$LOG_DIR"
echo -e "${GREEN}✓ 日志目录已创建${NC}\n"

# 4. 创建LaunchAgent配置
echo -e "${YELLOW}[4/5] 创建LaunchAgent配置...${NC}"
mkdir -p "$HOME_DIR/Library/LaunchAgents"

cat > "$PLIST_FILE" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.hmw.cos-uploader</string>

    <key>ProgramArguments</key>
    <array>
        <string>$BIN_DIR/cos-uploader</string>
        <string>-config</string>
        <string>$CONFIG_FILE</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>ThrottleInterval</key>
    <integer>10</integer>

    <key>StandardOutPath</key>
    <string>$LOG_DIR/stdout.log</string>

    <key>StandardErrorPath</key>
    <string>$LOG_DIR/stderr.log</string>

    <key>WorkingDirectory</key>
    <string>$HOME_DIR</string>
</dict>
</plist>
EOF

echo -e "${GREEN}✓ 配置文件已创建${NC}\n"

# 5. 加载LaunchAgent
echo -e "${YELLOW}[5/5] 加载LaunchAgent...${NC}"

# 如果已存在则先卸载
if launchctl list com.hmw.cos-uploader &>/dev/null; then
    echo "检测到已存在的LaunchAgent，正在卸载..."
    launchctl unload "$PLIST_FILE" 2>/dev/null || true
fi

launchctl load "$PLIST_FILE"

if launchctl list com.hmw.cos-uploader &>/dev/null; then
    echo -e "${GREEN}✓ LaunchAgent已加载${NC}\n"
else
    echo -e "${RED}✗ LaunchAgent加载失败${NC}\n"
    exit 1
fi

# 显示完成信息
echo -e "${GREEN}=== 设置完成 ===${NC}"
echo ""
echo "配置信息："
echo "  程序位置: $BIN_DIR/cos-uploader"
echo "  配置文件: $CONFIG_FILE"
echo "  日志目录: $LOG_DIR"
echo "  Plist文件: $PLIST_FILE"
echo ""
echo "常用命令："
echo "  启动:    launchctl start com.hmw.cos-uploader"
echo "  停止:    launchctl stop com.hmw.cos-uploader"
echo "  查看日志: tail -f $LOG_DIR/stdout.log"
echo "  卸载:    launchctl unload $PLIST_FILE"
echo ""
echo -e "${GREEN}程序已启动，可以通过以下命令查看日志:${NC}"
echo "  tail -f $LOG_DIR/stdout.log"
```

## 使用快速设置脚本

```bash
# 下载或复制脚本
chmod +x setup-launchagent.sh

# 运行脚本
./setup-launchagent.sh
```

## 查看日志

```bash
# 查看stdout日志
tail -f ~/Library/Logs/cos-uploader/stdout.log

# 查看stderr日志
tail -f ~/Library/Logs/cos-uploader/stderr.log

# 同时查看两个日志
tail -f ~/Library/Logs/cos-uploader/*.log
```

## 监控运行状态

```bash
# 查看进程是否运行
ps aux | grep cos-uploader

# 查看LaunchAgent详细信息
launchctl list com.hmw.cos-uploader

# 查看是否自动重启
launchctl list | grep -i cos
```

## 故障排除

### 问题1：配置文件路径不正确

**错误信息**: `Failed to load config`

**解决方案**:
```bash
# 确保配置文件存在
ls -la ~/.cos-uploader/config.yaml

# 在plist中使用完整路径
# 不要使用 ~，用完整路径
```

### 问题2：程序一启动就退出

**症状**: `launchctl list` 显示 `- 0`

**解决方案**:
```bash
# 查看错误日志
tail -f ~/Library/Logs/cos-uploader/stderr.log

# 手动运行测试
~/bin/cos-uploader -config ~/.cos-uploader/config.yaml

# 检查权限
chmod +x ~/bin/cos-uploader
```

### 问题3：日志目录权限问题

**错误**: Permission denied

**解决方案**:
```bash
# 重新创建日志目录并设置权限
mkdir -p ~/Library/Logs/cos-uploader
chmod 755 ~/Library/Logs/cos-uploader
```

### 问题4：修改配置后需要重启

```bash
# 重新加载
launchctl unload ~/Library/LaunchAgents/com.hmw.cos-uploader.plist
launchctl load ~/Library/LaunchAgents/com.hmw.cos-uploader.plist

# 验证
launchctl start com.hmw.cos-uploader
```

## 开机自启

LaunchAgent 已默认配置 `RunAtLoad=true`，所以：
- ✅ 用户登录时自动启动
- ✅ 程序意外退出时自动重启
- ✅ macOS重启后会自动启动

## 卸载

```bash
# 停止程序
launchctl stop com.hmw.cos-uploader

# 卸载LaunchAgent
launchctl unload ~/Library/LaunchAgents/com.hmw.cos-uploader.plist

# 删除配置文件
rm ~/Library/LaunchAgents/com.hmw.cos-uploader.plist

# 删除日志（可选）
rm -rf ~/Library/Logs/cos-uploader
```

## 完整工作流示例

```bash
# 1. 编译
go build -o cos-uploader .

# 2. 配置COS信息
nano ~/.cos-uploader/config.yaml

# 3. 运行快速设置脚本
./setup-launchagent.sh

# 4. 验证运行
launchctl list | grep cos-uploader

# 5. 查看日志
tail -f ~/Library/Logs/cos-uploader/stdout.log

# 程序现在会在后台运行，开机自启
```

## 与systemd对比（Linux用户参考）

macOS LaunchAgent ≈ Linux systemd

| 功能 | LaunchAgent | systemd |
|------|------------|---------|
| 配置文件位置 | ~/Library/LaunchAgents/ | /etc/systemd/system/ |
| 启动命令 | launchctl load | systemctl start |
| 查看状态 | launchctl list | systemctl status |
| 自动重启 | KeepAlive | Restart=always |
| 开机自启 | RunAtLoad | enable |

