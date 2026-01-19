# Slack MCP Server

A Go-based MCP (Model Context Protocol) server that provides Claude Code with read-only access to Slack workspaces.

## Installation

```bash
go build -o slack-mcp ./cmd/slack-mcp
```

## Configuration

Add to your Claude Code config (`~/.claude.json`):

```json
{
  "mcpServers": {
    "slack": {
      "type": "stdio",
      "command": "/path/to/slack-mcp",
      "args": [],
      "env": {
        "SLACK_TOKEN": "xoxc-your-token-here",
        "SLACK_COOKIE": "xoxd-your-cookie-here"
      }
    }
  }
}
```

### Authentication

The server supports two authentication methods:

| Token Type | Environment Variables | Use Case |
|------------|----------------------|----------|
| Bot/User OAuth | `SLACK_TOKEN` only | Slack apps with proper OAuth scopes |
| Browser token | `SLACK_TOKEN` + `SLACK_COOKIE` | Personal use without creating a Slack app |

#### Getting browser tokens

1. Open `<workspace>.slack.com` in your browser and sign in
2. Open Developer Tools (F12) → Application → Cookies
3. Find the `d` cookie value (starts with `xoxd-`) → use as `SLACK_COOKIE`
4. In the Console tab, run: `JSON.parse(localStorage.localConfig_v2).teams[Object.keys(JSON.parse(localStorage.localConfig_v2).teams)[0]].token`
5. Copy the token (starts with `xoxc-`) → use as `SLACK_TOKEN`

### Logging

The server uses structured logging (zap) with configurable log levels. Control verbosity with the `LOG_LEVEL` environment variable:

```json
{
  "mcpServers": {
    "slack": {
      "type": "stdio",
      "command": "/path/to/slack-mcp",
      "args": [],
      "env": {
        "SLACK_TOKEN": "xoxc-your-token-here",
        "SLACK_COOKIE": "xoxd-your-cookie-here",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

**Available log levels:**

| Level | Description | Use Case |
|-------|-------------|----------|
| `debug` | Verbose logging including cache hits, HTTP requests, page details | Troubleshooting and development |
| `info` | Normal operational logging (default) | Production use |
| `warn` | Warnings and errors only | Minimal logging |
| `error` | Errors only | Critical issues only |

**What gets logged:**
- **Debug**: Cache lookups, API calls, pagination details
- **Info**: Client initialization, channel searches, rate limit retries
- **Warn**: Rate limits, channels not found
- **Error**: Authentication failures, API errors

**Log output:**
- Logs are written to both stderr and `~/.claude/slack-mcp.log`
- Format: JSON with ISO8601 timestamps
- The log file is appended to, not rotated (manual cleanup needed if it grows large)

## Available Tools

| Tool | Description |
|------|-------------|
| `slack_list_channels` | List channels you have access to |
| `slack_read_history` | Read messages from a channel |
| `slack_search_messages` | Search messages across workspace |
| `slack_get_user` | Look up user by ID or email |
| `slack_get_permalink` | Get permalink to a message |

## Usage Examples

Once configured, Claude Code can use these tools:

- "List all my Slack channels"
- "Show me the last 10 messages in #general"
- "Search for messages about the deployment"
- "Who is user U1234567890?"
- "Get a link to that message"

## Required Slack Scopes

For bot/user OAuth tokens, you'll need these scopes:

- `channels:read` - List public channels
- `groups:read` - List private channels
- `channels:history` - Read public channel messages
- `groups:history` - Read private channel messages
- `search:read` - Search messages
- `users:read` - Look up users
- `users:read.email` - Look up users by email

## Development

```bash
# Build
make build

# Build with optimizations
make build-release

# Run tests
make test

# Format code
make fmt

# Run all checks (fmt, vet, test)
make check

# Test MCP protocol with debug logging
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | LOG_LEVEL=debug SLACK_TOKEN=fake ./slack-mcp
```

Run `make help` to see all available targets.

## License

MIT
