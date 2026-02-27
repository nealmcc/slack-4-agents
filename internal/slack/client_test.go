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
	"go.uber.org/zap/zaptest"
)

func TestGetChannelID_ConcurrentIDPassthrough(t *testing.T) {
	cache := newIndex([]slack.Channel{})
	client := newClientWithAPI(nil, cache, zaptest.NewLogger(t), nil)

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := client.GetChannelID("CTEST12345")
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
	channels := make([]slack.Channel, 20)
	for i := 0; i < 20; i++ {
		channels[i] = fakeChannel(i)
	}
	ix := newIndex(channels)
	client := newClientWithAPI(nil, ix, zaptest.NewLogger(t), nil)

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := fmt.Sprintf("channel-%d", idx)
			wantID := fmt.Sprintf("C%09d", idx)
			id, err := client.GetChannelID(name)
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

func TestGetChannelID_CacheMiss(t *testing.T) {
	channels := []slack.Channel{
		{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{
					ID:             "C000000001",
					NameNormalized: "general",
				},
				Name: "General",
			},
			IsChannel: true,
			IsGeneral: true,
		},
	}
	index := newIndex(channels)
	client := newClientWithAPI(nil, index, zaptest.NewLogger(t), nil)

	_, err := client.GetChannelID("nonexistent")
	if err == nil {
		t.Fatal("got nil error, want error for missing channel")
	}
}

func TestFetchAllChannels(t *testing.T) {
	mock := newMockSlackServer()
	defer mock.close()

	mock.addHandler("/conversations.list", func(w http.ResponseWriter, r *http.Request) {
		channels := []slack.Channel{
			{GroupConversation: slack.GroupConversation{
				Name: "General",
				Conversation: slack.Conversation{
					ID:             "C000000001",
					NameNormalized: "general",
				},
			}},
			{GroupConversation: slack.GroupConversation{
				Name: "Random",
				Conversation: slack.Conversation{
					ID:             "C000000002",
					NameNormalized: "random",
				},
			}},
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

	api := slack.New("xoxb-test-token",
		slack.OptionAPIURL(mock.server.URL+"/"),
	)
	outputDir, err := os.MkdirTemp("", "slack-4-agents-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(outputDir)

	logger := newTestLogger()
	responses := NewFileResponseWriter(outputDir)
	client := newClientWithAPI(api, nil, logger.Logger, responses)

	channels, err := client.fetchAllChannels(context.Background())
	if err != nil {
		t.Fatalf("fetchAllChannels: %v", err)
	}

	if got, want := len(channels), 2; got != want {
		t.Fatalf("channel count: got %d, want %d", got, want)
	}

	wantChannels := map[string]string{
		"C000000001": "general",
		"C000000002": "random",
	}
	for _, ch := range channels {
		if wantName, ok := wantChannels[ch.ID]; ok {
			if ch.NameNormalized != wantName {
				t.Errorf("channel %s: got name %q, want %q", ch.ID, ch.NameNormalized, wantName)
			}
			delete(wantChannels, ch.ID)
		}
	}
	for id, name := range wantChannels {
		t.Errorf("missing channel %s (%s)", id, name)
	}
}

func fakeChannel(i int) slack.Channel {
	return slack.Channel{
		GroupConversation: slack.GroupConversation{
			Conversation: slack.Conversation{
				ID:             fmt.Sprintf("C%09d", i),
				NameNormalized: fmt.Sprintf("channel-%d", i),
			},
			Name: fmt.Sprintf("Channel-%d", i),
		},
		IsChannel: true,
		IsGeneral: false,
	}
}
