#!/usr/bin/env bash
# ============================================================================
# Cosmo Mail 飞牛 fnOS 安装包（fpk）构建脚本（Linux/macOS）
# 用法: ./build-fnos.sh [BinaryPath]
#   BinaryPath 默认 bin/cosmomail-linux-amd64（先执行 ./build.sh linux amd64）
# Windows 用户请使用 build-fnos.ps1
# ============================================================================
set -euo pipefail

BINARY_PATH="${1:-bin/cosmomail-linux-amd64}"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="${PROJECT_ROOT}/fnapp"
FNPACK_VERSION="1.2.1"

# 检查二进制
if [ ! -f "${PROJECT_ROOT}/${BINARY_PATH}" ]; then
    echo "✗ 未找到二进制: ${BINARY_PATH}"
    echo "  请先执行: ./build.sh linux amd64"
    exit 1
fi

# 检查/下载 fnpack
ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64) FNPACK_ARCH="amd64" ;;
    aarch64|arm64) FNPACK_ARCH="arm64" ;;
    *) echo "✗ 不支持的主机架构: ${ARCH}"; exit 1 ;;
esac
HOST_OS="$(uname -s | tr '[:upper:]' '[:lower:]')"

FNPACK="${PROJECT_ROOT}/.tools/fnpack-${FNPACK_VERSION}-${HOST_OS}-${FNPACK_ARCH}"
if [ ! -x "${FNPACK}" ]; then
    mkdir -p "$(dirname "${FNPACK}")"
    URL="https://static2.fnnas.com/fnpack/fnpack-${FNPACK_VERSION}-${HOST_OS}-${FNPACK_ARCH}"
    echo "▶ 下载 fnpack: ${URL}"
    curl -fsSL -o "${FNPACK}" "${URL}" || { echo "✗ fnpack 下载失败（可手动放置到 ${FNPACK}）"; exit 1; }
    chmod +x "${FNPACK}"
fi

# 同步资源到打包目录
SERVER_DIR="${PACKAGE_DIR}/app/server"
UI_IMAGES_DIR="${PACKAGE_DIR}/app/ui/images"
mkdir -p "${SERVER_DIR}" "${UI_IMAGES_DIR}"
cp "${PROJECT_ROOT}/${BINARY_PATH}" "${SERVER_DIR}/cosmomail"
cp "${PROJECT_ROOT}/public/images/icon_64.png" "${UI_IMAGES_DIR}/icon-64.png"
cp "${PROJECT_ROOT}/public/images/icon_256.png" "${UI_IMAGES_DIR}/icon-256.png"
cp "${PROJECT_ROOT}/public/images/icon_64.png" "${PACKAGE_DIR}/ICON.PNG"
cp "${PROJECT_ROOT}/public/images/icon_256.png" "${PACKAGE_DIR}/ICON_256.PNG"
cp "${PROJECT_ROOT}/LICENSE" "${PACKAGE_DIR}/LICENSE"

# 从 version.json 同步版本号到 manifest
VERSION="$(node -p "require('${PROJECT_ROOT}/version.json').latest.replace(/^v/, '')")"
sed -i.bak -E "s/^version=.*/version=${VERSION}/" "${PACKAGE_DIR}/manifest" && rm -f "${PACKAGE_DIR}/manifest.bak"
echo "▶ manifest 版本已同步: ${VERSION}"

# 打包
(
    cd "${PACKAGE_DIR}"
    "${FNPACK}" build
)

RAW_PACKAGE="${PACKAGE_DIR}/com.qwernot.cosmomail.fpk"
if [ ! -f "${RAW_PACKAGE}" ]; then
    echo "✗ fnpack 未生成 fpk 文件"
    exit 1
fi

OUTPUT_PACKAGE="${PACKAGE_DIR}/CosmoMail-fnOS-x86_64-${VERSION}.fpk"
mv -f "${RAW_PACKAGE}" "${OUTPUT_PACKAGE}"
echo "✅ 构建完成: ${OUTPUT_PACKAGE}"
