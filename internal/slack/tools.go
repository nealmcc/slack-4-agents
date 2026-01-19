package slack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// ListChannelsOutput contains the list of channels
type ListChannelsOutput struct {
	Channels   []ChannelInfo `json:"channels"`
	NextCursor string        `json:"next_cursor,omitempty"`
}

// ListChannels lists channels the user has access to
func (c *Client) ListChannels(ctx context.Context, req *mcp.CallToolRequest, input ListChannelsInput) (*mcp.CallToolResult, ListChannelsOutput, error) {
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

	channels, cursor, err := c.api.GetConversationsContext(ctx, params)
	if err != nil {
		return nil, ListChannelsOutput{}, fmt.Errorf("failed to list channels: %w", err)
	}

	output := ListChannelsOutput{
		Channels:   make([]ChannelInfo, 0, len(channels)),
		NextCursor: cursor,
	}

	for _, ch := range channels {
		output.Channels = append(output.Channels, ChannelInfo{
			ID:          ch.ID,
			Name:        ch.Name,
			Topic:       ch.Topic.Value,
			Purpose:     ch.Purpose.Value,
			MemberCount: ch.NumMembers,
			IsPrivate:   ch.IsPrivate,
			IsArchived:  ch.IsArchived,
		})
	}

	return nil, output, nil
}

// ReadHistoryInput defines input for reading channel history
type ReadHistoryInput struct {
	Channel string `json:"channel" jsonschema:"Channel ID or name (e.g., C1234567890 or #general)"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Number of messages to fetch (default 20, max 100)"`
	Latest  string `json:"latest,omitempty" jsonschema:"End of time range (Unix timestamp)"`
	Oldest  string `json:"oldest,omitempty" jsonschema:"Start of time range (Unix timestamp)"`
}

// MessageInfo represents a Slack message
type MessageInfo struct {
	Timestamp       string `json:"timestamp"`
	User            string `json:"user"`
	UserName        string `json:"user_name,omitempty"`
	Text            string `json:"text"`
	ThreadTimestamp string `json:"thread_ts,omitempty"`
	ReplyCount      int    `json:"reply_count,omitempty"`
}

// ReadHistoryOutput contains channel messages
type ReadHistoryOutput struct {
	ChannelID string        `json:"channel_id"`
	Messages  []MessageInfo `json:"messages"`
	HasMore   bool          `json:"has_more"`
}

// ReadHistory reads message history from a channel
func (c *Client) ReadHistory(ctx context.Context, req *mcp.CallToolRequest, input ReadHistoryInput) (*mcp.CallToolResult, ReadHistoryOutput, error) {
	channelID, err := c.GetChannelID(ctx, input.Channel)
	if err != nil {
		return nil, ReadHistoryOutput{}, err
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
		return nil, ReadHistoryOutput{}, fmt.Errorf("failed to get history: %w", err)
	}

	output := ReadHistoryOutput{
		ChannelID: channelID,
		Messages:  make([]MessageInfo, 0, len(history.Messages)),
		HasMore:   history.HasMore,
	}

	// Collect user IDs to fetch names
	userIDs := make(map[string]bool)
	for _, msg := range history.Messages {
		if msg.User != "" {
			userIDs[msg.User] = true
		}
	}

	// Fetch user names
	userNames := make(map[string]string)
	for userID := range userIDs {
		user, err := c.api.GetUserInfoContext(ctx, userID)
		if err == nil {
			userNames[userID] = user.Name
		}
	}

	for _, msg := range history.Messages {
		output.Messages = append(output.Messages, MessageInfo{
			Timestamp:       msg.Timestamp,
			User:            msg.User,
			UserName:        userNames[msg.User],
			Text:            msg.Text,
			ThreadTimestamp: msg.ThreadTimestamp,
			ReplyCount:      msg.ReplyCount,
		})
	}

	return nil, output, nil
}

// SearchMessagesInput defines input for searching messages
type SearchMessagesInput struct {
	Query string `json:"query" jsonschema:"Search query (supports Slack search modifiers like from:@user, in:#channel, before:date)"`
	Count int    `json:"count,omitempty" jsonschema:"Number of results to return (default 20, max 100)"`
	Sort  string `json:"sort,omitempty" jsonschema:"Sort order: score (relevance) or timestamp (recent first)"`
}

