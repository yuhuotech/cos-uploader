#!/bin/bash

# 交叉编译脚本 - 在macOS上编译所有平台的二进制文件

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 获取版本号
VERSION=${1:-v1.0.0}

echo -e "${YELLOW}=== COS Uploader 交叉编译脚本 ===${NC}\n"
echo "版本: $VERSION"
echo "编译目录: ./release"
echo ""

# 创建release目录
mkdir -p release

# 定义编译目标
TARGETS=(
  "linux:amd64:linux_amd64"
  "linux:arm64:linux_arm64"
  "windows:amd64:windows_amd64"
  "windows:arm64:windows_arm64"
  "darwin:amd64:darwin_amd64"
  "darwin:arm64:darwin_arm64"
)

# 编译统计
TOTAL=${#TARGETS[@]}
SUCCESSFUL=0
FAILED=0

# 编译所有目标
for TARGET in "${TARGETS[@]}"; do
  IFS=':' read -r OS ARCH NAME <<< "$TARGET"

  # 确定输出文件名
  if [ "$OS" == "windows" ]; then
    OUTPUT="release/cos-uploader-${VERSION}-${NAME}.exe"
  else
    OUTPUT="release/cos-uploader-${VERSION}-${NAME}"
  fi

  echo -ne "${YELLOW}[编译中]${NC} $OS/$ARCH ... "

  # 执行编译
  if GOOS=$OS GOARCH=$ARCH go build \
    -ldflags="-s -w -X main.Version=$VERSION" \
    -o "$OUTPUT" . 2>/dev/null; then

    # 获取文件大小
    SIZE=$(du -h "$OUTPUT" | cut -f1)
    echo -e "${GREEN}✓${NC} ($SIZE)"
    ((SUCCESSFUL++))
  else
    echo -e "${RED}✗${NC}"
    ((FAILED++))
  fi
done

echo ""
echo -e "${YELLOW}=== 编译完成 ===${NC}"
echo "总计: $TOTAL | ${GREEN}成功: $SUCCESSFUL${NC} | ${RED}失败: $FAILED${NC}"
echo ""

# 列出编译结果
echo -e "${YELLOW}=== 编译结果 ===${NC}"
ls -lh release/cos-uploader-${VERSION}-*

# 可选：创建压缩包
echo ""
echo -e "${YELLOW}=== 创建压缩包 ===${NC}"

cd release

for TARGET in "${TARGETS[@]}"; do
  IFS=':' read -r OS ARCH NAME <<< "$TARGET"

  BINARY="cos-uploader-${VERSION}-${NAME}"

  if [ "$OS" == "windows" ]; then
    # Windows使用zip
    ARCHIVE="${BINARY}.zip"
    echo -ne "压缩中: $ARCHIVE ... "
    zip -q "$ARCHIVE" "$BINARY.exe" && echo -e "${GREEN}✓${NC}" || echo -e "${RED}✗${NC}"
  else
    # Linux/macOS使用tar.gz
    ARCHIVE="${BINARY}.tar.gz"
    echo -ne "压缩中: $ARCHIVE ... "
    tar -czf "$ARCHIVE" "$BINARY" && echo -e "${GREEN}✓${NC}" || echo -e "${RED}✗${NC}"
  fi
done

echo ""
echo -e "${GREEN}✓ 编译完成！所有文件在 ./release 目录中${NC}"
