package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	slackclient "github.com/nealmcconachie/slack-mcp/internal/slack"
	"go.uber.org/zap"
)

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
}

// CreateServer creates an MCP server with all Slack tools registered
func CreateServer(logger *zap.Logger, handler ToolHandler) *mcp.Server {
	logger.Info("Starting MCP server")
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "slack-mcp",
			Version: "1.0.0",
		},
		nil,
	)
	registerTools(server, handler)
	logger.Info("Slack MCP server initialized, starting transport")
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
}
