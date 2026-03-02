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
	ix := newIndex()
	for i := 0; i < 20; i++ {
		ix.Add([]slack.Channel{fakeChannel(i)})
	}

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

	client := newClientWithAPI(nil, ix, zaptest.NewLogger(t), nil)

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
	client := newClientWithAPI(nil, ix, zaptest.NewLogger(t), nil)

	_, err := client.GetChannelID("does-not-exist")
	if err == nil {
		t.Fatal("got nil error, want error for missing channel")
	}

	want := `channel "does-not-exist" not found in index (0 entries); use a channel ID or call slack_list_channels first`
	if err.Error() != want {
		t.Errorf("error message:\ngot:  %s\nwant: %s", err.Error(), want)
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
