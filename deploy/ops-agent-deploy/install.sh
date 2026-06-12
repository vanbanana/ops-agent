#!/bin/bash
# OPS-Agent Interactive Setup & Launch Script
# Compatible: WSL2 / Ubuntu / CentOS / Kylin

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

INSTALL_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY="$INSTALL_DIR/ops-agent"
ENV_FILE="$INSTALL_DIR/.env"

clear
echo -e "${CYAN}"
echo "  ___  ____  ____       _                    _   "
echo " / _ \|  _ \/ ___|     / \   __ _  ___ _ __ | |_ "
echo "| | | | |_) \___ \    / _ \ / _\` |/ _ \ '_ \| __|"
echo "| |_| |  __/ ___) |  / ___ \ (_| |  __/ | | | |_ "
echo " \___/|_|   |____/  /_/   \_\__, |\___|_| |_|\__|"
echo "                            |___/                 "
echo -e "${NC}"
echo -e "${BOLD}Linux 运维智能体 v1.0.0${NC}"
echo -e "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check binary
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}[ERROR] Binary not found: $BINARY${NC}"
    exit 1
fi
chmod +x "$BINARY"
mkdir -p "$INSTALL_DIR/data"

# ============ Interactive Configuration ============
if [ -f "$ENV_FILE" ]; then
    echo -e "${GREEN}[OK] Found existing .env configuration${NC}"
    echo ""
    source "$ENV_FILE" 2>/dev/null || true
    echo -e "  LLM Provider: ${CYAN}${LLM_BASE_URL:-not set}${NC}"
    echo -e "  Model:        ${CYAN}${LLM_MODEL:-not set}${NC}"
    echo -e "  Port:         ${CYAN}${PORT:-8080}${NC}"
    echo ""
    read -p "Use existing config? [Y/n]: " USE_EXISTING
    if [[ "$USE_EXISTING" =~ ^[Nn] ]]; then
        rm -f "$ENV_FILE"
    fi
fi

if [ ! -f "$ENV_FILE" ]; then
    echo -e "${YELLOW}[Setup] Let's configure your LLM connection${NC}"
    echo ""
    echo -e "  Choose a provider:"
    echo -e "  ${BOLD}1${NC}) DeepSeek V4 (recommended, cheap & fast)"
    echo -e "  ${BOLD}2${NC}) Xiaomi MiMo V2.5 Pro (strong reasoning)"
    echo -e "  ${BOLD}3${NC}) Qwen 3.6 Plus (1M context)"
    echo -e "  ${BOLD}4${NC}) Custom (enter manually)"
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
            read -p "  Base URL (e.g. https://api.deepseek.com): " LLM_BASE_URL
            read -p "  Model ID (e.g. deepseek-v4-flash): " LLM_MODEL
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

    # API Key
    echo -e "  ${YELLOW}Enter your API Key for $PROVIDER_NAME:${NC}"
    read -sp "  API Key: " LLM_API_KEY
    echo ""

    if [ -z "$LLM_API_KEY" ]; then
        echo -e "${RED}[ERROR] API Key cannot be empty${NC}"
        exit 1
    fi

    # Port
    echo ""
    read -p "  Port [default 8080]: " PORT
    PORT=${PORT:-8080}

    # Write .env
    cat > "$ENV_FILE" << EOF
# OPS-Agent Configuration (auto-generated)
LLM_API_KEY=$LLM_API_KEY
LLM_BASE_URL=$LLM_BASE_URL
LLM_MODEL=$LLM_MODEL
PORT=$PORT
DB_PATH=./data/ops-agent.db
JWT_SECRET=dev-ops-agent-$(date +%s | sha256sum | head -c 16)
EOF

    echo ""
    echo -e "${GREEN}[OK] Configuration saved to .env${NC}"
fi

# ============ Pre-flight Checks ============
echo ""
echo -e "${BOLD}Pre-flight Checks:${NC}"

# Port check
source "$ENV_FILE" 2>/dev/null || true
PORT=${PORT:-8080}
if command -v lsof >/dev/null 2>&1 && lsof -i ":$PORT" >/dev/null 2>&1; then
    echo -e "  ${YELLOW}[WARN] Port $PORT in use. Killing existing process...${NC}"
    kill $(lsof -ti ":$PORT") 2>/dev/null || true
    sleep 1
fi
echo -e "  ${GREEN}[OK] Port $PORT available${NC}"

# Network check (quick DNS resolve)
if command -v host >/dev/null 2>&1; then
    if host api.deepseek.com >/dev/null 2>&1; then
        echo -e "  ${GREEN}[OK] Network connectivity${NC}"
    else
        echo -e "  ${YELLOW}[WARN] DNS resolution slow, LLM calls may timeout${NC}"
    fi
else
    echo -e "  ${CYAN}[SKIP] Network check (host command not found)${NC}"
fi

# Disk space
AVAIL=$(df -m "$INSTALL_DIR" 2>/dev/null | tail -1 | awk '{print $4}')
if [ -n "$AVAIL" ] && [ "$AVAIL" -gt 100 ]; then
    echo -e "  ${GREEN}[OK] Disk space: ${AVAIL}MB available${NC}"
else
    echo -e "  ${YELLOW}[WARN] Low disk space${NC}"
fi

echo ""
echo -e "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}${BOLD}  Starting OPS-Agent...${NC}"
echo -e "  Web UI:  ${CYAN}http://localhost:$PORT${NC}"
echo -e "  API:     ${CYAN}http://localhost:$PORT/api/v1/chat${NC}"
echo -e ""
echo -e "  Press ${BOLD}Ctrl+C${NC} to stop"
echo -e "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

cd "$INSTALL_DIR"
exec ./ops-agent
