package mcp

import (
	"context"

	slackclient "github.com/matillion/slack-4-agents/internal/slack"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// errorWrappingHandler wraps a ToolHandler to provide enhanced error messages
type errorWrappingHandler struct {
	handler ToolHandler
	logger  *zap.Logger
}

func (h *errorWrappingHandler) ListChannels(ctx context.Context, req *mcp.CallToolRequest, input slackclient.ListChannelsInput) (*mcp.CallToolResult, slackclient.ListChannelsOutput, error) {
	result, output, err := h.handler.ListChannels(ctx, req, input)
	return result, output, slackclient.WrapError(h.logger, "list_channels", err)
}

func (h *errorWrappingHandler) ReadHistory(ctx context.Context, req *mcp.CallToolRequest, input slackclient.ReadHistoryInput) (*mcp.CallToolResult, slackclient.ReadHistoryOutput, error) {
	result, output, err := h.handler.ReadHistory(ctx, req, input)
	return result, output, slackclient.WrapError(h.logger, "read_history", err)
}

func (h *errorWrappingHandler) SearchMessages(ctx context.Context, req *mcp.CallToolRequest, input slackclient.SearchMessagesInput) (*mcp.CallToolResult, slackclient.SearchMessagesOutput, error) {
	result, output, err := h.handler.SearchMessages(ctx, req, input)
	return result, output, slackclient.WrapError(h.logger, "search_messages", err)
}

func (h *errorWrappingHandler) GetUser(ctx context.Context, req *mcp.CallToolRequest, input slackclient.GetUserInput) (*mcp.CallToolResult, slackclient.GetUserOutput, error) {
	result, output, err := h.handler.GetUser(ctx, req, input)
	return result, output, slackclient.WrapError(h.logger, "get_user", err)
}

func (h *errorWrappingHandler) GetPermalink(ctx context.Context, req *mcp.CallToolRequest, input slackclient.GetPermalinkInput) (*mcp.CallToolResult, slackclient.GetPermalinkOutput, error) {
	result, output, err := h.handler.GetPermalink(ctx, req, input)
	return result, output, slackclient.WrapError(h.logger, "get_permalink", err)
}

func (h *errorWrappingHandler) ReadThread(ctx context.Context, req *mcp.CallToolRequest, input slackclient.ReadThreadInput) (*mcp.CallToolResult, slackclient.ReadThreadOutput, error) {
	result, output, err := h.handler.ReadThread(ctx, req, input)
	return result, output, slackclient.WrapError(h.logger, "read_thread", err)
}

func (h *errorWrappingHandler) ExportChannel(ctx context.Context, req *mcp.CallToolRequest, input slackclient.ExportChannelInput) (*mcp.CallToolResult, slackclient.ExportChannelOutput, error) {
	result, output, err := h.handler.ExportChannel(ctx, req, input)
	return result, output, slackclient.WrapError(h.logger, "export_channel", err)
}

func (h *errorWrappingHandler) ReadCanvas(ctx context.Context, req *mcp.CallToolRequest, input slackclient.ReadCanvasInput) (*mcp.CallToolResult, slackclient.ReadCanvasOutput, error) {
	result, output, err := h.handler.ReadCanvas(ctx, req, input)
	return result, output, slackclient.WrapError(h.logger, "read_canvas", err)
}

// ToolHandler defines the interface for Slack tool operations
//
//go:generate go tool mockgen -source=$GOFILE -destination=mcp_mocks.go -package=mcp
type ToolHandler interface {
	ListChannels(ctx context.Context, req *mcp.CallToolRequest, input slackclient.ListChannelsInput) (*mcp.CallToolResult, slackclient.ListChannelsOutput, error)
	ReadHistory(ctx context.Context, req *mcp.CallToolRequest, input slackclient.ReadHistoryInput) (*mcp.CallToolResult, slackclient.ReadHistoryOutput, error)
	SearchMessages(ctx context.Context, req *mcp.CallToolRequest, input slackclient.SearchMessagesInput) (*mcp.CallToolResult, slackclient.SearchMessagesOutput, error)
	GetUser(ctx context.Context, req *mcp.CallToolRequest, input slackclient.GetUserInput) (*mcp.CallToolResult, slackclient.GetUserOutput, error)
	GetPermalink(ctx context.Context, req *mcp.CallToolRequest, input slackclient.GetPermalinkInput) (*mcp.CallToolResult, slackclient.GetPermalinkOutput, error)
	ReadThread(ctx context.Context, req *mcp.CallToolRequest, input slackclient.ReadThreadInput) (*mcp.CallToolResult, slackclient.ReadThreadOutput, error)
	ExportChannel(ctx context.Context, req *mcp.CallToolRequest, input slackclient.ExportChannelInput) (*mcp.CallToolResult, slackclient.ExportChannelOutput, error)
	ReadCanvas(ctx context.Context, req *mcp.CallToolRequest, input slackclient.ReadCanvasInput) (*mcp.CallToolResult, slackclient.ReadCanvasOutput, error)
}

// CreateServer creates an MCP server with all Slack tools registered
func CreateServer(logger *zap.Logger, handler ToolHandler) *mcp.Server {
	logger.Info("Starting MCP server")
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "slack-4-agents",
			Version: "1.0.0",
		},
		nil,
	)

	// Wrap handler to provide enhanced error messages for auth failures
	wrappedHandler := &errorWrappingHandler{handler: handler, logger: logger}
	registerTools(server, wrappedHandler)
	logger.Info("Slack 4 Agents server initialized, starting transport")
	return server
}

// registerTools registers all Slack tools with the MCP server
func registerTools(server *mcp.Server, handler ToolHandler) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_list_channels",
		Description: "List Slack channels the user has access to. Returns channel names, IDs, topics, and member counts.",
	}, handler.ListChannels)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_read_history",
		Description: "Read message history from a Slack channel or conversation. Returns messages with author info, timestamps, and thread details.",
	}, handler.ReadHistory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_search_messages",
		Description: "Search for messages across the Slack workspace. Supports Slack search syntax like from:@user, in:#channel, before:2024-01-01.",
	}, handler.SearchMessages)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_get_user",
		Description: "Look up a Slack user by ID or email address. Returns profile information including name, title, status, and timezone.",
	}, handler.GetUser)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_get_permalink",
		Description: "Get a permanent link (URL) to a specific Slack message. Useful for sharing or referencing messages.",
	}, handler.GetPermalink)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_read_thread",
		Description: "Read all replies in a Slack thread. Use the thread parent's timestamp from slack_read_history (messages with reply_count > 0).",
	}, handler.ReadThread)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_export_channel",
		Description: "Export a Slack channel's contents (including threads) to JSON-lines format. Returns a file with all messages and thread replies, reactions, and user names.",
	}, handler.ExportChannel)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_read_canvas",
		Description: "Read a Slack canvas document. Provide either a channel (to read the channel's canvas) or a file_id (for standalone canvases). Returns the canvas content as plain text.",
	}, handler.ReadCanvas)
}
