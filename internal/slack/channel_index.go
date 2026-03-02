package slack

import (
	"strings"
	"sync"

	"github.com/slack-go/slack"
)

type channelIndex struct {
	mu    sync.RWMutex
	names map[string]slack.Channel
	ids   map[string]slack.Channel
}

func newIndex() *channelIndex {
	return &channelIndex{
		names: make(map[string]slack.Channel),
		ids:   make(map[string]slack.Channel),
	}
}

// Add inserts channels into the index. Safe for concurrent use.
func (ix *channelIndex) Add(channels []slack.Channel) {
	ix.mu.Lock()
	defer ix.mu.Unlock()

	for _, ch := range channels {
		if ch.NameNormalized != "" && ch.ID != "" {
			ix.names[strings.ToLower(ch.NameNormalized)] = ch
			ix.ids[strings.ToLower(ch.ID)] = ch
		}
	}
}

// GetByName returns a channel by name. Safe for concurrent use.
func (ix *channelIndex) GetByName(name string) (slack.Channel, bool) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	ch, ok := ix.names[strings.ToLower(name)]
	return ch, ok
}

// GetByID returns a channel by ID. Safe for concurrent use.
func (ix *channelIndex) GetByID(id string) (slack.Channel, bool) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	ch, ok := ix.ids[strings.ToLower(id)]
	return ch, ok
}

// Size returns the number of channels in the index. Safe for concurrent use.
func (ix *channelIndex) Size() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return len(ix.ids)
}
