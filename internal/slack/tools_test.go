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

	client, logger, responsesDir := newTestClient(t, mock)
	_ = logger
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ListChannelsInput{
		Limit: 10,
	}

	_, output, err := client.ListChannels(ctx, nil, input)
	if err != nil {
		t.Fatalf("ListChannels failed: %v", err)
	}

	if output.TotalCount != 2 {
		t.Errorf("TotalCount: got %d, want 2", output.TotalCount)
	}

	if output.FirstChannel == nil {
		t.Fatal("FirstChannel: got nil, want non-nil")
	}

	if output.FirstChannel.Name != "general" {
		t.Errorf("FirstChannel.Name: got %q, want %q", output.FirstChannel.Name, "general")
	}

	if output.FirstChannel.ID != "C123456789" {
		t.Errorf("FirstChannel.ID: got %q, want %q", output.FirstChannel.ID, "C123456789")
	}

	if output.FirstChannel.MemberCount != 100 {
		t.Errorf("FirstChannel.MemberCount: got %d, want 100", output.FirstChannel.MemberCount)
	}

	if output.LastChannel == nil {
		t.Fatal("LastChannel: got nil, want non-nil")
	}

	if output.LastChannel.Name != "random" {
		t.Errorf("LastChannel.Name: got %q, want %q", output.LastChannel.Name, "random")
	}

	// Verify FileRef metadata
	if output.File.Path == "" {
		t.Error("File.Path: got empty, want non-empty")
	}
	if output.File.Name == "" {
		t.Error("File.Name: got empty, want non-empty")
	}
	if output.File.Bytes == 0 {
		t.Error("File.Bytes: got 0, want non-zero")
	}
	if output.File.Lines == 0 {
		t.Error("File.Lines: got 0, want non-zero")
	}

	// Verify response file was created and contains correct data
	data, err := os.ReadFile(output.File.Path)
	if err != nil {
		t.Fatalf("Failed to read response file: %v", err)
	}

	if int64(len(data)) != output.File.Bytes {
		t.Errorf("File.Bytes: got %d, want %d", output.File.Bytes, len(data))
	}

	var channels []ChannelInfo
	if err := json.Unmarshal(data, &channels); err != nil {
		t.Fatalf("Failed to unmarshal response file: %v", err)
	}

	if len(channels) != 2 {
		t.Errorf("channels in response file: got %d, want 2", len(channels))
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

	client, logger, responsesDir := newTestClient(t, mock)
	_ = logger
	defer os.RemoveAll(responsesDir)

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
		t.Errorf("ChannelID: got %q, want %q", output.ChannelID, "C123456789")
	}

	if len(output.Messages) != 2 {
		t.Errorf("len(Messages): got %d, want 2", len(output.Messages))
	}

	if output.Messages[0].Text != "Hello world" {
		t.Errorf("Messages[0].Text: got %q, want %q", output.Messages[0].Text, "Hello world")
	}

	if output.Messages[0].UserName != "alice" {
		t.Errorf("Messages[0].UserName: got %q, want %q", output.Messages[0].UserName, "alice")
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

	client, logger, responsesDir := newTestClient(t, mock)
	_ = logger
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := GetUserInput{
		User: "U123456789",
	}

	_, output, err := client.GetUser(ctx, nil, input)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if output.User.ID != "U123456789" {
		t.Errorf("User.ID: got %q, want %q", output.User.ID, "U123456789")
	}

	if output.User.Name != "alice" {
		t.Errorf("User.Name: got %q, want %q", output.User.Name, "alice")
	}

	if output.User.Email != "alice@example.com" {
		t.Errorf("User.Email: got %q, want %q", output.User.Email, "alice@example.com")
	}

	if !output.User.IsAdmin {
		t.Error("User.IsAdmin: got false, want true")
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

	client, logger, responsesDir := newTestClient(t, mock)
	_ = logger
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := GetPermalinkInput{
		Channel:   "C123456789",
		Timestamp: "1234567890.123456",
	}

	_, output, err := client.GetPermalink(ctx, nil, input)
	if err != nil {
		t.Fatalf("GetPermalink failed: %v", err)
	}

	wantPermalink := "https://example.slack.com/archives/C123456789/p1234567890123456"
	if output.Permalink != wantPermalink {
		t.Errorf("Permalink: got %q, want %q", output.Permalink, wantPermalink)
	}

	if output.Channel != "C123456789" {
		t.Errorf("Channel: got %q, want %q", output.Channel, "C123456789")
	}
}
