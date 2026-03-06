package slack

import (
	"context"
	"fmt"
	"time"

	"github.com/slack-go/slack"
)

// MessageInfo represents a Slack message
type MessageInfo struct {
	Timestamp        string         `json:"timestamp"`
	TimestampDisplay string         `json:"timestamp_display,omitempty"`
	User             string         `json:"user"`
	UserName         string         `json:"user_name,omitempty"`
	Text             string         `json:"text"`
	ThreadTimestamp  string         `json:"thread_ts,omitempty"`
	ReplyCount       int            `json:"reply_count,omitempty"`
	Reactions        []ReactionInfo `json:"reactions,omitempty"`
}

// ReactionInfo represents an emoji reaction with its count
type ReactionInfo struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// formatSlackTimestamp converts a Slack timestamp (e.g. "1234567890.123456") to ISO 8601.
func formatSlackTimestamp(ts string) string {
	if ts == "" {
		return ""
	}
	var sec int64
	fmt.Sscanf(ts, "%d", &sec)
	return time.Unix(sec, 0).UTC().Format(time.RFC3339)
}

// userNameCache provides lazy, cached user-name lookups within a single tool call.
type userNameCache struct {
	svc   *Service
	ctx   context.Context
	cache map[string]string
}

func (c *Service) newUserNameCache(ctx context.Context) *userNameCache {
	return &userNameCache{svc: c, ctx: ctx, cache: make(map[string]string)}
}

func (u *userNameCache) Get(userID string) string {
	if userID == "" {
		return ""
	}
	if name, ok := u.cache[userID]; ok {
		return name
	}
	user, err := u.svc.api.GetUserInfoContext(u.ctx, userID)
	if err == nil {
		u.cache[userID] = user.Name
		return user.Name
	}
	return ""
}

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