// SearchMatch represents a search result
type SearchMatch struct {
	Timestamp string `json:"timestamp"`
	Channel   string `json:"channel"`
	User      string `json:"user"`
	UserName  string `json:"user_name,omitempty"`
	Text      string `json:"text"`
	Permalink string `json:"permalink"`
}

// SearchMessagesOutput contains search results
type SearchMessagesOutput struct {
	Query   string        `json:"query"`
	Total   int           `json:"total"`
	Matches []SearchMatch `json:"matches"`
}

// SearchMessages searches messages across the workspace
func (c *Client) SearchMessages(ctx context.Context, req *mcp.CallToolRequest, input SearchMessagesInput) (*mcp.CallToolResult, SearchMessagesOutput, error) {
	count := 20
	if input.Count > 0 && input.Count <= 100 {
		count = input.Count
	}

	sort := "score"
	if input.Sort == "timestamp" {
		sort = "timestamp"
	}

	params := slack.SearchParameters{
		Sort:          sort,
		SortDirection: "desc",
		Count:         count,
	}

	results, err := c.api.SearchMessagesContext(ctx, input.Query, params)
	if err != nil {
		return nil, SearchMessagesOutput{}, fmt.Errorf("failed to search: %w", err)
	}

	output := SearchMessagesOutput{
		Query:   input.Query,
		Total:   results.Total,
		Matches: make([]SearchMatch, 0, len(results.Matches)),
	}

	for _, match := range results.Matches {
		output.Matches = append(output.Matches, SearchMatch{
			Timestamp: match.Timestamp,
			Channel:   match.Channel.Name,
			User:      match.User,
			UserName:  match.Username,
			Text:      match.Text,
			Permalink: match.Permalink,
		})
	}

	return nil, output, nil
}

// GetUserInput defines input for getting user info
type GetUserInput struct {
	User  string `json:"user,omitempty" jsonschema:"User ID (e.g., U1234567890)"`
	Email string `json:"email,omitempty" jsonschema:"User email address"`
}

// UserInfo represents a Slack user
type UserInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	RealName    string `json:"real_name"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email,omitempty"`
	Title       string `json:"title,omitempty"`
	Status      string `json:"status,omitempty"`
	StatusEmoji string `json:"status_emoji,omitempty"`
	IsBot       bool   `json:"is_bot"`
	IsAdmin     bool   `json:"is_admin"`
	Timezone    string `json:"timezone,omitempty"`
}

// GetUserOutput contains user information
type GetUserOutput struct {
	User UserInfo `json:"user"`
}

// GetUser looks up user information by ID or email
func (c *Client) GetUser(ctx context.Context, req *mcp.CallToolRequest, input GetUserInput) (*mcp.CallToolResult, GetUserOutput, error) {
	var user *slack.User
	var err error

	if input.User != "" {
		user, err = c.api.GetUserInfoContext(ctx, input.User)
	} else if input.Email != "" {
		user, err = c.api.GetUserByEmailContext(ctx, input.Email)
	} else {
		return nil, GetUserOutput{}, fmt.Errorf("either user ID or email is required")
	}

	if err != nil {
		return nil, GetUserOutput{}, fmt.Errorf("failed to get user: %w", err)
	}

	output := GetUserOutput{
		User: UserInfo{
			ID:          user.ID,
			Name:        user.Name,
			RealName:    user.RealName,
			DisplayName: user.Profile.DisplayName,
			Email:       user.Profile.Email,
			Title:       user.Profile.Title,
			Status:      user.Profile.StatusText,
			StatusEmoji: user.Profile.StatusEmoji,
			IsBot:       user.IsBot,
			IsAdmin:     user.IsAdmin,
			Timezone:    user.TZ,
		},
	}

	return nil, output, nil
}

// GetPermalinkInput defines input for getting a message permalink
type GetPermalinkInput struct {
	Channel   string `json:"channel" jsonschema:"Channel ID (e.g., C1234567890)"`
	Timestamp string `json:"timestamp" jsonschema:"Message timestamp (e.g., 1234567890.123456)"`
}

