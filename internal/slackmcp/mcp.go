package slackmcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mcconachie.co/slack-4-agents/internal/slack"
	"go.uber.org/zap"
)

// NewServer creates an MCP server with all Slack tools registered
func NewServer(logger *zap.Logger, client *slack.Service) *mcp.Server {
	logger.Info("Starting MCP server")
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "slack-4-agents",
			Version: "1.0.0",
		},
		nil,
	)

	registerTools(server, client, logger)
	logger.Info("Slack 4 Agents server initialized, starting transport")
	return server
}

// registerTools registers all Slack tools with the MCP server
func registerTools(server *mcp.Server, client *slack.Service, logger *zap.Logger) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_list_channels",
		Description: "List Slack channels the user has access to. Returns channel names, IDs, topics, and member counts.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input slack.ListChannelsInput) (*mcp.CallToolResult, slack.ListChannelsOutput, error) {
		output, err := client.ListChannels(ctx, input)
		return nil, output, slack.WrapError(logger, "list_channels", err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_read_history",
		Description: "Read message history from a Slack channel or conversation. Returns messages with author info, timestamps, and thread details.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input slack.ReadHistoryInput) (*mcp.CallToolResult, slack.ReadHistoryOutput, error) {
		output, err := client.ReadHistory(ctx, input)
		return nil, output, slack.WrapError(logger, "read_history", err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_search_messages",
		Description: "Search for messages across the Slack workspace. Supports Slack search syntax like from:@user, in:#channel, before:2024-01-01.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input slack.SearchMessagesInput) (*mcp.CallToolResult, slack.SearchMessagesOutput, error) {
		output, err := client.SearchMessages(ctx, input)
		return nil, output, slack.WrapError(logger, "search_messages", err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_get_user",
		Description: "Look up a Slack user by ID or email address. Returns profile information including name, title, status, and timezone.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input slack.GetUserInput) (*mcp.CallToolResult, slack.GetUserOutput, error) {
		output, err := client.GetUser(ctx, input)
		return nil, output, slack.WrapError(logger, "get_user", err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_get_permalink",
		Description: "Get a permanent link (URL) to a specific Slack message. Useful for sharing or referencing messages.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input slack.GetPermalinkInput) (*mcp.CallToolResult, slack.GetPermalinkOutput, error) {
		output, err := client.GetPermalink(ctx, input)
		return nil, output, slack.WrapError(logger, "get_permalink", err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_read_thread",
		Description: "Read all replies in a Slack thread. Use the thread parent's timestamp from slack_read_history (messages with reply_count > 0).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input slack.ReadThreadInput) (*mcp.CallToolResult, slack.ReadThreadOutput, error) {
		output, err := client.ReadThread(ctx, input)
		return nil, output, slack.WrapError(logger, "read_thread", err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_export_channel",
		Description: "Export a Slack channel's contents (including threads) to JSON-lines format. Returns a file with all messages and thread replies, reactions, and user names.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input slack.ExportChannelInput) (*mcp.CallToolResult, slack.ExportChannelOutput, error) {
		output, err := client.ExportChannel(ctx, input)
		return nil, output, slack.WrapError(logger, "export_channel", err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_read_canvas",
		Description: "Read a Slack canvas document. Provide either a channel (to read the channel's canvas) or a file_id (for standalone canvases). Returns the canvas content as plain text.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input slack.ReadCanvasInput) (*mcp.CallToolResult, slack.ReadCanvasOutput, error) {
		output, err := client.ReadCanvas(ctx, input)
		return nil, output, slack.WrapError(logger, "read_canvas", err)
	})
}
