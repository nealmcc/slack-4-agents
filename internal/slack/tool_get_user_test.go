package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

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

	output, err := client.GetUser(ctx, input)
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
