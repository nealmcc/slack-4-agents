package slack

import (
	"context"
	"fmt"
	"strings"

	"github.com/slack-go/slack"
)

// ListChannelsInput defines input for listing channels
type ListChannelsInput struct {
	Types  string `json:"types,omitempty" jsonschema:"Channel types: public_channel, private_channel, mpim, im (comma-separated). Default: public_channel, private_channel"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Max channels to return (default 100)"`
	Cursor string `json:"cursor,omitempty" jsonschema:"Pagination cursor for fetching more results"`
}

// ChannelInfo represents a Slack channel
type ChannelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Topic       string `json:"topic,omitempty"`
	Purpose     string `json:"purpose,omitempty"`
	MemberCount int    `json:"member_count"`
	IsPrivate   bool   `json:"is_private"`
	IsArchived  bool   `json:"is_archived"`
}

// ListChannelsOutput contains a summary and file reference (to save tokens)
type ListChannelsOutput struct {
	File         FileRef      `json:"file"`
	TotalCount   int          `json:"total_count"`
	FirstChannel *ChannelInfo `json:"first_channel,omitempty"`
	LastChannel  *ChannelInfo `json:"last_channel,omitempty"`
	NextCursor   string       `json:"next_cursor,omitempty"`
}

// ListChannels lists channels the user has access to
func (c *Service) ListChannels(ctx context.Context, input ListChannelsInput) (ListChannelsOutput, error) {
	types := []string{"public_channel", "private_channel"}
	if input.Types != "" {
		types = strings.Split(input.Types, ",")
		for i := range types {
			types[i] = strings.TrimSpace(types[i])
		}
	}

	limit := 100
	if input.Limit > 0 && input.Limit <= 1000 {
		limit = input.Limit
	}

	params := &slack.GetConversationsParameters{
		Types:  types,
		Limit:  limit,
		Cursor: input.Cursor,
	}

	channels, cursor, err := c.listConversations(ctx, params)
	if err != nil {
		return ListChannelsOutput{}, fmt.Errorf("failed to list channels: %w", err)
	}

	channelInfos := make([]ChannelInfo, 0, len(channels))
	for _, ch := range channels {
		channelInfos = append(channelInfos, ChannelInfo{
			ID:          ch.ID,
			Name:        ch.Name,
			Topic:       ch.Topic.Value,
			Purpose:     ch.Purpose.Value,
			MemberCount: ch.NumMembers,
			IsPrivate:   ch.IsPrivate,
			IsArchived:  ch.IsArchived,
		})
	}

	fileRef, err := c.responses.WriteJSON("channels", channelInfos)
	if err != nil {
		return ListChannelsOutput{}, fmt.Errorf("failed to write response: %w", err)
	}

	output := ListChannelsOutput{
		File:       fileRef,
		TotalCount: len(channelInfos),
		NextCursor: cursor,
	}

	if len(channelInfos) > 0 {
		output.FirstChannel = &channelInfos[0]
		output.LastChannel = &channelInfos[len(channelInfos)-1]
	}

	return output, nil
}
