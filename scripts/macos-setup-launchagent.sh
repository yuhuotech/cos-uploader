#!/bin/bash

# macOS LaunchAgent 一键设置脚本
# 用途：让 cos-uploader 在 macOS 后台自动运行
# 推荐目录结构：/opt/cos-uploader

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 安装目录（推荐使用 /opt/cos-uploader）
INSTALL_DIR="${INSTALL_DIR:-/opt/cos-uploader}"
BINARY_PATH="$INSTALL_DIR/cos-uploader"
CONFIG_FILE="$INSTALL_DIR/config.yaml"
LOG_DIR="$INSTALL_DIR/logs"
LAUNCH_LOG_DIR="$HOME/Library/Logs/cos-uploader"
LAUNCH_AGENTS_DIR="$HOME/Library/LaunchAgents"
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
echo -e "${YELLOW}[1/7] 编译程序...${NC}"
if go build -o cos-uploader . 2>/dev/null; then
    echo -e "${GREEN}✓ 编译完成${NC}\n"
else
    echo -e "${RED}✗ 编译失败${NC}"
    exit 1
fi

# 步骤2：创建安装目录
echo -e "${YELLOW}[2/7] 创建安装目录 $INSTALL_DIR...${NC}"
mkdir -p "$INSTALL_DIR"
mkdir -p "$LOG_DIR"
echo -e "${GREEN}✓ 目录已创建${NC}\n"

# 步骤3：复制程序和配置
echo -e "${YELLOW}[3/7] 复制程序和配置...${NC}"

# 检查配置文件是否存在
if [ ! -f "config.yaml" ]; then
    echo -e "${RED}✗ 错误：当前目录没有 config.yaml${NC}"
    echo "请在项目根目录运行此脚本"
    exit 1
fi

# 检查配置文件中是否有 log_path
if ! grep -q "log_path:" config.yaml; then
    echo -e "${YELLOW}⚠ 警告：config.yaml 中没有 log_path 配置${NC}"
    echo "正在添加 log_path 配置..."
    sed -i '' "1i\\
# 日志文件路径\\
log_path: $LOG_DIR/cos-uploader.log\\
\\
" config.yaml
fi

cp cos-uploader "$BINARY_PATH"
cp config.yaml "$INSTALL_DIR/"
chmod +x "$BINARY_PATH"
rm -f cos-uploader
echo -e "${GREEN}✓ 文件已复制${NC}\n"

# 步骤4：验证配置文件
echo -e "${YELLOW}[4/7] 验证配置文件...${NC}"
if [ -f "$CONFIG_FILE" ]; then
    echo -e "${GREEN}✓ 配置文件检查通过${NC}\n"
else
    echo -e "${RED}✗ 错误：配置文件不存在${NC}"
    exit 1
fi

# 步骤5：创建 LaunchAgent 日志目录
echo -e "${YELLOW}[5/7] 创建 LaunchAgent 日志目录...${NC}"
mkdir -p "$LAUNCH_LOG_DIR"
echo -e "${GREEN}✓ 日志目录已创建${NC}\n"

# 步骤6：创建 LaunchAgent 配置
echo -e "${YELLOW}[6/7] 创建 LaunchAgent 配置...${NC}"
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
        <string>$BINARY_PATH</string>
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
    <string>$LAUNCH_LOG_DIR/stdout.log</string>

    <key>StandardErrorPath</key>
    <string>$LAUNCH_LOG_DIR/stderr.log</string>

    <key>WorkingDirectory</key>
    <string>$INSTALL_DIR</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
</dict>
</plist>
EOF

chmod 644 "$PLIST_FILE"
echo -e "${GREEN}✓ 配置文件已创建${NC}\n"

# 步骤7：加载 LaunchAgent
echo -e "${YELLOW}[7/7] 加载 LaunchAgent...${NC}"

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
echo -e "${YELLOW}安装目录结构：${NC}"
echo "  $INSTALL_DIR/"
echo "  ├── cos-uploader"
echo "  ├── config.yaml"
echo "  └── logs/"
echo ""
echo -e "${YELLOW}配置信息：${NC}"
echo "  二进制文件:   $BINARY_PATH"
echo "  配置文件:     $CONFIG_FILE"
echo "  应用日志:     $LOG_DIR/cos-uploader.log"
echo "  LaunchAgent日志: $LAUNCH_LOG_DIR/"
echo "  Plist文件:    $PLIST_FILE"
echo ""
echo -e "${YELLOW}常用命令：${NC}"
echo "  查看状态:     launchctl list $LABEL"
echo "  查看应用日志: tail -f $LOG_DIR/cos-uploader.log"
echo "  查看系统日志: tail -f $LAUNCH_LOG_DIR/stdout.log"
echo "  查看错误日志: tail -f $LAUNCH_LOG_DIR/stderr.log"
echo "  重启程序:     launchctl stop $LABEL && launchctl start $LABEL"
echo "  停止程序:     launchctl stop $LABEL"
echo "  启动程序:     launchctl start $LABEL"
echo "  卸载程序:     launchctl unload $PLIST_FILE"
echo ""
echo -e "${GREEN}✓ 程序现在在后台运行，开机时会自动启动${NC}"
echo ""
