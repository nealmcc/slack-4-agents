#!/bin/bash
# update-auth.sh - Update Slack MCP authentication in ~/.claude.json

set -e

CLAUDE_CONFIG="$HOME/.claude.json"

echo "=========================================="
echo "  Slack MCP Authentication Updater"
echo "=========================================="
echo ""

if [ ! -f "$CLAUDE_CONFIG" ]; then
    echo "Error: $CLAUDE_CONFIG not found"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed"
    echo "Install with: brew install jq"
    exit 1
fi

echo "Step 1: Get the SLACK_COOKIE"
echo "----------------------------"
echo "1. Open your Slack workspace in a browser"
echo "2. Open Developer Tools (F12)"
echo "3. Go to Application → Cookies → your workspace URL"
echo "4. Find the 'd' cookie"
echo "5. Copy the value (starts with 'xoxd-')"
echo ""
read -p "Enter SLACK_COOKIE: " SLACK_COOKIE

if [[ ! "$SLACK_COOKIE" =~ ^xoxd- ]]; then
    echo "Warning: Cookie doesn't start with 'xoxd-'. Continue anyway? (y/n)"
    read -n 1 confirm
    echo ""
    if [[ "$confirm" != "y" ]]; then
        exit 1
    fi
fi

echo ""
echo "Step 2: Get the SLACK_TOKEN"
echo "---------------------------"
echo "1. In the same browser, open the Console tab"
echo "2. Paste and run this JavaScript:"
echo ""
echo '   JSON.parse(localStorage.localConfig_v2).teams[Object.keys(JSON.parse(localStorage.localConfig_v2).teams)[0]].token'
echo ""
echo "3. Copy the result (starts with 'xoxc-')"
echo ""
read -p "Enter SLACK_TOKEN: " SLACK_TOKEN

if [[ ! "$SLACK_TOKEN" =~ ^xoxc- ]]; then
    echo "Warning: Token doesn't start with 'xoxc-'. Continue anyway? (y/n)"
    read -n 1 confirm
    echo ""
    if [[ "$confirm" != "y" ]]; then
        exit 1
    fi
fi

echo ""
echo "Updating $CLAUDE_CONFIG..."

# Create backup
cp "$CLAUDE_CONFIG" "$CLAUDE_CONFIG.bak"

# Update the values using jq
jq --arg token "$SLACK_TOKEN" --arg cookie "$SLACK_COOKIE" \
   '.mcpServers.slack.env.SLACK_TOKEN = $token | .mcpServers.slack.env.SLACK_COOKIE = $cookie' \
   "$CLAUDE_CONFIG" > "$CLAUDE_CONFIG.tmp" && mv "$CLAUDE_CONFIG.tmp" "$CLAUDE_CONFIG"

echo "Done! Backup saved to $CLAUDE_CONFIG.bak"
echo ""
echo "Restart Claude Code for changes to take effect."
