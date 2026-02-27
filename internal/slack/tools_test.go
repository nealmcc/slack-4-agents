package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/slack-go/slack"
)

func TestProcessReactions(t *testing.T) {
	reactions := []slack.ItemReaction{
		{Name: "thumbsup", Count: 3},
		{Name: "heart", Count: 2},
	}

	got := processReactions(reactions)

	if len(got) != 2 {
		t.Fatalf("len(result): got %d, want 2", len(got))
	}

	if got[0].Name != "thumbsup" || got[0].Count != 3 {
		t.Errorf("got[0]: got %+v, want {Name:thumbsup Count:3}", got[0])
	}

	if got[1].Name != "heart" || got[1].Count != 2 {
		t.Errorf("got[1]: got %+v, want {Name:heart Count:2}", got[1])
	}
}

func TestProcessReactions_Empty(t *testing.T) {
	tests := []struct {
		name      string
		reactions []slack.ItemReaction
	}{
		{"nil slice", nil},
		{"empty slice", []slack.ItemReaction{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processReactions(tt.reactions)
			if got != nil {
				t.Errorf("got %v, want nil", got)
			}
		})
	}
}

func TestBuildExportMessage(t *testing.T) {
	msg := slack.Message{
		Msg: slack.Msg{
			Timestamp:  "1234567890.123456",
			User:       "U123456789",
			Text:       "Hello world",
			ReplyCount: 5,
			Reactions: []slack.ItemReaction{
				{Name: "thumbsup", Count: 2},
			},
		},
	}

	got := buildExportMessage(msg, "", "alice")

	if got.Timestamp != "1234567890.123456" {
		t.Errorf("Timestamp: got %q, want %q", got.Timestamp, "1234567890.123456")
	}
	if got.User != "U123456789" {
		t.Errorf("User: got %q, want %q", got.User, "U123456789")
	}
	if got.UserName != "alice" {
		t.Errorf("UserName: got %q, want %q", got.UserName, "alice")
	}
	if got.Text != "Hello world" {
		t.Errorf("Text: got %q, want %q", got.Text, "Hello world")
	}
	if got.ThreadTimestamp != "" {
		t.Errorf("ThreadTimestamp: got %q, want empty", got.ThreadTimestamp)
	}
	if got.ReplyCount != 5 {
		t.Errorf("ReplyCount: got %d, want 5", got.ReplyCount)
	}
	if len(got.Reactions) != 1 {
		t.Errorf("len(Reactions): got %d, want 1", len(got.Reactions))
	}
}

func TestBuildExportMessage_ThreadReply(t *testing.T) {
	msg := slack.Message{
		Msg: slack.Msg{
			Timestamp: "1234567891.123456",
			User:      "U987654321",
			Text:      "This is a reply",
		},
	}

	got := buildExportMessage(msg, "1234567890.123456", "bob")

	if got.ThreadTimestamp != "1234567890.123456" {
		t.Errorf("ThreadTimestamp: got %q, want %q", got.ThreadTimestamp, "1234567890.123456")
	}
	if got.UserName != "bob" {
		t.Errorf("UserName: got %q, want %q", got.UserName, "bob")
	}
}

func TestExportStats(t *testing.T) {
	stats := newExportStats()

	stats.trackUser("U123")
	stats.trackUser("U456")
	stats.trackUser("U123")

	if len(stats.uniqueUsers) != 2 {
		t.Errorf("unique users: got %d, want 2", len(stats.uniqueUsers))
	}

	stats.addReactions([]slack.ItemReaction{
		{Name: "thumbsup", Count: 3},
		{Name: "heart", Count: 2},
	})

	if stats.reactionCount != 5 {
		t.Errorf("reactionCount: got %d, want 5", stats.reactionCount)
	}

	stats.addReactions([]slack.ItemReaction{
		{Name: "fire", Count: 1},
	})

	if stats.reactionCount != 6 {
		t.Errorf("reactionCount after second add: got %d, want 6", stats.reactionCount)
	}
}

