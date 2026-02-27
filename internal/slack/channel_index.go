package slack

import (
	"strings"

	"github.com/slack-go/slack"
)

type channelIndex struct {
	names map[string]slack.Channel
	ids   map[string]slack.Channel
}

// newIndex initializes a new channelIndex containing the given slack channels
func newIndex(channels []slack.Channel) *channelIndex {
	names := make(map[string]slack.Channel)
	ids := make(map[string]slack.Channel)
	for _, ch := range channels {
		names[ch.NameNormalized] = ch
		id := strings.ToLower(ch.ID)
		ids[id] = ch
	}
	return &channelIndex{names, ids}
}

/*
Get a channel by name
*/
func (ix *channelIndex) GetByName(name string) (slack.Channel, bool) {
	name = strings.ToLower(name)
	ch, ok := ix.names[name]
	return ch, ok
}

/*
Get a channel by ID
*/
func (ix *channelIndex) GetByID(id string) (slack.Channel, bool) {
	id = strings.ToLower(id)
	ch, ok := ix.ids[id]
	return ch, ok
}

// Size returns the number of channels in the cache
func (ix *channelIndex) Size() int {
	return len(ix.ids)
}
