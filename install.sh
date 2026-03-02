#!/bin/bash
set -eou pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Slack 4 Agents Installer${NC}"
echo ""

SERVER_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVER_NAME="slack-4-agents"
INSTALL_DIR="$HOME/.claude/servers/$SERVER_NAME"

# --- Build ---

echo "  Building $SERVER_NAME..."
(cd "$SERVER_DIR" && go build -ldflags "-s -w" -o "$SERVER_NAME" ./cmd/slack-4-agents)
echo -e "${GREEN}  [BUILT] $SERVER_NAME${NC}"

# --- Install binary ---

echo ""
mkdir -p "$INSTALL_DIR"
cp "$SERVER_DIR/$SERVER_NAME" "$INSTALL_DIR/$SERVER_NAME"

GIT_COMMIT=$(cd "$SERVER_DIR" && git rev-parse --short HEAD 2>/dev/null || echo "unknown")
cat > "$INSTALL_DIR/.version" <<EOF
commit=$GIT_COMMIT
installed=$(date +"%Y-%m-%d %H:%M:%S")
source=$SERVER_DIR
EOF
echo -e "${GREEN}  [COPIED] Binary to $INSTALL_DIR/$SERVER_NAME (commit $GIT_COMMIT)${NC}"

# --- Register MCP server with Claude Code ---

echo ""
if ! command -v claude >/dev/null 2>&1; then
    echo -e "${YELLOW}  [SKIP] Claude CLI not found. Register manually:${NC}"
    echo "    claude mcp add -s user -t stdio -- slack-4-agents $INSTALL_DIR/$SERVER_NAME"
    echo ""
    echo -e "${GREEN}Installation complete!${NC}"
    echo ""
    exit 0
fi

TARGET_PATH="$INSTALL_DIR/$SERVER_NAME"

CLAUDE_JSON="$HOME/.claude.json"
if [ -f "$CLAUDE_JSON" ] && command -v jq >/dev/null 2>&1; then
    CURRENT_PATH=$(jq -r '.mcpServers["slack-4-agents"].command // empty' "$CLAUDE_JSON" 2>/dev/null || true)
    if [ "$CURRENT_PATH" = "$TARGET_PATH" ]; then
        echo -e "${YELLOW}  [SKIP] slack-4-agents already registered with correct path${NC}"
        echo ""
        echo -e "${GREEN}Installation complete!${NC}"
        echo ""
        exit 0
    fi
fi

# Remove existing registration (ignore error if not found)
claude mcp remove -s user slack-4-agents 2>/dev/null || true

MISSING_VARS=""
[ -z "${SLACK_TOKEN:-}" ] && MISSING_VARS="$MISSING_VARS SLACK_TOKEN"
[ -z "${SLACK_COOKIE:-}" ] && MISSING_VARS="$MISSING_VARS SLACK_COOKIE"

if [ -n "$MISSING_VARS" ]; then
    echo -e "${RED}  [ERROR] Cannot register MCP server: missing env vars:${MISSING_VARS}${NC}"
    echo ""
    echo "  Export these in your shell and re-run, or register manually:"
    echo "    claude mcp add -s user -t stdio \\"
    echo "      -e SLACK_TOKEN='\$SLACK_TOKEN' \\"
    echo "      -e SLACK_COOKIE='\$SLACK_COOKIE' \\"
    echo "      -e LOG_LEVEL=debug \\"
    echo "      -- slack-4-agents $TARGET_PATH"
    exit 1
fi

# Note: -e <env...> is variadic, so -- is needed before positional args
claude mcp add -s user -t stdio \
    -e SLACK_TOKEN="$SLACK_TOKEN" \
    -e SLACK_COOKIE="$SLACK_COOKIE" \
    -e LOG_LEVEL="${LOG_LEVEL:-info}" \
    -- slack-4-agents "$TARGET_PATH"
echo -e "${GREEN}  [REGISTERED] MCP server with Claude Code${NC}"

echo ""
echo -e "${GREEN}Installation complete!${NC}"
echo ""
