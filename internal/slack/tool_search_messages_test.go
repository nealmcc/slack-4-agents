package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

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

	output, err := client.SearchMessages(ctx, input)
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