// GetPermalinkOutput contains the permalink
type GetPermalinkOutput struct {
	Permalink string `json:"permalink"`
	Channel   string `json:"channel"`
	Timestamp string `json:"timestamp"`
}

// GetPermalink gets a permalink to a specific message
func (c *Client) GetPermalink(ctx context.Context, req *mcp.CallToolRequest, input GetPermalinkInput) (*mcp.CallToolResult, GetPermalinkOutput, error) {
	channelID, err := c.GetChannelID(ctx, input.Channel)
	if err != nil {
		return nil, GetPermalinkOutput{}, err
	}

	permalink, err := c.api.GetPermalinkContext(ctx, &slack.PermalinkParameters{
		Channel: channelID,
		Ts:      input.Timestamp,
	})
	if err != nil {
		return nil, GetPermalinkOutput{}, fmt.Errorf("failed to get permalink: %w", err)
	}

	return nil, GetPermalinkOutput{
		Permalink: permalink,
		Channel:   channelID,
		Timestamp: input.Timestamp,
	}, nil
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
func (c *Client) ReadThread(ctx context.Context, req *mcp.CallToolRequest, input ReadThreadInput) (*mcp.CallToolResult, ReadThreadOutput, error) {
	channelID, err := c.GetChannelID(ctx, input.Channel)
	if err != nil {
		return nil, ReadThreadOutput{}, err
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
		return nil, ReadThreadOutput{}, fmt.Errorf("failed to get thread replies: %w", err)
	}

	output := ReadThreadOutput{
		ChannelID:       channelID,
		ThreadTimestamp: input.Timestamp,
		Messages:        make([]MessageInfo, 0, len(messages)),
		HasMore:         hasMore,
		NextCursor:      nextCursor,
	}

	// Collect user IDs to fetch names
	userIDs := make(map[string]bool)
	for _, msg := range messages {
		if msg.User != "" {
			userIDs[msg.User] = true
		}
	}

	// Fetch user names
	userNames := make(map[string]string)
	for userID := range userIDs {
		user, err := c.api.GetUserInfoContext(ctx, userID)
		if err == nil {
			userNames[userID] = user.Name
		}
	}

	for _, msg := range messages {
		output.Messages = append(output.Messages, MessageInfo{
			Timestamp:       msg.Timestamp,
			User:            msg.User,
			UserName:        userNames[msg.User],
			Text:            msg.Text,
			ThreadTimestamp: msg.ThreadTimestamp,
			ReplyCount:      msg.ReplyCount,
		})
	}

	return nil, output, nil
}

// RegisterTools registers all Slack tools with the MCP server
func (c *Client) RegisterTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_list_channels",
		Description: "List Slack channels the user has access to. Returns channel names, IDs, topics, and member counts.",
	}, c.ListChannels)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_read_history",
		Description: "Read message history from a Slack channel or conversation. Returns messages with author info, timestamps, and thread details.",
	}, c.ReadHistory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_search_messages",
		Description: "Search for messages across the Slack workspace. Supports Slack search syntax like from:@user, in:#channel, before:2024-01-01.",
	}, c.SearchMessages)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_get_user",
		Description: "Look up a Slack user by ID or email address. Returns profile information including name, title, status, and timezone.",
	}, c.GetUser)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_get_permalink",
		Description: "Get a permanent link (URL) to a specific Slack message. Useful for sharing or referencing messages.",
	}, c.GetPermalink)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "slack_read_thread",
		Description: "Read all replies in a Slack thread. Use the thread parent's timestamp from slack_read_history (messages with reply_count > 0).",
	}, c.ReadThread)
}

// formatTimestamp converts a Slack timestamp to a readable format
func formatTimestamp(ts string) string {
	// Slack timestamps are Unix timestamps with microseconds: "1234567890.123456"
	parts := strings.Split(ts, ".")
	if len(parts) == 0 {
		return ts
	}
	var sec int64
	fmt.Sscanf(parts[0], "%d", &sec)
	t := time.Unix(sec, 0)
	return t.Format("2006-01-02 15:04:05")
}
