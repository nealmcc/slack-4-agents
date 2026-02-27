package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/slack-go/slack"
)

func TestGetChannelID_ConcurrentIDValidation(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	mock.addHandler("/conversations.info", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		channelID := r.FormValue("channel")
		resp := map[string]any{
			"ok": true,
			"channel": map[string]any{
				"id":   channelID,
				"name": "test-channel",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	client, _, outputDir := newTestClient(t, mock)
	defer os.RemoveAll(outputDir)

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := client.GetChannelID(context.Background(), "CTEST12345")
			if err != nil {
				errs <- err
				return
			}
			if id != "CTEST12345" {
				errs <- fmt.Errorf("got %q, want %q", id, "CTEST12345")
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}

func TestGetChannelID_ConcurrentNameLookup(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	mock.addHandler("/conversations.list", func(w http.ResponseWriter, r *http.Request) {
		var channels []slack.Channel
		for i := 0; i < 20; i++ {
			channels = append(channels, slack.Channel{
				GroupConversation: slack.GroupConversation{
					Name: fmt.Sprintf("channel-%d", i),
					Conversation: slack.Conversation{
						ID: fmt.Sprintf("C%09d", i),
					},
				},
			})
		}
		resp := map[string]any{
			"ok":       true,
			"channels": channels,
			"response_metadata": map[string]any{
				"next_cursor": "",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	client, _, outputDir := newTestClient(t, mock)
	defer os.RemoveAll(outputDir)

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := fmt.Sprintf("channel-%d", idx)
			wantID := fmt.Sprintf("C%09d", idx)
			id, err := client.GetChannelID(context.Background(), name)
			if err != nil {
				errs <- fmt.Errorf("channel %q: %w", name, err)
				return
			}
			if id != wantID {
				errs <- fmt.Errorf("channel %q: got %q, want %q", name, id, wantID)
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
}
