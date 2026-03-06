package slack

import (
	"context"
	"fmt"
	"time"
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
