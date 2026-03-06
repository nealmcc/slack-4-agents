package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

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

	output, err := client.GetPermalink(ctx, input)
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
