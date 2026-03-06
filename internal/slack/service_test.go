package slack

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/slack-go/slack"
	"go.uber.org/zap/zaptest"
)

func TestGetChannelID_ConcurrentIDPassthrough(t *testing.T) {
	cache := newIndex()
	client := newServiceWithIndex(nil, cache, zaptest.NewLogger(t), nil)

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
	ix := newIndex()
	for i := 0; i < 20; i++ {
		ix.Add([]slack.Channel{fakeChannel(i)})
	}

	client := newServiceWithIndex(nil, ix, zaptest.NewLogger(t), nil)

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
	ix := newIndex()
	ix.Add([]slack.Channel{
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
	})

	client := newServiceWithIndex(nil, ix, zaptest.NewLogger(t), nil)

	_, err := client.GetChannelID("nonexistent")
	if err == nil {
		t.Fatal("got nil error, want error for missing channel")
	}

	if !strings.Contains(err.Error(), "not found in index") {
		t.Errorf("error message should mention index, got: %v", err)
	}
	if !strings.Contains(err.Error(), "slack_list_channels") {
		t.Errorf("error message should suggest slack_list_channels, got: %v", err)
	}
}

func TestFindChannelID_NotInIndex(t *testing.T) {
	ix := newIndex()
	client := newServiceWithIndex(nil, ix, zaptest.NewLogger(t), nil)

	_, err := client.GetChannelID("does-not-exist")
	if err == nil {
		t.Fatal("got nil error, want error for missing channel")
	}

	want := `channel "does-not-exist" not found in index (0 entries); use a channel ID or call slack_list_channels first`
	if err.Error() != want {
		t.Errorf("error message:\ngot:  %s\nwant: %s", err.Error(), want)
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
