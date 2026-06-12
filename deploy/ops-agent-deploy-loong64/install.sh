#!/bin/bash
# OPS-Agent One-Click Deployment Script for Loongson (loong64)
# Compatible: Loongson 3A5000/3A6000/3C5000 + Kylin/UOS/Loongnix
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

INSTALL_DIR="/opt/ops-agent"
BINARY_NAME="ops-agent"
SERVICE_NAME="ops-agent"
SYSTEMD_SERVICE="/etc/systemd/system/${SERVICE_NAME}.service"

clear
echo -e "${CYAN}"
echo "  ___  ____  ____       _                    _   "
echo " / _ \|  _ \/ ___|     / \   __ _  ___ _ __ | |_ "
echo "| | | | |_) \___ \    / _ \ / _\` |/ _ \ '_ \| __|"
echo "| |_| |  __/ ___) |  / ___ \ (_| |  __/ | | | |_ "
echo " \___/|_|   |____/  /_/   \_\__, |\___|_| |_|\__|"
echo "                            |___/                 "
echo -e "${NC}"
echo -e "${BOLD}Linux OPS-Agent v1.0.0 (Loongson loong64)${NC}"
echo -e "------------------------------------------------"
echo ""

# ============ Pre-flight Checks ============

# Check architecture
ARCH=$(uname -m)
echo -e "${BOLD}[1/8] System Architecture${NC}"
if [ "$ARCH" != "loongarch64" ]; then
    echo -e "  ${YELLOW}[WARN] Current arch: $ARCH, this package is for loongarch64${NC}"
else
    echo -e "  ${GREEN}[OK] Arch: $ARCH (Loongson)${NC}"
fi

# Check root
echo ""
echo -e "${BOLD}[2/8] Privilege Check${NC}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
if [ "$EUID" -ne 0 ]; then
    echo -e "  ${YELLOW}[INFO] Non-root, install to current directory${NC}"
    INSTALL_DIR="$SCRIPT_DIR"
    echo -e "  Install dir: ${CYAN}$INSTALL_DIR${NC}"
else
    echo -e "  ${GREEN}[OK] Root privilege, install to $INSTALL_DIR${NC}"
fi

ENV_FILE="$INSTALL_DIR/.env"

# Check binary
BINARY="$SCRIPT_DIR/$BINARY_NAME"
echo ""
echo -e "${BOLD}[3/8] Binary Check${NC}"
if [ ! -f "$BINARY" ]; then
    echo -e "  ${RED}[ERROR] Binary not found: $BINARY${NC}"
    exit 1
fi
chmod +x "$BINARY"
echo -e "  ${GREEN}[OK] Binary ready ($(du -h "$BINARY" | cut -f1 | tr -d ' '))${NC}"

# Check web assets
echo ""
echo -e "${BOLD}[4/8] Web Assets Check${NC}"
if [ ! -d "$SCRIPT_DIR/web" ] || [ ! -f "$SCRIPT_DIR/web/index.html" ]; then
    echo -e "  ${RED}[ERROR] Web assets not found: $SCRIPT_DIR/web/${NC}"
    exit 1
fi
echo -e "  ${GREEN}[OK] Web assets ready${NC}"

# Check system dependencies
echo ""
echo -e "${BOLD}[5/8] System Dependencies${NC}"
MISSING_DEPS=()
for cmd in df ps free ss ip uname uptime cat head tail grep find; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        MISSING_DEPS+=("$cmd")
    fi
done
OPTIONAL_MISSING=()
for cmd in lsof journalctl systemctl top; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        OPTIONAL_MISSING+=("$cmd")
    fi
done