func TestIsChannelID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid C channel", "C123456789", true},
		{"valid D channel (DM)", "D123456789", true},
		{"valid G channel (group)", "G123456789", true},
		{"longer valid ID", "C12345678901", true},
		{"too short", "C12345", false},
		{"starts with lowercase", "c123456789", false},
		{"starts with invalid letter", "X123456789", false},
		{"contains lowercase", "C12345678a", false},
		{"contains special char", "C12345678-", false},
		{"channel name", "general", false},
		{"channel name with hash", "#general", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isChannelID(tt.input)
			if got != tt.want {
				t.Errorf("isChannelID(%q): got %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTimestamp_String(t *testing.T) {
	tests := []struct {
		name  string
		input Timestamp
		want  string
	}{
		{"standard slack timestamp", Timestamp("1234567890.123456"), "2009-02-13T23:31:30Z"},
		{"timestamp without microseconds", Timestamp("1234567890"), "2009-02-13T23:31:30Z"},
		{"empty timestamp", Timestamp(""), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.String()
			if got != tt.want {
				t.Errorf("Timestamp(%q).String(): want %q; got %q", tt.input, tt.want, got)
			}
		})
	}
}

func TestTimestamp_Raw(t *testing.T) {
	ts := Timestamp("1234567890.123456")
	want := "1234567890.123456"
	if got := ts.Raw(); got != want {
		t.Errorf("Raw(): want %q; got %q", want, got)
	}
}

func TestTimestamp_MarshalJSON(t *testing.T) {
	ts := Timestamp("1234567890.123456")
	got, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	want := `"2009-02-13T23:31:30Z"`
	if string(got) != want {
		t.Errorf("MarshalJSON: want %s; got %s", want, got)
	}
}

func TestTimestamp_MarshalJSON_InStruct(t *testing.T) {
	type testStruct struct {
		Timestamp Timestamp `json:"ts"`
	}
	s := testStruct{Timestamp: Timestamp("1234567890.123456")}
	got, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	want := `{"ts":"2009-02-13T23:31:30Z"}`
	if string(got) != want {
		t.Errorf("MarshalJSON: want %s; got %s", want, got)
	}
}

// Helper to create a test client with mocked HTTP server
func newTestClient(t *testing.T, mock *mockSlackServer) (*Client, *testLogger, string) {
	t.Helper()

	// Create a Slack client that points to our mock server
	api := slack.New("xoxb-test-token",
		slack.OptionAPIURL(mock.server.URL+"/"),
	)

	// Create temp directory for response files
	outputDir, err := os.MkdirTemp("", "slack-4-agents-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	logger := newTestLogger()
	responses := NewFileResponseWriter(outputDir)
	return newClientWithAPI(api, nil, logger.Logger, responses), logger, outputDir
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

func TestSearchMessages(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	// Mock search.messages
	mock.addHandler("/search.messages", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"messages": map[string]interface{}{
				"total": 2,
				"matches": []map[string]interface{}{
					{
						"ts":        "1234567890.123456",
						"channel":   map[string]interface{}{"name": "general"},
						"user":      "U123456789",
						"username":  "alice",
						"text":      "Hello world",
						"permalink": "https://example.slack.com/archives/C123/p1234567890123456",
					},
					{
						"ts":        "1234567891.123456",
						"channel":   map[string]interface{}{"name": "random"},
						"user":      "U987654321",
						"username":  "bob",
						"text":      "Hi there",
						"permalink": "https://example.slack.com/archives/C456/p1234567891123456",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := SearchMessagesInput{
		Query: "hello",
		Count: 10,
		Sort:  "timestamp",
	}

	_, output, err := client.SearchMessages(ctx, nil, input)
	if err != nil {
		t.Fatalf("SearchMessages failed: %v", err)
	}

	wantQuery := "hello"
	if got := output.Query; got != wantQuery {
		t.Errorf("Query: got %q, want %q", got, wantQuery)
	}

	wantTotal := 2
	if got := output.Total; got != wantTotal {
		t.Errorf("Total: got %d, want %d", got, wantTotal)
	}

	wantMatchCount := 2
	if got := len(output.Matches); got != wantMatchCount {
		t.Errorf("len(Matches): got %d, want %d", got, wantMatchCount)
	}

	wantText := "Hello world"
	if got := output.Matches[0].Text; got != wantText {
		t.Errorf("Matches[0].Text: got %q, want %q", got, wantText)
	}

	wantUserName := "alice"
	if got := output.Matches[0].UserName; got != wantUserName {
		t.Errorf("Matches[0].UserName: got %q, want %q", got, wantUserName)
	}

	wantChannel := "general"
	if got := output.Matches[0].Channel; got != wantChannel {
		t.Errorf("Matches[0].Channel: got %q, want %q", got, wantChannel)
	}
}

func TestExportChannel_BasicMessages(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

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

	// Mock returns messages newest-first (like real Slack API)
	mock.addHandler("/conversations.history", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{
					"type":        "message",
					"user":        "U987654321",
					"text":        "Hi there",
					"ts":          "1704067201.000001",
					"reply_count": 0,
				},
				{
					"type":        "message",
					"user":        "U123456789",
					"text":        "Hello world",
					"ts":          "1704067200.000001",
					"reply_count": 0,
				},
			},
			"has_more":          false,
			"response_metadata": map[string]string{"next_cursor": ""},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

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

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ExportChannelInput{
		Channel: "C123456789",
	}

	_, output, err := client.ExportChannel(ctx, nil, input)
	if err != nil {
		t.Fatalf("ExportChannel failed: %v", err)
	}

	if output.ChannelID != "C123456789" {
		t.Errorf("ChannelID: got %q, want %q", output.ChannelID, "C123456789")
	}

	if output.MessageCount != 2 {
		t.Errorf("MessageCount: got %d, want 2", output.MessageCount)
	}

	if output.UniqueUsers != 2 {
		t.Errorf("UniqueUsers: got %d, want 2", output.UniqueUsers)
	}

	if output.File.Lines != 2 {
		t.Errorf("File.Lines: got %d, want 2", output.File.Lines)
	}

	data, err := os.ReadFile(output.File.Path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("File lines: got %d, want 2", len(lines))
	}

	// After reversal, messages should be in chronological order (oldest first)
	// alice's message (ts=1704067200) should be first
	var msg ExportMessage
	if err := json.Unmarshal([]byte(lines[0]), &msg); err != nil {
		t.Fatalf("Failed to unmarshal first line: %v", err)
	}

	if msg.UserName != "alice" {
		t.Errorf("First message UserName: got %q, want %q", msg.UserName, "alice")
	}

	// Verify timestamps are ISO 8601 format
	if !strings.HasPrefix(string(msg.Timestamp), "2024-") {
		t.Errorf("Timestamp not in ISO format: got %q", msg.Timestamp)
	}
}

func TestExportChannel_WithThreads(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

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

	mock.addHandler("/conversations.history", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{
					"type":        "message",
					"user":        "U123456789",
					"text":        "Thread parent",
					"ts":          "1704067200.000001",
					"reply_count": 2,
				},
			},
			"has_more":          false,
			"response_metadata": map[string]string{"next_cursor": ""},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	mock.addHandler("/conversations.replies", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{
					"type":      "message",
					"user":      "U123456789",
					"text":      "Thread parent",
					"ts":        "1704067200.000001",
					"thread_ts": "1704067200.000001",
				},
				{
					"type":      "message",
					"user":      "U987654321",
					"text":      "First reply",
					"ts":        "1704067201.000001",
					"thread_ts": "1704067200.000001",
				},
				{
					"type":      "message",
					"user":      "U123456789",
					"text":      "Second reply",
					"ts":        "1704067202.000001",
					"thread_ts": "1704067200.000001",
				},
			},
			"has_more": false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

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

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ExportChannelInput{
		Channel: "C123456789",
	}

	_, output, err := client.ExportChannel(ctx, nil, input)
	if err != nil {
		t.Fatalf("ExportChannel failed: %v", err)
	}

	// Main file only contains top-level message (1 line)
	// Thread replies (2) are in a separate file
	if output.MessageCount != 3 {
		t.Errorf("MessageCount: got %d, want 3", output.MessageCount)
	}

	if output.ThreadCount != 1 {
		t.Errorf("ThreadCount: got %d, want 1", output.ThreadCount)
	}

	// Main file should have only the parent message
	mainData, err := os.ReadFile(output.File.Path)
	if err != nil {
		t.Fatalf("Failed to read main file: %v", err)
	}

	mainLines := strings.Split(strings.TrimSuffix(string(mainData), "\n"), "\n")
	if len(mainLines) != 1 {
		t.Fatalf("Main file lines: got %d, want 1", len(mainLines))
	}

	// Verify thread files were created
	if len(output.ThreadFiles) != 1 {
		t.Fatalf("ThreadFiles: got %d, want 1", len(output.ThreadFiles))
	}

	// Read thread file
	threadData, err := os.ReadFile(output.ThreadFiles[0].Path)
	if err != nil {
		t.Fatalf("Failed to read thread file: %v", err)
	}

	threadLines := strings.Split(strings.TrimSuffix(string(threadData), "\n"), "\n")
	if len(threadLines) != 3 {
		t.Fatalf("Thread file lines: got %d, want 3", len(threadLines))
	}

	// Second line should be first reply with thread_ts
	var reply ExportMessage
	if err := json.Unmarshal([]byte(threadLines[1]), &reply); err != nil {
		t.Fatalf("Failed to unmarshal second line: %v", err)
	}

	// ThreadTimestamp is now ISO formatted
	if !strings.HasPrefix(string(reply.ThreadTimestamp), "2024-") {
		t.Errorf("Reply ThreadTimestamp not ISO format: got %q", reply.ThreadTimestamp)
	}
}

func TestExportChannel_WithReactions(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

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

	mock.addHandler("/conversations.history", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{
					"type":        "message",
					"user":        "U123456789",
					"text":        "Great idea!",
					"ts":          "1704067200.000001",
					"reply_count": 0,
					"reactions": []map[string]interface{}{
						{"name": "thumbsup", "count": 3},
						{"name": "heart", "count": 2},
					},
				},
			},
			"has_more":          false,
			"response_metadata": map[string]string{"next_cursor": ""},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	mock.addHandler("/users.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id":   "U123456789",
				"name": "alice",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ExportChannelInput{
		Channel: "C123456789",
	}

	_, output, err := client.ExportChannel(ctx, nil, input)
	if err != nil {
		t.Fatalf("ExportChannel failed: %v", err)
	}

	if output.ReactionCount != 5 {
		t.Errorf("ReactionCount: got %d, want 5", output.ReactionCount)
	}

	data, err := os.ReadFile(output.File.Path)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var msg ExportMessage
	if err := json.Unmarshal(data[:len(data)-1], &msg); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if len(msg.Reactions) != 2 {
		t.Errorf("Reactions: got %d, want 2", len(msg.Reactions))
	}

	if msg.Reactions[0].Name != "thumbsup" || msg.Reactions[0].Count != 3 {
		t.Errorf("First reaction: got %+v, want {Name:thumbsup Count:3}", msg.Reactions[0])
	}
}

func TestExportChannel_TimeRange(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	var receivedOldest, receivedLatest string

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

	mock.addHandler("/conversations.history", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		receivedOldest = r.FormValue("oldest")
		receivedLatest = r.FormValue("latest")

		response := map[string]interface{}{
			"ok":                true,
			"messages":          []map[string]interface{}{},
			"has_more":          false,
			"response_metadata": map[string]string{"next_cursor": ""},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ExportChannelInput{
		Channel: "C123456789",
		Oldest:  "1704067200.000000",
		Latest:  "1704153600.000000",
	}

	_, _, err := client.ExportChannel(ctx, nil, input)
	if err != nil {
		t.Fatalf("ExportChannel failed: %v", err)
	}

	if receivedOldest != "1704067200.000000" {
		t.Errorf("Oldest: got %q, want %q", receivedOldest, "1704067200.000000")
	}

	if receivedLatest != "1704153600.000000" {
		t.Errorf("Latest: got %q, want %q", receivedLatest, "1704153600.000000")
	}
}

func TestExportChannel_EmptyChannel(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	mock.addHandler("/conversations.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"channel": map[string]interface{}{
				"id":   "C123456789",
				"name": "empty-channel",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	mock.addHandler("/conversations.history", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok":                true,
			"messages":          []map[string]interface{}{},
			"has_more":          false,
			"response_metadata": map[string]string{"next_cursor": ""},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ExportChannelInput{
		Channel: "C123456789",
	}

	_, output, err := client.ExportChannel(ctx, nil, input)
	if err != nil {
		t.Fatalf("ExportChannel failed: %v", err)
	}

	if output.MessageCount != 0 {
		t.Errorf("MessageCount: got %d, want 0", output.MessageCount)
	}

	if output.ThreadCount != 0 {
		t.Errorf("ThreadCount: got %d, want 0", output.ThreadCount)
	}

	if output.ReactionCount != 0 {
		t.Errorf("ReactionCount: got %d, want 0", output.ReactionCount)
	}

	if output.UniqueUsers != 0 {
		t.Errorf("UniqueUsers: got %d, want 0", output.UniqueUsers)
	}

	if output.File.Lines != 0 {
		t.Errorf("File.Lines: got %d, want 0", output.File.Lines)
	}
}

func TestExportChannel_Pagination(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

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

	pageCount := 0
	mock.addHandler("/conversations.history", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		cursor := r.FormValue("cursor")

		pageCount++
		var response map[string]interface{}

		if cursor == "" {
			response = map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{"type": "message", "user": "U123456789", "text": "Message 1", "ts": "1704067200.000001", "reply_count": 0},
					{"type": "message", "user": "U123456789", "text": "Message 2", "ts": "1704067200.000002", "reply_count": 0},
				},
				"has_more":          true,
				"response_metadata": map[string]string{"next_cursor": "page2"},
			}
		} else {
			response = map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{"type": "message", "user": "U123456789", "text": "Message 3", "ts": "1704067200.000003", "reply_count": 0},
				},
				"has_more":          false,
				"response_metadata": map[string]string{"next_cursor": ""},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	mock.addHandler("/users.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok":   true,
			"user": map[string]interface{}{"id": "U123456789", "name": "alice"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ExportChannelInput{
		Channel: "C123456789",
	}

	_, output, err := client.ExportChannel(ctx, nil, input)
	if err != nil {
		t.Fatalf("ExportChannel failed: %v", err)
	}

	if pageCount != 2 {
		t.Errorf("Page count: got %d, want 2", pageCount)
	}

	if output.MessageCount != 3 {
		t.Errorf("MessageCount: got %d, want 3", output.MessageCount)
	}
}

func TestReadThread(t *testing.T) {
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

	// Mock conversations.replies
	mock.addHandler("/conversations.replies", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{
					"type":      "message",
					"user":      "U123456789",
					"text":      "Thread parent message",
					"ts":        "1234567890.123456",
					"thread_ts": "1234567890.123456",
				},
				{
					"type":      "message",
					"user":      "U987654321",
					"text":      "First reply",
					"ts":        "1234567891.123456",
					"thread_ts": "1234567890.123456",
				},
				{
					"type":      "message",
					"user":      "U123456789",
					"text":      "Second reply",
					"ts":        "1234567892.123456",
					"thread_ts": "1234567890.123456",
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

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ReadThreadInput{
		Channel:   "C123456789",
		Timestamp: "1234567890.123456",
		Limit:     50,
	}

	_, output, err := client.ReadThread(ctx, nil, input)
	if err != nil {
		t.Fatalf("ReadThread failed: %v", err)
	}

	wantChannelID := "C123456789"
	if got := output.ChannelID; got != wantChannelID {
		t.Errorf("ChannelID: got %q, want %q", got, wantChannelID)
	}

	wantThreadTS := "1234567890.123456"
	if got := output.ThreadTimestamp; got != wantThreadTS {
		t.Errorf("ThreadTimestamp: got %q, want %q", got, wantThreadTS)
	}

	wantMsgCount := 3
	if got := len(output.Messages); got != wantMsgCount {
		t.Errorf("len(Messages): got %d, want %d", got, wantMsgCount)
	}

	wantParentText := "Thread parent message"
	if got := output.Messages[0].Text; got != wantParentText {
		t.Errorf("Messages[0].Text: got %q, want %q", got, wantParentText)
	}

	wantReplyText := "First reply"
	if got := output.Messages[1].Text; got != wantReplyText {
		t.Errorf("Messages[1].Text: got %q, want %q", got, wantReplyText)
	}

	wantReplyUser := "bob"
	if got := output.Messages[1].UserName; got != wantReplyUser {
		t.Errorf("Messages[1].UserName: got %q, want %q", got, wantReplyUser)
	}

	wantHasMore := false
	if got := output.HasMore; got != wantHasMore {
		t.Errorf("HasMore: got %v, want %v", got, wantHasMore)
	}
}

func TestReadCanvas_ByFileID(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	canvasHTML := "<h1>My Canvas</h1><p>Hello <b>world</b></p>"

	// Mock files.info
	mock.addHandler("/files.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"file": map[string]interface{}{
				"id":                   "F123CANVAS",
				"name":                 "My Canvas",
				"title":                "My Canvas",
				"filetype":             "quip",
				"url_private_download": mock.server.URL + "/files/F123CANVAS/download",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Mock file download
	mock.addHandler("/files/F123CANVAS/download", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(canvasHTML))
	})

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ReadCanvasInput{
		FileID: "F123CANVAS",
	}

	_, output, err := client.ReadCanvas(ctx, nil, input)
	if err != nil {
		t.Fatalf("ReadCanvas failed: %v", err)
	}

	if output.FileID != "F123CANVAS" {
		t.Errorf("FileID: got %q, want %q", output.FileID, "F123CANVAS")
	}

	if output.Title != "My Canvas" {
		t.Errorf("Title: got %q, want %q", output.Title, "My Canvas")
	}

	if output.File.Path == "" {
		t.Error("File.Path: got empty, want non-empty")
	}

	// Verify content was written
	data, err := os.ReadFile(output.File.Path)
	if err != nil {
		t.Fatalf("Failed to read response file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "My Canvas") {
		t.Errorf("Content should contain 'My Canvas', got %q", content)
	}
	if !strings.Contains(content, "world") {
		t.Errorf("Content should contain 'world', got %q", content)
	}
}

func TestReadCanvas_ByChannel(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	// Mock conversations.info returning canvas properties
	mock.addHandler("/conversations.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"channel": map[string]interface{}{
				"id":   "C123456789",
				"name": "design-docs",
				"properties": map[string]interface{}{
					"canvas": map[string]interface{}{
						"file_id":  "F456CANVAS",
						"is_empty": false,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Mock files.info
	mock.addHandler("/files.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"file": map[string]interface{}{
				"id":                   "F456CANVAS",
				"name":                 "Channel Canvas",
				"title":                "Channel Canvas",
				"filetype":             "quip",
				"url_private_download": mock.server.URL + "/files/F456CANVAS/download",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Mock file download
	mock.addHandler("/files/F456CANVAS/download", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<p>Channel canvas content</p>"))
	})

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ReadCanvasInput{
		Channel: "C123456789",
	}

	_, output, err := client.ReadCanvas(ctx, nil, input)
	if err != nil {
		t.Fatalf("ReadCanvas failed: %v", err)
	}

	if output.FileID != "F456CANVAS" {
		t.Errorf("FileID: got %q, want %q", output.FileID, "F456CANVAS")
	}

	if output.Title != "Channel Canvas" {
		t.Errorf("Title: got %q, want %q", output.Title, "Channel Canvas")
	}

	data, err := os.ReadFile(output.File.Path)
	if err != nil {
		t.Fatalf("Failed to read response file: %v", err)
	}

	if !strings.Contains(string(data), "Channel canvas content") {
		t.Errorf("Content should contain 'Channel canvas content', got %q", string(data))
	}
}

func TestReadCanvas_ChannelWithoutCanvas(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	// Mock conversations.info with no canvas
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

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ReadCanvasInput{
		Channel: "C123456789",
	}

	_, _, err := client.ReadCanvas(ctx, nil, input)
	if err == nil {
		t.Fatal("Expected error for channel without canvas, got nil")
	}

	if !strings.Contains(err.Error(), "no canvas") {
		t.Errorf("Error should mention 'no canvas', got %q", err.Error())
	}
}

func TestReadCanvas_ValidationError(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()

	// Neither channel nor file_id provided
	_, _, err := client.ReadCanvas(ctx, nil, ReadCanvasInput{})
	if err == nil {
		t.Fatal("Expected error when neither channel nor file_id provided, got nil")
	}

	// Both channel and file_id provided
	_, _, err = client.ReadCanvas(ctx, nil, ReadCanvasInput{
		Channel: "C123456789",
		FileID:  "F123CANVAS",
	})
	if err == nil {
		t.Fatal("Expected error when both channel and file_id provided, got nil")
	}
}

func TestReadCanvas_NonCanvasFile(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	// Mock files.info returning a non-canvas file
	mock.addHandler("/files.info", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"file": map[string]interface{}{
				"id":                   "F789PDF",
				"name":                 "document.pdf",
				"title":                "Some PDF",
				"filetype":             "pdf",
				"url_private_download": mock.server.URL + "/files/F789PDF/download",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	client, _, responsesDir := newTestClient(t, mock)
	defer os.RemoveAll(responsesDir)

	ctx := context.Background()
	input := ReadCanvasInput{
		FileID: "F789PDF",
	}

	_, _, err := client.ReadCanvas(ctx, nil, input)
	if err == nil {
		t.Fatal("Expected error for non-canvas file, got nil")
	}

	if !strings.Contains(err.Error(), "not a canvas") {
		t.Errorf("Error should mention 'not a canvas', got %q", err.Error())
	}
}
