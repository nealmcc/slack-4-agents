package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	slackclient "github.com/nealmcconachie/slack-mcp/internal/slack"
)

func main() {
	// Create Slack client
	client, err := slackclient.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Slack client: %v", err)
	}

	// Create MCP server
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "slack-mcp",
			Version: "1.0.0",
		},
		nil,
	)

	// Register Slack tools
	client.RegisterTools(server)

	// Run on STDIO transport
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
