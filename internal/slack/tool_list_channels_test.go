package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

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

	output, err := client.ListChannels(ctx, input)
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
