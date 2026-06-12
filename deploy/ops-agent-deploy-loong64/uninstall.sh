#!/bin/bash
# OPS-Agent Uninstall Script
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

SERVICE_NAME="ops-agent"
INSTALL_DIR="/opt/ops-agent"

echo -e "${BOLD}OPS-Agent 卸载程序${NC}"
echo ""

if [ "$EUID" -ne 0 ]; then
    echo -e "${YELLOW}[WARN] 需要root权限卸载${NC}"
    echo -e "请使用: sudo $0"
    exit 1
fi

read -p "确认卸载 OPS-Agent? 此操作将删除所有文件和数据 [y/N]: " CONFIRM
if [[ ! "$CONFIRM" =~ ^[Yy] ]]; then
    echo "已取消"
    exit 0
fi

# Stop service
if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
    echo -e "停止服务..."
    systemctl stop "$SERVICE_NAME"
fi

if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
    echo -e "禁用服务..."
    systemctl disable "$SERVICE_NAME" 2>/dev/null || true
fi

# Remove systemd service
if [ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
    echo -e "删除systemd服务文件..."
    rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
    systemctl daemon-reload
fi

# Remove installation
if [ -d "$INSTALL_DIR" ]; then
    echo -e "删除安装目录: $INSTALL_DIR"
    read -p "是否同时删除数据目录 $INSTALL_DIR/data? [y/N]: " DEL_DATA
    if [[ "$DEL_DATA" =~ ^[Yy] ]]; then
        rm -rf "$INSTALL_DIR"
        echo -e "${GREEN}[OK] 已删除所有文件${NC}"
    else
        rm -rf "$INSTALL_DIR/bin" "$INSTALL_DIR/web" "$INSTALL_DIR/.env" "$INSTALL_DIR/ops-agent" "$INSTALL_DIR/install.sh" "$INSTALL_DIR/uninstall.sh"
        echo -e "${GREEN}[OK] 已删除程序文件, 保留数据目录${NC}"
    fi
fi

echo ""
echo -e "${GREEN}${BOLD}卸载完成${NC}"
