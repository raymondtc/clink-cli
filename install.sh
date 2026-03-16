#!/bin/bash
# Clink CLI 一键安装脚本
# 支持 macOS 和 Linux，自动识别系统架构

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印信息
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 获取最新版本
get_latest_version() {
    curl -s "https://api.github.com/repos/raymondtc/clink-cli/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# 检测操作系统
detect_os() {
    local os
    case "$(uname -s)" in
        Linux*)     os=linux;;
        Darwin*)    os=darwin;;
        CYGWIN*)    os=windows;;
        MINGW*)     os=windows;;
        MSYS*)      os=windows;;
        *)          os=unknown;;
    esac
    echo "$os"
}

# 检测架构
detect_arch() {
    local arch
    case "$(uname -m)" in
        x86_64)     arch=amd64;;
        amd64)      arch=amd64;;
        arm64)      arch=arm64;;
        aarch64)    arch=arm64;;
        *)          arch=unknown;;
    esac
    echo "$arch"
}

# 下载文件
download_file() {
    local url=$1
    local output=$2
    
    if command -v curl &> /dev/null; then
        curl -fsSL "$url" -o "$output"
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O "$output"
    else
        error "需要 curl 或 wget 来下载文件"
        exit 1
    fi
}

# 主安装流程
main() {
    info "开始安装 Clink CLI..."
    
    # 检测系统和架构
    OS=$(detect_os)
    ARCH=$(detect_arch)
    
    if [ "$OS" = "unknown" ]; then
        error "不支持的操作系统"
        exit 1
    fi
    
    if [ "$ARCH" = "unknown" ]; then
        error "不支持的架构: $(uname -m)"
        exit 1
    fi
    
    info "检测到系统: $OS, 架构: $ARCH"
    
    # 获取最新版本
    info "获取最新版本..."
    VERSION=$(get_latest_version)
    
    if [ -z "$VERSION" ]; then
        error "无法获取最新版本信息"
        exit 1
    fi
    
    success "最新版本: $VERSION"
    
    # 创建临时目录
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT
    
    # 下载二进制文件
    BASE_URL="https://github.com/raymondtc/clink-cli/releases/download/$VERSION"
    CLI_FILE="clink-${OS}-${ARCH}"
    MCP_FILE="clink-mcp-${OS}-${ARCH}"
    
    info "下载 clink..."
    download_file "$BASE_URL/$CLI_FILE" "$TMP_DIR/clink"
    chmod +x "$TMP_DIR/clink"
    
    info "下载 clink-mcp..."
    download_file "$BASE_URL/$MCP_FILE" "$TMP_DIR/clink-mcp"
    chmod +x "$TMP_DIR/clink-mcp"
    
    # 安装路径
    INSTALL_DIR="/usr/local/bin"
    
    # 检查是否有写入权限
    if [ ! -w "$INSTALL_DIR" ]; then
        warning "需要 sudo 权限来安装到 $INSTALL_DIR"
        SUDO="sudo"
    else
        SUDO=""
    fi
    
    # 安装
    info "安装到 $INSTALL_DIR..."
    $SUDO mv "$TMP_DIR/clink" "$INSTALL_DIR/clink"
    $SUDO mv "$TMP_DIR/clink-mcp" "$INSTALL_DIR/clink-mcp"
    
    # 验证安装
    if command -v clink &> /dev/null; then
        success "Clink CLI 安装成功!"
        clink --help 2>/dev/null | head -3
    else
        error "安装失败，请检查 $INSTALL_DIR 是否在 PATH 中"
        exit 1
    fi
    
    if command -v clink-mcp &> /dev/null; then
        success "Clink MCP Server 安装成功!"
    else
        warning "MCP Server 安装可能失败"
    fi
    
    echo ""
    success "安装完成!"
    echo ""
    info "下一步:"
    echo "  1. 配置环境变量:"
    echo "     export CLINK_ACCESS_ID='your_access_key_id'"
    echo "     export CLINK_ACCESS_SECRET='your_secret'"
    echo ""
    echo "  2. 验证安装:"
    echo "     clink --help"
    echo "     clink agents"
    echo ""
    echo "  3. 查看文档:"
    echo "     https://github.com/raymondtc/clink-cli/blob/main/docs/AGENT_MANUAL.md"
    echo ""
}

# 执行主流程
main