if [ ${#MISSING_DEPS[@]} -gt 0 ]; then
    echo -e "  ${RED}[ERROR] Missing required commands: ${MISSING_DEPS[*]}${NC}"
    echo -e "  ${YELLOW}Install with: apt install procps iproute2 coreutils findutils${NC}"
    echo -e "  ${YELLOW}Or:           yum install procps-ng iproute coreutils findutils${NC}"
    exit 1
fi
echo -e "  ${GREEN}[OK] Required commands available${NC}"

if [ ${#OPTIONAL_MISSING[@]} -gt 0 ]; then
    echo -e "  ${YELLOW}[WARN] Optional commands missing: ${OPTIONAL_MISSING[*]}${NC}"
    echo -e "  ${YELLOW}Some probe tools may not work (lsof, journalctl, etc.)${NC}"
fi

# Check systemd
echo ""
echo -e "${BOLD}[6/8] Systemd Check${NC}"
HAS_SYSTEMD=false
if command -v systemctl >/dev/null 2>&1; then
    HAS_SYSTEMD=true
    echo -e "  ${GREEN}[OK] Systemd available${NC}"
else
    echo -e "  ${YELLOW}[WARN] Systemd not available, will run in foreground mode${NC}"
fi

# Check port
echo ""
echo -e "${BOLD}[7/8] Port Check${NC}"
PORT=${PORT:-8080}
PORT_BUSY=false
if command -v ss >/dev/null 2>&1 && ss -tlnp 2>/dev/null | grep -q ":${PORT} "; then
    PORT_BUSY=true
elif command -v netstat >/dev/null 2>&1 && netstat -tlnp 2>/dev/null | grep -q ":${PORT} "; then
    PORT_BUSY=true
fi

if [ "$PORT_BUSY" = true ]; then
    echo -e "  ${YELLOW}[WARN] Port $PORT is in use${NC}"
    read -p "  Kill the process using this port? [y/N]: " KILL_PROC
    if [[ "$KILL_PROC" =~ ^[Yy] ]]; then
        if command -v fuser >/dev/null 2>&1; then
            fuser -k "${PORT}/tcp" 2>/dev/null || true
        else
            PID=$(ss -tlnp 2>/dev/null | grep ":${PORT} " | grep -oP 'pid=\K[0-9]+' | head -1)
            if [ -n "$PID" ]; then
                kill "$PID" 2>/dev/null || true
            fi
        fi
        sleep 1
    fi
else
    echo -e "  ${GREEN}[OK] Port $PORT available${NC}"
fi

# Check firewall
echo ""
echo -e "${BOLD}[8/8] Firewall Check${NC}"
FIREWALL_OPENED=false
if command -v firewall-cmd >/dev/null 2>&1; then
    if firewall-cmd --state 2>/dev/null | grep -q "running"; then
        echo -e "  ${YELLOW}[INFO] firewalld is running, opening port $PORT...${NC}"
        firewall-cmd --add-port="${PORT}/tcp" --permanent 2>/dev/null && \
        firewall-cmd --reload 2>/dev/null && \
        FIREWALL_OPENED=true
        if [ "$FIREWALL_OPENED" = true ]; then
            echo -e "  ${GREEN}[OK] firewalld: port $PORT opened${NC}"
        else
            echo -e "  ${YELLOW}[WARN] Failed to open port, please do it manually${NC}"
        fi
    else
        echo -e "  ${GREEN}[OK] firewalld not running${NC}"
    fi
elif command -v ufw >/dev/null 2>&1; then
    if ufw status 2>/dev/null | grep -q "active"; then
        echo -e "  ${YELLOW}[INFO] ufw is running, opening port $PORT...${NC}"
        ufw allow "${PORT}/tcp" 2>/dev/null && FIREWALL_OPENED=true
        if [ "$FIREWALL_OPENED" = true ]; then
            echo -e "  ${GREEN}[OK] ufw: port $PORT opened${NC}"
        else
            echo -e "  ${YELLOW}[WARN] Failed to open port, please do it manually${NC}"
        fi
    else
        echo -e "  ${GREEN}[OK] ufw not active${NC}"
    fi
elif command -v iptables >/dev/null 2>&1; then
    if iptables -L INPUT -n 2>/dev/null | grep -q "DROP\|REJECT"; then
        echo -e "  ${YELLOW}[INFO] iptables has DROP/REJECT rules, opening port $PORT...${NC}"
        iptables -I INPUT -p tcp --dport "$PORT" -j ACCEPT 2>/dev/null && FIREWALL_OPENED=true
        if [ "$FIREWALL_OPENED" = true ]; then
            echo -e "  ${GREEN}[OK] iptables: port $PORT opened${NC}"
            echo -e "  ${YELLOW}[WARN] This rule is not persistent across reboots${NC}"
        fi
    else
        echo -e "  ${GREEN}[OK] iptables: no restrictive rules${NC}"
    fi
else
    echo -e "  ${GREEN}[OK] No firewall detected${NC}"
fi

# ============ Configuration ============
echo ""
echo -e "${BOLD}------------------------------------------------${NC}"
echo -e "${BOLD}  LLM Configuration${NC}"
echo -e "${BOLD}------------------------------------------------${NC}"

strip_quotes() {
    local val="$1"
    if [ "${val:0:1}" = '"' ] && [ "${val: -1}" = '"' ]; then
        val="${val:1:${#val}-2}"
    elif [ "${val:0:1}" = "'" ] && [ "${val: -1}" = "'" ]; then
        val="${val:1:${#val}-2}"
    fi
    printf '%s' "$val"
}

if [ -f "$ENV_FILE" ]; then
    echo -e "  ${GREEN}[OK] Existing config found: $ENV_FILE${NC}"
    LLM_BASE_URL=$(strip_quotes "$(grep '^LLM_BASE_URL=' "$ENV_FILE" | cut -d= -f2-)")
    LLM_MODEL=$(strip_quotes "$(grep '^LLM_MODEL=' "$ENV_FILE" | cut -d= -f2-)")
    PORT=$(strip_quotes "$(grep '^PORT=' "$ENV_FILE" | cut -d= -f2-)")
    PORT=${PORT:-8080}
    echo -e "  Provider: ${CYAN}${LLM_BASE_URL:-not set}${NC}"
    echo -e "  Model:    ${CYAN}${LLM_MODEL:-not set}${NC}"
    echo -e "  Port:     ${CYAN}${PORT}${NC}"
    echo ""
    read -p "  Use existing config? [Y/n]: " USE_EXISTING
    if [[ "$USE_EXISTING" =~ ^[Nn] ]]; then
        rm -f "$ENV_FILE"
    fi
fi

if [ ! -f "$ENV_FILE" ]; then
    echo ""
    echo -e "  ${YELLOW}Configure LLM connection${NC}"
    echo ""
    echo -e "  Select LLM Provider:"
    echo -e "  ${BOLD}1${NC}) DeepSeek V4 Flash (recommended)"
    echo -e "  ${BOLD}2${NC}) Xiaomi MiMo V2.5 Pro (strong reasoning)"
    echo -e "  ${BOLD}3${NC}) Qwen 3.6 Plus (1M context)"
    echo -e "  ${BOLD}4${NC}) Custom"
    echo ""
    read -p "  Select [1-4]: " PROVIDER_CHOICE

    case "$PROVIDER_CHOICE" in
        1)
            LLM_BASE_URL="https://api.deepseek.com"
            LLM_MODEL="deepseek-v4-flash"
            PROVIDER_NAME="DeepSeek"
            ;;
        2)
            LLM_BASE_URL="https://token-plan-cn.xiaomimimo.com/v1"
            LLM_MODEL="mimo-v2.5-pro"
            PROVIDER_NAME="Xiaomi MiMo"
            ;;
        3)
            LLM_BASE_URL="https://dashscope.aliyuncs.com/compatible-mode/v1"
            LLM_MODEL="qwen3.6-plus"
            PROVIDER_NAME="Qwen (Alibaba)"
            ;;
        4)
            echo ""
            read -p "  Base URL: " LLM_BASE_URL
            read -p "  Model ID: " LLM_MODEL
            PROVIDER_NAME="Custom"
            ;;
        *)
            LLM_BASE_URL="https://api.deepseek.com"
            LLM_MODEL="deepseek-v4-flash"
            PROVIDER_NAME="DeepSeek"
            ;;
    esac

    echo ""
    echo -e "  Provider: ${GREEN}$PROVIDER_NAME${NC}"
    echo -e "  URL:      ${CYAN}$LLM_BASE_URL${NC}"
    echo -e "  Model:    ${CYAN}$LLM_MODEL${NC}"
    echo ""

    read -sp "  API Key: " LLM_API_KEY
    echo ""

    if [ -z "$LLM_API_KEY" ]; then
        echo -e "\n  ${RED}[ERROR] API Key cannot be empty${NC}"
        exit 1
    fi

    echo ""
    read -p "  Port [default 8080]: " PORT
    PORT=${PORT:-8080}

    read -p "  Admin password [default admin123]: " ADMIN_PASS
    ADMIN_PASS=${ADMIN_PASS:-admin123}

    # Generate JWT secret (portable, no xxd dependency)
    if command -v openssl >/dev/null 2>&1; then
        JWT_SECRET="ops-$(openssl rand -hex 16)"
    elif command -v head >/dev/null 2>&1 && [ -c /dev/urandom ]; then
        JWT_SECRET="ops-$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n' | head -c 32)"
    else
        JWT_SECRET="ops-$(date +%s%N 2>/dev/null || date +%s)$$"
    fi

    mkdir -p "$(dirname "$ENV_FILE")"

    # Write .env using printf to avoid shell expansion of $ in API key
    # This is critical: heredoc << EOF would expand $ characters in the key
    {
        printf '%s\n' '# OPS-Agent Configuration (auto-generated)'
        printf 'LLM_API_KEY=%s\n' "$LLM_API_KEY"
        printf 'LLM_BASE_URL=%s\n' "$LLM_BASE_URL"
        printf 'LLM_MODEL=%s\n' "$LLM_MODEL"
        printf 'PORT=%s\n' "$PORT"
        printf 'DB_PATH=./data/ops-agent.db\n'
        printf 'JWT_SECRET=%s\n' "$JWT_SECRET"
        printf 'ADMIN_PASSWORD=%s\n' "$ADMIN_PASS"
    } > "$ENV_FILE"

    chmod 600 "$ENV_FILE"
    echo ""
    echo -e "  ${GREEN}[OK] Config saved to $ENV_FILE${NC}"
