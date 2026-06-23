#!/bin/bash
# 多平台打包脚本
# 用法: ./build_dist.sh [版本号]
# 输出: dist/ 目录下各平台 .tar.gz 压缩包

set -e

VERSION="${1:-$(date +%Y%m%d)}"
BINARY_NAME="ctf-agent"
DIST_DIR="dist"
MODULE_NAME=$(grep '^module' go.mod | awk '{print $2}')

# 打包内容（除二进制外的静态文件）
ASSETS=(
    "config.example.yaml"
    "system_prompt.txt"
    "doc"
    "readme.md"
)

# 目标平台: OS/ARCH
TARGETS=(
    "linux/amd64"
    "linux/arm64"
    "linux/386"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

echo "========================================"
echo " CTF Agent 多平台打包"
echo " 版本: ${VERSION}"
echo " 模块: ${MODULE_NAME}"
echo "========================================"
echo

# 清理并创建 dist
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

BUILD_FLAGS="-ldflags=-s -w -X main.version=${VERSION}"
FAILED=()
SUCCESS=()

for target in "${TARGETS[@]}"; do
    OS="${target%%/*}"
    ARCH="${target##*/}"
    PKG_NAME="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}"
    STAGE_DIR="${DIST_DIR}/_stage_${OS}_${ARCH}"

    printf "构建 %-20s ... " "${OS}/${ARCH}"

    # 编译
    EXT=""
    if [ "${OS}" = "windows" ]; then
        EXT=".exe"
    fi
    OUT_BIN="${STAGE_DIR}/${BINARY_NAME}${EXT}"

    mkdir -p "${STAGE_DIR}"

    if ! GOOS="${OS}" GOARCH="${ARCH}" go build \
        -ldflags="-s -w -X main.version=${VERSION}" \
        -o "${OUT_BIN}" . 2>/dev/null; then
        echo "✗ 编译失败"
        FAILED+=("${OS}/${ARCH}")
        rm -rf "${STAGE_DIR}"
        continue
    fi

    # 复制静态文件
    for asset in "${ASSETS[@]}"; do
        if [ -e "${asset}" ]; then
            cp -r "${asset}" "${STAGE_DIR}/"
        fi
    done

    # 打包
    if [ "${OS}" = "windows" ]; then
        OUT_FILE="${DIST_DIR}/${PKG_NAME}.zip"
        (cd "${DIST_DIR}" && zip -qr "../${OUT_FILE}" "_stage_${OS}_${ARCH}")
        # zip 不支持相对路径改名，重新打包更清晰
        rm -f "${OUT_FILE}"
        OUT_FILE="${DIST_DIR}/${PKG_NAME}.zip"
        (cd "${STAGE_DIR}" && zip -qr "../../${OUT_FILE}" .)
    else
        OUT_FILE="${DIST_DIR}/${PKG_NAME}.tar.gz"
        tar -czf "${OUT_FILE}" -C "${STAGE_DIR}" .
    fi

    SIZE=$(du -sh "${OUT_FILE}" | cut -f1)
    echo "✓  ${SIZE}  →  $(basename "${OUT_FILE}")"
    SUCCESS+=("${OS}/${ARCH}")

    rm -rf "${STAGE_DIR}"
done

echo
echo "========================================"
echo " 结果"
echo "========================================"

if [ ${#SUCCESS[@]} -gt 0 ]; then
    echo "成功 (${#SUCCESS[@]}):"
    for t in "${SUCCESS[@]}"; do printf "  ✓ %s\n" "${t}"; done
fi

if [ ${#FAILED[@]} -gt 0 ]; then
    echo "失败 (${#FAILED[@]}):"
    for t in "${FAILED[@]}"; do printf "  ✗ %s\n" "${t}"; done
fi

echo
echo "输出目录: ${DIST_DIR}/"
ls -lh "${DIST_DIR}/"
