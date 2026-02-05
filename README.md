# Slack MCP Server

A Go-based [MCP server](https://modelcontextprotocol.io/) that provides Claude Code with read-only access to Slack. This enables Claude to search messages, read channel history, look up users, and export conversations.

## Quick Start

### 1. Download the binary

Download the latest release for your platform from [GitHub Releases](https://github.com/matillion/slack-mcp-server/releases).

| Platform      | File                              |
|---------------|-----------------------------------|
| macOS (Apple) | `slack-mcp_darwin_arm64.tar.gz`   |
| macOS (Intel) | `slack-mcp_darwin_amd64.tar.gz`   |
| Linux (x64)   | `slack-mcp_linux_amd64.tar.gz`    |
| Linux (ARM)   | `slack-mcp_linux_arm64.tar.gz`    |
| Windows       | `slack-mcp_windows_amd64.zip`     |

### 2. Verify the checksum

Download `checksums.txt` from the same release and verify:

```bash
# macOS / Linux
shasum -a 256 -c checksums.txt --ignore-missing

# Windows (PowerShell)
(Get-FileHash slack-mcp_windows_amd64.zip -Algorithm SHA256).Hash -eq `
  (Select-String -Path checksums.txt -Pattern "slack-mcp_windows_amd64.zip").Line.Split(" ")[0]
```

### 3. Extract and install

```bash
# macOS / Linux
tar -xzf slack-mcp_darwin_arm64.tar.gz
mv slack-mcp ~/bin/  # or anywhere in your PATH
```

### 4. Get Slack credentials

The server uses cookie-based authentication to access Slack with your user permissions.

1. Open `<workspace>.slack.com` in your browser and sign in
2. Open Developer Tools (F12) → Application → Cookies
3. Find the `d` cookie value (starts with `xoxd-`) → use as `SLACK_COOKIE`
4. In the Console tab, run:
   ```javascript
   JSON.parse(localStorage.localConfig_v2).teams[Object.keys(JSON.parse(localStorage.localConfig_v2).teams)[0]].token
   ```
5. Copy the token (starts with `xoxc-`) → use as `SLACK_TOKEN`

### 5. Configure Claude Code

Add the server to your Claude Code MCP configuration (`~/.claude.json`):

```json
{
  "mcpServers": {
    "slack": {
      "command": "/path/to/slack-mcp",
      "env": {
        "SLACK_TOKEN": "xoxc-your-token-here",
        "SLACK_COOKIE": "xoxd-your-cookie-here"
      }
    }
  }
}
```

### 6. Verify it works

Restart Claude Code and ask it to list your Slack channels.

## Available Tools

| Tool                    | Description                                               |
|-------------------------|-----------------------------------------------------------|
| `slack_list_channels`   | List channels you have access to                          |
| `slack_read_history`    | Read messages from a channel                              |
| `slack_read_thread`     | Read all replies in a thread                              |
| `slack_search_messages` | Search messages across workspace                          |
| `slack_get_user`        | Look up user by ID or email                               |
| `slack_get_permalink`   | Get permalink to a message                                |
| `slack_export_channel`  | Export channel contents (including threads) to JSON-lines |

## Configuration Reference

### Environment Variables

| Variable       | Required | Description                                            |
|----------------|----------|--------------------------------------------------------|
| `SLACK_TOKEN`  | Yes      | Slack auth token (starts with `xoxc-`)                 |
| `SLACK_COOKIE` | Yes      | Slack browser cookie (starts with `xoxd-`)             |
| `LOG_LEVEL`    | No       | `debug`, `info` (default), `warn`, or `error`          |

### Authentication Methods

| Token Type     | Environment Variables            | Use Case                                   |
|----------------|----------------------------------|--------------------------------------------|
| Bot/User OAuth | `SLACK_TOKEN` only               | Slack apps with proper OAuth scopes        |
| Browser token  | `SLACK_TOKEN` + `SLACK_COOKIE`   | Personal use without creating a Slack app  |

For bot/user OAuth tokens, you'll need these scopes:
- `channels:read`, `groups:read` - List channels
- `channels:history`, `groups:history` - Read messages
- `search:read` - Search messages
- `users:read`, `users:read.email` - Look up users

### Data Directory

The server creates `~/.claude/slack-mcp/` on startup:

```
~/.claude/slack-mcp/
├── slack-mcp.log      # Server logs (JSON, appended)
└── responses/         # Tool output files (exports, large results)
```

### Logging

Logs are written to both stderr and `~/.claude/slack-mcp/slack-mcp.log`.

| Level   | What's logged                                            |
|---------|----------------------------------------------------------|
| `debug` | Channel lookups, API calls, pagination details           |
| `info`  | Client initialization, channel searches, rate limits     |
| `warn`  | Rate limits, channels not found                          |
| `error` | Authentication failures, API errors                      |

---

## Contributing

### Prerequisites

- Go 1.24 or later
- GNU Make

### Setup

```bash
git clone https://github.com/matillion/slack-mcp-server.git
cd slack-mcp-server
make build
```

### Development Workflow

```bash
make check      # Run formatter check, vet, and tests (use before committing)
make test       # Run tests only
make cover      # Run tests with coverage report
make fmt        # Format code
make generate   # Regenerate mocks after interface changes
make clean      # Remove build artifacts
make help       # Show all available targets
```

### Running Locally

```bash
export SLACK_TOKEN="xoxc-..."
export SLACK_COOKIE="xoxd-..."
./slack-mcp
```

Test the MCP protocol manually:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | LOG_LEVEL=debug SLACK_TOKEN=fake ./slack-mcp
```

### Project Structure

```
cmd/slack-mcp/         # Main entry point
internal/mcp/          # MCP server setup and tool registration
internal/slack/        # Slack client and tool implementations
```

See [CLAUDE.md](CLAUDE.md) for code conventions, testing guidelines, and release instructions.

## License

MIT