fi

# ============ Installation ============
echo ""
echo -e "${BOLD}------------------------------------------------${NC}"
echo -e "${BOLD}  Installing...${NC}"
echo -e "${BOLD}------------------------------------------------${NC}"

if [ "$EUID" -eq 0 ] && [ "$SCRIPT_DIR" != "$INSTALL_DIR" ]; then
    echo -e "  Installing to ${CYAN}$INSTALL_DIR${NC}..."
    mkdir -p "$INSTALL_DIR"
    cp "$BINARY" "$INSTALL_DIR/"
    cp -r "$SCRIPT_DIR/web" "$INSTALL_DIR/"
    if [ -f "$SCRIPT_DIR/.env" ]; then
        cp "$SCRIPT_DIR/.env" "$INSTALL_DIR/"
    fi
    if [ -f "$SCRIPT_DIR/providers.json.example" ]; then
        cp "$SCRIPT_DIR/providers.json.example" "$INSTALL_DIR/"
    fi
    mkdir -p "$INSTALL_DIR/data"
    echo -e "  ${GREEN}[OK] Files installed${NC}"
else
    INSTALL_DIR="$SCRIPT_DIR"
    mkdir -p "$INSTALL_DIR/data"
    echo -e "  ${GREEN}[OK] Using current directory: $INSTALL_DIR${NC}"
