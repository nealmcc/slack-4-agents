package slack

import (
	"context"
	"fmt"

	"github.com/slack-go/slack"
)

// ReadHistoryInput defines input for reading channel history
type ReadHistoryInput struct {
	Channel string `json:"channel" jsonschema:"Channel ID or name (e.g., C1234567890 or #general)"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Number of messages to fetch (default 20, max 100)"`
	Latest  string `json:"latest,omitempty" jsonschema:"End of time range (Unix timestamp)"`
	Oldest  string `json:"oldest,omitempty" jsonschema:"Start of time range (Unix timestamp)"`
}

// ReadHistoryOutput contains channel messages
type ReadHistoryOutput struct {
	ChannelID string        `json:"channel_id"`
	Messages  []MessageInfo `json:"messages"`
	HasMore   bool          `json:"has_more"`
}

// ReadHistory reads message history from a channel
func (c *Service) ReadHistory(ctx context.Context, input ReadHistoryInput) (ReadHistoryOutput, error) {
	channelID, err := c.GetChannelID(input.Channel)
	if err != nil {
		return ReadHistoryOutput{}, err
	}

	limit := 20
	if input.Limit > 0 && input.Limit <= 100 {
		limit = input.Limit
	}

	params := &slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Limit:     limit,
		Latest:    input.Latest,
		Oldest:    input.Oldest,
	}

	history, err := c.api.GetConversationHistoryContext(ctx, params)
	if err != nil {
		return ReadHistoryOutput{}, fmt.Errorf("failed to get history: %w", err)
	}

	output := ReadHistoryOutput{
		ChannelID: channelID,
		Messages:  make([]MessageInfo, 0, len(history.Messages)),
		HasMore:   history.HasMore,
	}

	names := c.newUserNameCache(ctx)

	for _, msg := range history.Messages {
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
