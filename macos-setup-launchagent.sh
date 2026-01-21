#!/bin/bash

# macOS LaunchAgent 一键设置脚本
# 用途：让 cos-uploader 在 macOS 后台自动运行

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 获取用户和路径信息
USERNAME=$(whoami)
HOME_DIR=$(eval echo ~$USERNAME)
BIN_DIR="$HOME_DIR/bin"
CONFIG_DIR="$HOME_DIR/.cos-uploader"
CONFIG_FILE="$CONFIG_DIR/config.yaml"
LOG_DIR="$HOME_DIR/Library/Logs/cos-uploader"
LAUNCH_AGENTS_DIR="$HOME_DIR/Library/LaunchAgents"
PLIST_FILE="$LAUNCH_AGENTS_DIR/com.hmw.cos-uploader.plist"
LABEL="com.hmw.cos-uploader"

# 打印标题
echo ""
echo -e "${YELLOW}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${YELLOW}║   macOS LaunchAgent 一键设置 - cos-uploader           ║${NC}"
echo -e "${YELLOW}╚════════════════════════════════════════════════════════╝${NC}"
echo ""

# 检查是否已安装Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}✗ 错误：未检测到 Go 环境${NC}"
    echo "请先安装 Go: https://golang.org/dl"
    exit 1
fi

# 步骤1：编译
echo -e "${YELLOW}[1/6] 编译程序...${NC}"
if go build -o cos-uploader . 2>/dev/null; then
    echo -e "${GREEN}✓ 编译完成${NC}\n"
else
    echo -e "${RED}✗ 编译失败${NC}"
    exit 1
fi

# 步骤2：创建bin目录并移动程序
echo -e "${YELLOW}[2/6] 移动程序到 $BIN_DIR...${NC}"
mkdir -p "$BIN_DIR"
cp cos-uploader "$BIN_DIR/"
chmod +x "$BIN_DIR/cos-uploader"
rm -f cos-uploader
echo -e "${GREEN}✓ 程序已安装${NC}\n"

# 步骤3：检查配置文件
echo -e "${YELLOW}[3/6] 检查配置文件...${NC}"
if [ -f "$CONFIG_FILE" ]; then
    echo -e "${GREEN}✓ 配置文件已存在: $CONFIG_FILE${NC}\n"
else
    echo -e "${YELLOW}⚠ 警告：配置文件不存在${NC}"
    echo "位置应该在: $CONFIG_FILE"
    echo "请先创建配置文件，然后再运行此脚本"
    echo ""
    echo "配置文件示例："
    cat << 'EOF'
projects:
  - name: "my-project"
    directories:
      - "/path/to/watch/dir1"
      - "/path/to/watch/dir2"
    cos_config:
      secret_id: "YOUR_SECRET_ID"
      secret_key: "YOUR_SECRET_KEY"
      bucket: "your-bucket"
      region: "ap-beijing"
      path_prefix: "uploads/"
    watcher:
      pool_size: 3
      events:
        - create
        - write
    alert:
      enabled: false
      dingtalk_webhook: ""
EOF
    exit 1
fi

# 步骤4：创建日志目录
echo -e "${YELLOW}[4/6] 创建日志目录...${NC}"
mkdir -p "$LOG_DIR"
echo -e "${GREEN}✓ 日志目录已创建${NC}\n"

# 步骤5：创建LaunchAgent配置
echo -e "${YELLOW}[5/6] 创建 LaunchAgent 配置...${NC}"
mkdir -p "$LAUNCH_AGENTS_DIR"

cat > "$PLIST_FILE" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$LABEL</string>

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

chmod 644 "$PLIST_FILE"
echo -e "${GREEN}✓ 配置文件已创建${NC}\n"

# 步骤6：加载LaunchAgent
echo -e "${YELLOW}[6/6] 加载 LaunchAgent...${NC}"

# 如果已存在则先卸载
if launchctl list "$LABEL" &>/dev/null 2>&1; then
    echo "检测到已存在的 LaunchAgent，正在卸载..."
    launchctl unload "$PLIST_FILE" 2>/dev/null || true
    sleep 1
fi

# 加载新配置
if launchctl load "$PLIST_FILE" 2>/dev/null; then
    echo -e "${GREEN}✓ LaunchAgent 已加载${NC}\n"
else
    echo -e "${RED}✗ LaunchAgent 加载失败${NC}"
    echo "请检查配置文件内容"
    exit 1
fi

# 验证是否成功加载
sleep 1
if launchctl list "$LABEL" &>/dev/null 2>&1; then
    STATUS_PID=$(launchctl list "$LABEL" 2>/dev/null | head -1 || echo "unknown")
    echo -e "${GREEN}✓ LaunchAgent 已成功启动${NC}"
else
    echo -e "${YELLOW}⚠ LaunchAgent 状态未知${NC}"
fi

# 显示完成信息
echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║            安装完成！                                  ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}配置信息：${NC}"
echo "  程序位置:     $BIN_DIR/cos-uploader"
echo "  配置文件:     $CONFIG_FILE"
echo "  日志目录:     $LOG_DIR"
echo "  Plist文件:    $PLIST_FILE"
echo ""
echo -e "${YELLOW}常用命令：${NC}"
echo "  查看状态:     launchctl list $LABEL"
echo "  查看日志:     tail -f $LOG_DIR/stdout.log"
echo "  查看错误:     tail -f $LOG_DIR/stderr.log"
echo "  重启程序:     launchctl stop $LABEL && launchctl start $LABEL"
echo "  停止程序:     launchctl stop $LABEL"
echo "  启动程序:     launchctl start $LABEL"
echo "  卸载程序:     launchctl unload $PLIST_FILE"
echo ""
echo -e "${GREEN}✓ 程序现在在后台运行，开机时会自动启动${NC}"
echo ""
