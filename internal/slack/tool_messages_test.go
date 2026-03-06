package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

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

	output, err := client.ReadHistory(ctx, input)
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

	output, err := client.ReadThread(ctx, input)
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
