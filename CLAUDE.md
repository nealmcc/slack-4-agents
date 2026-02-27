# Slack 4 Agents

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
cmd/slack-4-agents/         # Main entry point
internal/mcp/          # MCP server setup and tool registration
  mcp.go               # Server creation, ToolHandler interface
internal/slack/        # Slack client and tool implementations
  client.go            # Client struct, SlackAPI and ResponseWriter interfaces
  response_writer.go   # FileResponseWriter implementation
  tools.go             # Tool method implementations
  transport.go         # HTTP transport with cookie auth
```

## Releasing

Releases are managed with [GoReleaser](https://goreleaser.com/). Binaries are built for darwin (arm64/amd64), linux (arm64/amd64), and windows (amd64).

### Create a release

```bash
git tag v1.0.0
git push origin v1.0.0
goreleaser release --clean
```

This builds binaries, generates checksums, and uploads everything to a GitHub Release.

### Test locally (no upload)

```bash
goreleaser build --snapshot --clean
```

Binaries are output to `dist/`.

### Install goreleaser

```bash
go install github.com/goreleaser/goreleaser/v2@latest
```