fi

# ============ Systemd Service ============
if [ "$HAS_SYSTEMD" = true ] && [ "$EUID" -eq 0 ]; then
    echo ""
    echo -e "  Configuring systemd service..."

    PORT=$(strip_quotes "$(grep '^PORT=' "$ENV_FILE" | cut -d= -f2-)")
    PORT=${PORT:-8080}

    cat > "$SYSTEMD_SERVICE" << SYSEOF
[Unit]
Description=OPS-Agent - Linux OPS Agent
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/$BINARY_NAME
Restart=on-failure
RestartSec=5
StartLimitBurst=3
StartLimitIntervalSec=60

EnvironmentFile=$INSTALL_DIR/.env

StandardOutput=journal
StandardError=journal
SyslogIdentifier=$SERVICE_NAME

[Install]
WantedBy=multi-user.target
SYSEOF

    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME" 2>/dev/null || true
    echo -e "  ${GREEN}[OK] Systemd service configured${NC}"
    echo ""
    echo -e "  Management commands:"
    echo -e "    Start:      ${CYAN}systemctl start $SERVICE_NAME${NC}"
    echo -e "    Stop:       ${CYAN}systemctl stop $SERVICE_NAME${NC}"
    echo -e "    Status:     ${CYAN}systemctl status $SERVICE_NAME${NC}"
    echo -e "    Logs:       ${CYAN}journalctl -u $SERVICE_NAME -f${NC}"
    echo -e "    Auto-start: ${CYAN}systemctl enable $SERVICE_NAME${NC}"
    echo -e "    Uninstall:  ${CYAN}bash $INSTALL_DIR/uninstall.sh${NC}"

    echo ""
    read -p "  Start service now? [Y/n]: " START_NOW
    if [[ ! "$START_NOW" =~ ^[Nn] ]]; then
        systemctl start "$SERVICE_NAME"
        sleep 2

        # Health check
        echo -e "  Health check..."
        HEALTH_OK=false
        for i in 1 2 3 4 5; do
            sleep 1
            if command -v curl >/dev/null 2>&1 && curl -sf "http://localhost:${PORT}/health" >/dev/null 2>&1; then
                HEALTH_OK=true
                break
            elif command -v wget >/dev/null 2>&1 && wget -q -O /dev/null "http://localhost:${PORT}/health" 2>/dev/null; then
                HEALTH_OK=true
                break
            elif echo > /dev/tcp/localhost/${PORT} 2>/dev/null; then
                HEALTH_OK=true
                break
            fi
        done

        if [ "$HEALTH_OK" = true ]; then
            echo -e "  ${GREEN}[OK] Service started and healthy${NC}"
        elif systemctl is-active --quiet "$SERVICE_NAME"; then
            echo -e "  ${YELLOW}[WARN] Service is running but health check failed${NC}"
            echo -e "  ${YELLOW}       Check logs: journalctl -u $SERVICE_NAME -n 20${NC}"
        else
            echo -e "  ${RED}[ERROR] Service failed to start${NC}"
            echo -e "  ${CYAN}journalctl -u $SERVICE_NAME -n 30${NC}"
            exit 1
        fi
    fi
