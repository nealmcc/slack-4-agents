# Slack MCP Server

A Go-based MCP server providing Claude Code with read-only Slack access.

## Build & Test

```bash
make build      # Build binary
make test       # Run tests
make check      # Run fmt, vet, and test
make install    # Install to ~/bin
```

## Code Conventions

### Interface Definition

Follow the Go convention: **consumers define the interfaces they require**.

- Define interfaces in the package that uses them, not in the package that implements them
- Keep interfaces small and focused
- Example: `ResponseWriter` is defined in `client.go` (the consumer), while `FileResponseWriter` implementation lives in `cache.go`

### Project Structure

```
cmd/slack-mcp/     # Main entry point
internal/slack/    # Slack client and MCP tools
  client.go        # Client struct, interfaces it consumes
  cache.go         # ResponseWriter implementation (FileResponseWriter)
  tools.go         # MCP tool implementations
  transport.go     # HTTP transport with cookie auth
```
