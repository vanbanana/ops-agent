#!/bin/bash
# Build OPS-Agent Loongson (loong64) Deployment Package
# Run on any machine with Go 1.22+ and Node.js 18+
set -e

GREEN='\033[0;32m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEPLOY_DIR="$PROJECT_ROOT/deploy/ops-agent-deploy-loong64"
VERSION="${1:-1.0.0}"
GIT_COMMIT=$(cd "$PROJECT_ROOT" && git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
OUTPUT_NAME="ops-agent-loong64-v${VERSION}"

echo -e "${BOLD}OPS-Agent Loongson Build${NC}"
echo -e "Version:   $VERSION"
echo -e "Commit:    $GIT_COMMIT"
echo -e "BuildTime: $BUILD_TIME"
echo ""

# Step 1: Build frontend
echo -e "${CYAN}[1/4] Building frontend...${NC}"
cd "$PROJECT_ROOT/web"
npm install --silent 2>/dev/null
npm run build
echo -e "${GREEN}[OK] Frontend built${NC}"

# Step 2: Cross-compile Go binary for loong64
echo -e "${CYAN}[2/4] Cross-compiling for loong64...${NC}"
cd "$PROJECT_ROOT"
rm -f "$DEPLOY_DIR/ops-agent"
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build \
    -ldflags "-s -w -X main.Version=$VERSION -X main.GitCommit=$GIT_COMMIT -X 'main.BuildTime=$BUILD_TIME'" \
    -o "$DEPLOY_DIR/ops-agent" \
    ./cmd/server/
echo -e "${GREEN}[OK] Binary built ($(du -h "$DEPLOY_DIR/ops-agent" | cut -f1))${NC}"

# Step 3: Copy web assets
echo -e "${CYAN}[3/4] Copying web assets...${NC}"
rm -rf "$DEPLOY_DIR/web"
mkdir -p "$DEPLOY_DIR/web"
cp -r "$PROJECT_ROOT/web/dist/"* "$DEPLOY_DIR/web/"
echo -e "${GREEN}[OK] Web assets copied${NC}"

# Step 4: Package
echo -e "${CYAN}[4/4] Packaging...${NC}"
cd "$PROJECT_ROOT/deploy"
rm -f "${OUTPUT_NAME}.tar.gz"
chmod +x "$DEPLOY_DIR/install.sh" "$DEPLOY_DIR/uninstall.sh"
tar -czf "${OUTPUT_NAME}.tar.gz" -C "$PROJECT_ROOT/deploy" "ops-agent-deploy-loong64"
echo -e "${GREEN}[OK] Package: deploy/${OUTPUT_NAME}.tar.gz ($(du -h "${OUTPUT_NAME}.tar.gz" | cut -f1))${NC}"

echo ""
echo -e "${BOLD}------------------------------------------------${NC}"
echo -e "${GREEN}${BOLD}Build complete!${NC}"
echo -e "${BOLD}------------------------------------------------${NC}"
echo -e "  Package:  ${CYAN}deploy/${OUTPUT_NAME}.tar.gz${NC}"
echo -e "  Deploy:   ${CYAN}scp ${OUTPUT_NAME}.tar.gz root@loongson-server:/tmp/${NC}"
echo -e "  Install:  ${CYAN}cd /tmp && tar xzf ${OUTPUT_NAME}.tar.gz && cd ops-agent-deploy-loong64 && bash install.sh${NC}"
