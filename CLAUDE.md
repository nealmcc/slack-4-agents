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

Follow the Go convention: **accept interfaces, return structs**.

- Define interfaces in the package that uses them, not in the package that implements them
- Keep interfaces small and focused
- Functions/constructors should accept interface parameters and return concrete struct types
- Exception: `*zap.Logger` is accepted directly (not wrapped in an interface)
- Example: `ResponseWriter` is defined in `client.go` (the consumer), while `FileResponseWriter` implementation lives in `response_writer.go`

### Testing

- Use `got` and `want` for test assertions: `t.Errorf("Field: got %v, want %v", got, want)`
- Use [gomock](https://github.com/uber-go/mock) to generate mocks for interfaces
- Mock files use `_mocks.go` suffix (e.g., `client.go` → `client_mocks.go`)
- Add `//go:generate` directives above interfaces to automate mock generation:
  ```go
  //go:generate go tool mockgen -source=$GOFILE -destination=${GOFILE%.go}_mocks.go -package=slack
  type MyInterface interface {
      DoSomething() error
  }
  ```
- Run `go generate ./...` to regenerate all mocks
- Exclude mock files from coverage reports (see `make cover`)

### Project Structure

```
cmd/slack-mcp/         # Main entry point
internal/mcp/          # MCP server setup and tool registration
  mcp.go               # Server creation, ToolHandler interface
internal/slack/        # Slack client and tool implementations
  client.go            # Client struct, SlackAPI and ResponseWriter interfaces
  response_writer.go   # FileResponseWriter implementation
  tools.go             # Tool method implementations
  transport.go         # HTTP transport with cookie auth
```
