package slack

import (
	"context"
	"fmt"

	"github.com/slack-go/slack"
)

// ReadThreadInput defines input for reading thread replies
type ReadThreadInput struct {
	Channel   string `json:"channel" jsonschema:"Channel ID (e.g., C1234567890)"`
	Timestamp string `json:"timestamp" jsonschema:"Thread parent message timestamp (e.g., 1234567890.123456)"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Number of replies to fetch (default 100, max 1000)"`
	Cursor    string `json:"cursor,omitempty" jsonschema:"Pagination cursor for fetching more replies"`
}

// ReadThreadOutput contains thread replies
type ReadThreadOutput struct {
	ChannelID       string        `json:"channel_id"`
	ThreadTimestamp string        `json:"thread_ts"`
	Messages        []MessageInfo `json:"messages"`
	HasMore         bool          `json:"has_more"`
	NextCursor      string        `json:"next_cursor,omitempty"`
}

// ReadThread reads all replies in a thread
func (c *Service) ReadThread(ctx context.Context, input ReadThreadInput) (ReadThreadOutput, error) {
	channelID, err := c.GetChannelID(input.Channel)
	if err != nil {
		return ReadThreadOutput{}, err
	}

	limit := 100
	if input.Limit > 0 && input.Limit <= 1000 {
		limit = input.Limit
	}

	params := &slack.GetConversationRepliesParameters{
		ChannelID: channelID,
		Timestamp: input.Timestamp,
		Limit:     limit,
		Cursor:    input.Cursor,
	}

	messages, hasMore, nextCursor, err := c.api.GetConversationRepliesContext(ctx, params)
	if err != nil {
		return ReadThreadOutput{}, fmt.Errorf("failed to get thread replies: %w", err)
	}

	output := ReadThreadOutput{
		ChannelID:       channelID,
		ThreadTimestamp: input.Timestamp,
		Messages:        make([]MessageInfo, 0, len(messages)),
		HasMore:         hasMore,
		NextCursor:      nextCursor,
	}

	names := c.newUserNameCache(ctx)

	for _, msg := range messages {
		output.Messages = append(output.Messages, MessageInfo{
			Timestamp:        msg.Timestamp,
			TimestampDisplay: formatSlackTimestamp(msg.Timestamp),
			User:             msg.User,
			UserName:         names.Get(msg.User),
			Text:             msg.Text,
			ThreadTimestamp:  msg.ThreadTimestamp,
			ReplyCount:       msg.ReplyCount,
		})
	}

	return output, nil
}