else
    echo ""
    echo -e "  ${YELLOW}Non-root or no systemd, using foreground mode${NC}"
    echo ""
    read -p "  Start now? [Y/n]: " START_NOW
    if [[ ! "$START_NOW" =~ ^[Nn] ]]; then
        PORT=$(strip_quotes "$(grep '^PORT=' "$ENV_FILE" | cut -d= -f2-)")
        PORT=${PORT:-8080}
        echo ""
        echo -e "  ${GREEN}Starting OPS-Agent...${NC}"
        echo -e "  Web UI:  ${CYAN}http://localhost:$PORT${NC}"
        echo -e "  API:     ${CYAN}http://localhost:$PORT/api/v1/chat${NC}"
        echo -e "  Press ${BOLD}Ctrl+C${NC} to stop"
        echo ""
        cd "$INSTALL_DIR"
        exec "./$BINARY_NAME"
    fi
fi

# ============ Summary ============
echo ""
echo -e "${BOLD}------------------------------------------------${NC}"
PORT=$(strip_quotes "$(grep '^PORT=' "$ENV_FILE" | cut -d= -f2-)")
PORT=${PORT:-8080}

# Detect external IP for remote access
EXT_IP=""
if command -v hostname >/dev/null 2>&1; then
    EXT_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
fi

echo -e "${GREEN}${BOLD}  Installation Complete!${NC}"
echo -e "${BOLD}------------------------------------------------${NC}"
echo -e "  Local:    ${CYAN}http://localhost:$PORT${NC}"
if [ -n "$EXT_IP" ]; then
    echo -e "  Remote:   ${CYAN}http://$EXT_IP:$PORT${NC}"
fi
echo -e "  API:      ${CYAN}http://localhost:$PORT/api/v1/chat${NC}"
echo -e "  Config:   ${CYAN}$ENV_FILE${NC}"
echo -e "  Data:     ${CYAN}$INSTALL_DIR/data/${NC}"
echo -e "  Uninstall: ${CYAN}bash $INSTALL_DIR/uninstall.sh${NC}"
echo ""
echo -e "  ${YELLOW}Troubleshooting:${NC}"
echo -e "    Locked out:  ${CYAN}curl -X DELETE http://localhost:$PORT/api/v1/auth/lockout/YOUR_IP${NC}"
echo -e "    View locks:  ${CYAN}curl http://localhost:$PORT/api/v1/auth/lockouts${NC}"
echo -e "    Reset pass:  ${CYAN}Edit $ENV_FILE and restart service${NC}"
echo ""
