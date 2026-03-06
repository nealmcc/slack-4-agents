package slackmcp

import (
	"slices"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	goslack "github.com/slack-go/slack"
	"go.mcconachie.co/slack-4-agents/internal/slack"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zaptest"
)

func newTestClient(t *testing.T) *slack.Service {
	ctrl := gomock.NewController(t)
	api := slack.NewMockSlackAPI(ctrl)
	return slack.NewService(api, zaptest.NewLogger(t), nil)
}

func TestCreateServer_ReturnsValidServer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := newTestClient(t)

	server := NewServer(logger, client)

	if server == nil {
		t.Fatal("CreateServer returned nil")
	}
}

func TestServer_ListsAllRegisteredTools(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := newTestClient(t)

	server := NewServer(logger, client)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := t.Context()

	go func() {
		server.Run(ctx, serverTransport)
	}()

	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer session.Close()

	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	wantTools := []string{
		"slack_list_channels",
		"slack_read_history",
		"slack_search_messages",
		"slack_get_user",
		"slack_get_permalink",
		"slack_read_thread",
		"slack_export_channel",
		"slack_read_canvas",
	}

	if len(result.Tools) != len(wantTools) {
		t.Errorf("tool count: got %d, want %d", len(result.Tools), len(wantTools))
	}

	gotNames := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		gotNames[i] = tool.Name
	}

	for _, want := range wantTools {
		if !slices.Contains(gotNames, want) {
			t.Errorf("tool %q not found in registered tools: %v", want, gotNames)
		}
	}
}

func TestServer_ToolsHaveDescriptions(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := newTestClient(t)

	server := NewServer(logger, client)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := t.Context()

	go func() {
		server.Run(ctx, serverTransport)
	}()

	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer session.Close()

	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	for _, tool := range result.Tools {
		if tool.Description == "" {
			t.Errorf("tool %q has no description", tool.Name)
		}
	}
}

func TestServer_CallToolInvokesHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	api := slack.NewMockSlackAPI(ctrl)
	logger := zaptest.NewLogger(t)
	client := slack.NewService(api, logger, nil)

	api.EXPECT().
		GetUserInfoContext(gomock.Any(), "U123456789").
		Return(&goslack.User{
			ID:       "U123456789",
			Name:     "testuser",
			RealName: "Test User",
		}, nil)

	server := NewServer(logger, client)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx := t.Context()

	go func() {
		server.Run(ctx, serverTransport)
	}()

	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "slack_get_user",
		Arguments: map[string]any{
			"user": "U123456789",
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result.IsError {
		t.Errorf("tool call returned error: %v", result.Content)
	}
}
