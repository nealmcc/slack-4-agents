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

func TestBuildMessageInfo(t *testing.T) {
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

	got := buildMessageInfo(msg, "", "alice")

	if got.Timestamp != "1234567890.123456" {
		t.Errorf("Timestamp: got %q, want %q", got.Timestamp, "1234567890.123456")
	}
	if got.TimestampDisplay != "2009-02-13T23:31:30Z" {
		t.Errorf("TimestampDisplay: got %q, want %q", got.TimestampDisplay, "2009-02-13T23:31:30Z")
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

func TestBuildMessageInfo_ThreadReply(t *testing.T) {
	msg := slack.Message{
		Msg: slack.Msg{
			Timestamp: "1234567891.123456",
			User:      "U987654321",
			Text:      "This is a reply",
		},
	}

	got := buildMessageInfo(msg, "1234567890.123456", "bob")

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

func TestFormatSlackTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"standard slack timestamp", "1234567890.123456", "2009-02-13T23:31:30Z"},
		{"timestamp without microseconds", "1234567890", "2009-02-13T23:31:30Z"},
		{"empty timestamp", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSlackTimestamp(tt.input)
			if got != tt.want {
				t.Errorf("formatSlackTimestamp(%q): got %q, want %q", tt.input, got, tt.want)
			}
		})
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

	output, err := client.ExportChannel(ctx, input)
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
	var msg MessageInfo
	if err := json.Unmarshal([]byte(lines[0]), &msg); err != nil {
		t.Fatalf("Failed to unmarshal first line: %v", err)
	}

	if msg.UserName != "alice" {
		t.Errorf("First message UserName: got %q, want %q", msg.UserName, "alice")
	}

	if msg.Timestamp != "1704067200.000001" {
		t.Errorf("Timestamp: got %q, want raw Slack ts", msg.Timestamp)
	}
	if !strings.HasPrefix(msg.TimestampDisplay, "2024-") {
		t.Errorf("TimestampDisplay not in ISO format: got %q", msg.TimestampDisplay)
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

	output, err := client.ExportChannel(ctx, input)
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
	var reply MessageInfo
	if err := json.Unmarshal([]byte(threadLines[1]), &reply); err != nil {
		t.Fatalf("Failed to unmarshal second line: %v", err)
	}

	if reply.ThreadTimestamp != "1704067200.000001" {
		t.Errorf("Reply ThreadTimestamp: got %q, want %q", reply.ThreadTimestamp, "1704067200.000001")
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

	output, err := client.ExportChannel(ctx, input)
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

	var msg MessageInfo
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

	_, err := client.ExportChannel(ctx, input)
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

	output, err := client.ExportChannel(ctx, input)
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

	output, err := client.ExportChannel(ctx, input)
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
