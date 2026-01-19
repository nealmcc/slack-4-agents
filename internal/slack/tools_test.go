package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/slack-go/slack"
)

// Helper to create a test client with mocked HTTP server
func newTestClient(t *testing.T, mock *mockSlackServer) (*Client, *testLogger, string) {
	t.Helper()

	// Create a Slack client that points to our mock server
	api := slack.New("xoxb-test-token",
		slack.OptionAPIURL(mock.server.URL+"/"),
	)

	// Create temp directory for response files
	outputDir, err := os.MkdirTemp("", "slack-mcp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	logger := newTestLogger()
	responses := NewFileResponseWriter(outputDir)
	return newClientWithAPI(api, logger.Logger, responses), logger, outputDir
}

func TestListChannels(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	// Mock conversations.list response
	mock.addHandler("/conversations.list", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{
					"id":          "C123456789",
					"name":        "general",
					"topic":       map[string]string{"value": "General discussion"},
					"purpose":     map[string]string{"value": "Company-wide announcements"},
					"num_members": 100,
					"is_private":  false,
					"is_archived": false,
				},
				{
					"id":          "C987654321",
					"name":        "random",
					"topic":       map[string]string{"value": "Random chat"},
					"purpose":     map[string]string{"value": "Non-work discussions"},
					"num_members": 50,
					"is_private":  false,
					"is_archived": false,
				},
			},
			"response_metadata": map[string]string{
				"next_cursor": "",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	client, logger, cacheDir := newTestClient(t, mock)
	_ = logger // Can be used for log assertions
	defer os.RemoveAll(cacheDir)

	// Test the ListChannels tool
	ctx := context.Background()
	input := ListChannelsInput{
		Limit: 10,
	}

	_, output, err := client.ListChannels(ctx, nil, input)
	if err != nil {
		t.Fatalf("ListChannels failed: %v", err)
	}

	if output.TotalCount != 2 {
		t.Errorf("Expected 2 channels, got %d", output.TotalCount)
	}

	if output.FirstChannel == nil {
		t.Fatal("Expected first channel to be set")
	}

	if output.FirstChannel.Name != "general" {
		t.Errorf("Expected first channel to be 'general', got '%s'", output.FirstChannel.Name)
	}

	if output.FirstChannel.ID != "C123456789" {
		t.Errorf("Expected first channel ID to be 'C123456789', got '%s'", output.FirstChannel.ID)
	}

	if output.FirstChannel.MemberCount != 100 {
		t.Errorf("Expected first channel member count to be 100, got %d", output.FirstChannel.MemberCount)
	}

	if output.LastChannel == nil {
		t.Fatal("Expected last channel to be set")
	}

	if output.LastChannel.Name != "random" {
		t.Errorf("Expected last channel to be 'random', got '%s'", output.LastChannel.Name)
	}

	// Verify FileRef metadata
	if output.File.Path == "" {
		t.Error("Expected file path to be set")
	}
	if output.File.Name == "" {
		t.Error("Expected file name to be set")
	}
	if output.File.Bytes == 0 {
		t.Error("Expected file bytes to be non-zero")
	}
	if output.File.Lines == 0 {
		t.Error("Expected file lines to be non-zero")
	}

	// Verify response file was created and contains expected data
	data, err := os.ReadFile(output.File.Path)
	if err != nil {
		t.Fatalf("Failed to read response file: %v", err)
	}

	if int64(len(data)) != output.File.Bytes {
		t.Errorf("File size mismatch: got %d bytes, FileRef says %d", len(data), output.File.Bytes)
	}

	var channels []ChannelInfo
	if err := json.Unmarshal(data, &channels); err != nil {
		t.Fatalf("Failed to unmarshal response file: %v", err)
	}

	if len(channels) != 2 {
		t.Errorf("Expected 2 channels in response file, got %d", len(channels))
	}
}

func TestReadHistory(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	// Mock conversations.info (for channel ID validation)
	mock.addHandler("/conversations.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"channel": map[string]interface{}{
				"id":   "C123456789",
				"name": "general",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Mock conversations.history
	mock.addHandler("/conversations.history", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{
					"type":      "message",
					"user":      "U123456789",
					"text":      "Hello world",
					"ts":        "1234567890.123456",
					"thread_ts": "",
				},
				{
					"type":      "message",
					"user":      "U987654321",
					"text":      "Hi there",
					"ts":        "1234567891.123456",
					"thread_ts": "",
				},
			},
			"has_more": false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Mock users.info (for user name lookup)
	mock.addHandler("/users.info", func(w http.ResponseWriter, r *http.Request) {
		// Parse form data for POST requests (Slack API uses POST with form data)
		r.ParseForm()
		userID := r.FormValue("user")
		if userID == "" {
			userID = r.URL.Query().Get("user")
		}
		userName := "user"
		if userID == "U123456789" {
			userName = "alice"
		} else if userID == "U987654321" {
			userName = "bob"
		}

		response := map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id":   userID,
				"name": userName,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	client, logger, cacheDir := newTestClient(t, mock)
	_ = logger // Can be used for log assertions
	defer os.RemoveAll(cacheDir)

	// Test the ReadHistory tool with channel ID
	ctx := context.Background()
	input := ReadHistoryInput{
		Channel: "C123456789",
		Limit:   10,
	}

	_, output, err := client.ReadHistory(ctx, nil, input)
	if err != nil {
		t.Fatalf("ReadHistory failed: %v", err)
	}

	if output.ChannelID != "C123456789" {
		t.Errorf("Expected channel ID 'C123456789', got '%s'", output.ChannelID)
	}

	if len(output.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(output.Messages))
	}

	if output.Messages[0].Text != "Hello world" {
		t.Errorf("Expected first message text 'Hello world', got '%s'", output.Messages[0].Text)
	}

	if output.Messages[0].UserName != "alice" {
		t.Errorf("Expected first message user name 'alice', got '%s'", output.Messages[0].UserName)
	}
}

func TestGetUser(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	// Mock users.info
	mock.addHandler("/users.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id":        "U123456789",
				"name":      "alice",
				"real_name": "Alice Smith",
				"profile": map[string]interface{}{
					"display_name": "Alice",
					"email":        "alice@example.com",
					"title":        "Engineer",
					"status_text":  "Working",
					"status_emoji": ":computer:",
				},
				"is_bot":   false,
				"is_admin": true,
				"tz":       "America/Los_Angeles",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	client, logger, cacheDir := newTestClient(t, mock)
	_ = logger // Can be used for log assertions
	defer os.RemoveAll(cacheDir)

	// Test the GetUser tool
	ctx := context.Background()
	input := GetUserInput{
		User: "U123456789",
	}

	_, output, err := client.GetUser(ctx, nil, input)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if output.User.ID != "U123456789" {
		t.Errorf("Expected user ID 'U123456789', got '%s'", output.User.ID)
	}

	if output.User.Name != "alice" {
		t.Errorf("Expected user name 'alice', got '%s'", output.User.Name)
	}

	if output.User.Email != "alice@example.com" {
		t.Errorf("Expected user email 'alice@example.com', got '%s'", output.User.Email)
	}

	if !output.User.IsAdmin {
		t.Error("Expected user to be admin")
	}
}

func TestGetPermalink(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	// Mock conversations.info (for channel ID validation)
	mock.addHandler("/conversations.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"channel": map[string]interface{}{
				"id":   "C123456789",
				"name": "general",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Mock chat.getPermalink
	mock.addHandler("/chat.getPermalink", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok":        true,
			"permalink": "https://example.slack.com/archives/C123456789/p1234567890123456",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	client, logger, cacheDir := newTestClient(t, mock)
	_ = logger // Can be used for log assertions
	defer os.RemoveAll(cacheDir)

	// Test the GetPermalink tool
	ctx := context.Background()
	input := GetPermalinkInput{
		Channel:   "C123456789",
		Timestamp: "1234567890.123456",
	}

	_, output, err := client.GetPermalink(ctx, nil, input)
	if err != nil {
		t.Fatalf("GetPermalink failed: %v", err)
	}

	expectedPermalink := "https://example.slack.com/archives/C123456789/p1234567890123456"
	if output.Permalink != expectedPermalink {
		t.Errorf("Expected permalink '%s', got '%s'", expectedPermalink, output.Permalink)
	}

	if output.Channel != "C123456789" {
		t.Errorf("Expected channel 'C123456789', got '%s'", output.Channel)
	}
}
