package slack

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// ListChannelsOutput contains a summary and file reference (to save tokens)
type ListChannelsOutput struct {
	File         FileRef      `json:"file"`
	TotalCount   int          `json:"total_count"`
	FirstChannel *ChannelInfo `json:"first_channel,omitempty"`
	LastChannel  *ChannelInfo `json:"last_channel,omitempty"`
	NextCursor   string       `json:"next_cursor,omitempty"`
}

// ListChannels lists channels the user has access to
// Results are written to a response file and a summary is returned to save tokens
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

	channels, cursor, err := c.listConversations(ctx, params)
	if err != nil {
		return nil, ListChannelsOutput{}, fmt.Errorf("failed to list channels: %w", err)
	}

	// Convert to ChannelInfo slice
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

	// Write full results to file
	fileRef, err := c.responses.WriteJSON("channels", channelInfos)
	if err != nil {
		return nil, ListChannelsOutput{}, fmt.Errorf("failed to write response: %w", err)
	}

	// Build summary output
	output := ListChannelsOutput{
		File:       fileRef,
		TotalCount: len(channelInfos),
		NextCursor: cursor,
	}

	if len(channelInfos) > 0 {
		output.FirstChannel = &channelInfos[0]
		output.LastChannel = &channelInfos[len(channelInfos)-1]
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
	channelID, err := c.GetChannelID(input.Channel)
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

	results, err := c.searchMessages(ctx, input.Query, params)
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
	channelID, err := c.GetChannelID(input.Channel)
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
	channelID, err := c.GetChannelID(input.Channel)
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

// ReadCanvasInput defines input for reading a Slack canvas
type ReadCanvasInput struct {
	Channel string `json:"channel,omitempty" jsonschema:"Channel ID or name (for channel canvases)"`
	FileID  string `json:"file_id,omitempty" jsonschema:"Canvas file ID (for standalone canvases)"`
}

// ReadCanvasOutput contains the canvas content and metadata
type ReadCanvasOutput struct {
	File   FileRef `json:"file"`
	FileID string  `json:"file_id"`
	Title  string  `json:"title"`
}

// ReadCanvas reads a Slack canvas and returns its content as plain text
func (c *Client) ReadCanvas(ctx context.Context, req *mcp.CallToolRequest, input ReadCanvasInput) (*mcp.CallToolResult, ReadCanvasOutput, error) {
	if input.Channel == "" && input.FileID == "" {
		return nil, ReadCanvasOutput{}, fmt.Errorf("either channel or file_id is required")
	}
	if input.Channel != "" && input.FileID != "" {
		return nil, ReadCanvasOutput{}, fmt.Errorf("provide either channel or file_id, not both")
	}

	fileID := input.FileID

	if input.Channel != "" {
		channelID, err := c.GetChannelID(input.Channel)
		if err != nil {
			return nil, ReadCanvasOutput{}, err
		}

		ch, err := c.getConversationInfo(ctx, channelID)
		if err != nil {
			return nil, ReadCanvasOutput{}, fmt.Errorf("failed to get channel info: %w", err)
		}

		if ch.Properties == nil || ch.Properties.Canvas.FileId == "" {
			return nil, ReadCanvasOutput{}, fmt.Errorf("channel has no canvas")
		}
		fileID = ch.Properties.Canvas.FileId
	}

	var file *slack.File
	err := withRetry(ctx, c.logger, func() error {
		var e error
		file, _, _, e = c.api.GetFileInfoContext(ctx, fileID, 0, 0)
		return e
	})
	if err != nil {
		return nil, ReadCanvasOutput{}, fmt.Errorf("failed to get file info: %w", err)
	}

	if file.Filetype != "quip" {
		return nil, ReadCanvasOutput{}, fmt.Errorf("file is not a canvas (filetype %q, expected \"quip\")", file.Filetype)
	}

	var buf bytes.Buffer
	err = withRetry(ctx, c.logger, func() error {
		buf.Reset()
		return c.api.GetFileContext(ctx, file.URLPrivateDownload, &buf)
	})
	if err != nil {
		return nil, ReadCanvasOutput{}, fmt.Errorf("failed to download canvas: %w", err)
	}

	text := stripHTML(buf.String())

	ref, err := c.responses.WriteText("canvas", text)
	if err != nil {
		return nil, ReadCanvasOutput{}, fmt.Errorf("failed to write response: %w", err)
	}

	return nil, ReadCanvasOutput{
		File:   ref,
		FileID: fileID,
		Title:  file.Title,
	}, nil
}

// Timestamp wraps a Slack timestamp and formats as ISO 8601 for display and JSON
type Timestamp string

// String implements fmt.Stringer, returning ISO 8601 format
func (ts Timestamp) String() string {
	if ts == "" {
		return ""
	}
	var sec int64
	fmt.Sscanf(string(ts), "%d", &sec)
	t := time.Unix(sec, 0).UTC()
	return t.Format(time.RFC3339)
}

// MarshalJSON implements json.Marshaler, outputting ISO 8601 format
func (ts Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(ts.String())
}

// Raw returns the original Slack timestamp
func (ts Timestamp) Raw() string {
	return string(ts)
}

// ExportChannelInput defines input for exporting channel history
type ExportChannelInput struct {
	Channel string `json:"channel" jsonschema:"Channel ID or name"`
	Oldest  string `json:"oldest,omitempty" jsonschema:"Start of time range (Unix timestamp)"`
	Latest  string `json:"latest,omitempty" jsonschema:"End of time range (Unix timestamp)"`
}

// exportStats tracks statistics during channel export
type exportStats struct {
	messageCount  int
	threadCount   int
	reactionCount int
	uniqueUsers   map[string]bool
}

func newExportStats() *exportStats {
	return &exportStats{uniqueUsers: make(map[string]bool)}
}

func (s *exportStats) addReactions(reactions []slack.ItemReaction) {
	for _, r := range reactions {
		s.reactionCount += r.Count
	}
}

func (s *exportStats) trackUser(userID string) {
	s.uniqueUsers[userID] = true
}

// processReactions converts Slack reactions to export format
func processReactions(reactions []slack.ItemReaction) []ReactionInfo {
	if len(reactions) == 0 {
		return nil
	}
	result := make([]ReactionInfo, len(reactions))
	for i, r := range reactions {
		result[i] = ReactionInfo{Name: r.Name, Count: r.Count}
	}
	return result
}

// buildExportMessage converts a Slack message to export format
func buildExportMessage(msg slack.Message, threadTs Timestamp, userName string) ExportMessage {
	return ExportMessage{
		Timestamp:       Timestamp(msg.Timestamp),
		User:            msg.User,
		UserName:        userName,
		Text:            msg.Text,
		ThreadTimestamp: threadTs,
		ReplyCount:      msg.ReplyCount,
		Reactions:       processReactions(msg.Reactions),
	}
}

// writeThreadFile writes a complete thread (parent + replies) to a separate file
func (c *Client) writeThreadFile(
	ctx context.Context,
	channelID string,
	parentMsg slack.Message,
	getUserName func(string) string,
	stats *exportStats,
) (FileRef, error) {
	parentTs := Timestamp(parentMsg.Timestamp)
	filename := fmt.Sprintf("export-%s-thread-%s.jsonl", channelID, parentTs.Raw())

	return c.responses.WriteJSONLinesNamed(filename, func(jw JSONLineWriter) error {
		stats.trackUser(parentMsg.User)
		stats.addReactions(parentMsg.Reactions)
		if err := jw.WriteLine(buildExportMessage(parentMsg, "", getUserName(parentMsg.User))); err != nil {
			return err
		}

		cursor := ""
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			var replies []slack.Message
			var hasMore bool
			err := withRetry(ctx, c.logger, func() error {
				var err error
				replies, hasMore, cursor, err = c.api.GetConversationRepliesContext(ctx, &slack.GetConversationRepliesParameters{
					ChannelID: channelID,
					Timestamp: parentTs.Raw(),
					Cursor:    cursor,
					Limit:     200,
				})
				return err
			})
			if err != nil {
				return fmt.Errorf("failed to get thread replies: %w", err)
			}

			for _, reply := range replies {
				if reply.Timestamp == parentTs.Raw() {
					continue
				}

				stats.trackUser(reply.User)
				stats.addReactions(reply.Reactions)

				replyMsg := buildExportMessage(reply, parentTs, getUserName(reply.User))
				if err := jw.WriteLine(replyMsg); err != nil {
					return err
				}
				stats.messageCount++
			}

			if !hasMore || cursor == "" {
				break
			}
		}
		return nil
	})
}

// ExportChannelOutput contains export statistics and file reference
type ExportChannelOutput struct {
	File          FileRef   `json:"file"`
	ThreadFiles   []FileRef `json:"thread_files,omitempty"`
	ChannelID     string    `json:"channel_id"`
	MessageCount  int       `json:"message_count"`
	ThreadCount   int       `json:"thread_count"`
	ReactionCount int       `json:"reaction_count"`
	UniqueUsers   int       `json:"unique_users"`
}

// ExportMessage represents a message in the export output
type ExportMessage struct {
	Timestamp       Timestamp      `json:"timestamp"`
	User            string         `json:"user"`
	UserName        string         `json:"user_name,omitempty"`
	Text            string         `json:"text"`
	ThreadTimestamp Timestamp      `json:"thread_ts,omitempty"`
	ReplyCount      int            `json:"reply_count,omitempty"`
	Reactions       []ReactionInfo `json:"reactions,omitempty"`
}

// ReactionInfo represents an emoji reaction with its count
type ReactionInfo struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ExportChannel exports a channel's messages to JSON-lines format.
// The main file contains top-level messages in chronological order (oldest first).
// Each thread gets its own separate file containing the parent and all replies.
func (c *Client) ExportChannel(ctx context.Context, req *mcp.CallToolRequest, input ExportChannelInput) (*mcp.CallToolResult, ExportChannelOutput, error) {
	channelID, err := c.GetChannelID(input.Channel)
	if err != nil {
		return nil, ExportChannelOutput{}, err
	}

	stats := newExportStats()
	userNames := make(map[string]string)

	getUserName := func(userID string) string {
		if userID == "" {
			return ""
		}
		if name, ok := userNames[userID]; ok {
			return name
		}
		var user *slack.User
		err := withRetry(ctx, c.logger, func() error {
			var err error
			user, err = c.api.GetUserInfoContext(ctx, userID)
			return err
		})
		if err == nil {
			userNames[userID] = user.Name
			return user.Name
		}
		return ""
	}

	ref, threadFiles, err := c.exportChannelTwoPass(ctx, channelID, input, getUserName, stats)
	if err != nil {
		return nil, ExportChannelOutput{}, err
	}

	return nil, ExportChannelOutput{
		File:          ref,
		ThreadFiles:   threadFiles,
		ChannelID:     channelID,
		MessageCount:  stats.messageCount,
		ThreadCount:   stats.threadCount,
		ReactionCount: stats.reactionCount,
		UniqueUsers:   len(stats.uniqueUsers),
	}, nil
}

// exportChannelTwoPass implements the two-pass export for chronological ordering.
// Pass 1: Write messages (newest-first from API) to temp file, tracking offsets
// Pass 2: Read temp file in reverse order, write to final file (oldest-first)
func (c *Client) exportChannelTwoPass(
	ctx context.Context,
	channelID string,
	input ExportChannelInput,
	getUserName func(string) string,
	stats *exportStats,
) (FileRef, []FileRef, error) {
	dir := c.responses.Dir()

	tmpPath, offsets, threadsToExport, err := c.writeHistoryToTempFile(ctx, dir, channelID, input, getUserName, stats)
	if err != nil {
		return FileRef{}, nil, err
	}
	defer os.Remove(tmpPath)

	var threadFiles []FileRef

	for _, msg := range threadsToExport {
		threadRef, err := c.writeThreadFile(ctx, channelID, msg, getUserName, stats)
		if err != nil {
			return FileRef{}, nil, fmt.Errorf("failed to write thread file: %w", err)
		}
		threadFiles = append(threadFiles, threadRef)
	}

	if len(offsets) == 0 {
		filename := fmt.Sprintf("export-%s-%d.jsonl", channelID, time.Now().UnixNano())
		filePath := filepath.Join(dir, filename)
		if err := os.WriteFile(filePath, nil, 0o644); err != nil {
			return FileRef{}, nil, fmt.Errorf("failed to create empty file: %w", err)
		}
		return FileRef{Path: filePath, Name: filename, Bytes: 0, Lines: 0}, threadFiles, nil
	}

	filename := fmt.Sprintf("export-%s-%d.jsonl", channelID, time.Now().UnixNano())
	filePath := filepath.Join(dir, filename)
	finalFile, err := os.Create(filePath)
	if err != nil {
		return FileRef{}, nil, fmt.Errorf("failed to create final file: %w", err)
	}
	defer finalFile.Close()

	tmpReader, err := os.Open(tmpPath)
	if err != nil {
		return FileRef{}, nil, fmt.Errorf("failed to reopen temp file: %w", err)
	}
	defer tmpReader.Close()

	if err := reverseCopyLines(tmpReader, finalFile, offsets); err != nil {
		return FileRef{}, nil, err
	}

	fi, err := finalFile.Stat()
	if err != nil {
		return FileRef{}, nil, fmt.Errorf("failed to stat final file: %w", err)
	}

	return FileRef{
		Path:  filePath,
		Name:  filename,
		Bytes: fi.Size(),
		Lines: len(offsets),
	}, threadFiles, nil
}

// writeHistoryToTempFile fetches channel history and writes messages to a temp file.
// Returns the temp file path, byte offsets for each line, and messages with threads.
func (c *Client) writeHistoryToTempFile(
	ctx context.Context,
	dir string,
	channelID string,
	input ExportChannelInput,
	getUserName func(string) string,
	stats *exportStats,
) (tmpPath string, offsets []int64, threadsToExport []slack.Message, err error) {
	tmpFile, err := os.CreateTemp(dir, "export-tmp-*.jsonl")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		tmpFile.Close()
		if err != nil {
			os.Remove(tmpFile.Name())
		}
	}()

	bw := bufio.NewWriter(tmpFile)
	var pos int64
	cursor := ""

	for {
		select {
		case <-ctx.Done():
			return "", nil, nil, ctx.Err()
		default:
		}

		var history *slack.GetConversationHistoryResponse
		err = withRetry(ctx, c.logger, func() error {
			var e error
			history, e = c.api.GetConversationHistoryContext(ctx, &slack.GetConversationHistoryParameters{
				ChannelID: channelID,
				Cursor:    cursor,
				Oldest:    input.Oldest,
				Latest:    input.Latest,
				Limit:     200,
			})
			return e
		})
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to get history: %w", err)
		}

		for _, msg := range history.Messages {
			stats.trackUser(msg.User)
			stats.addReactions(msg.Reactions)

			exportMsg := buildExportMessage(msg, "", getUserName(msg.User))
			b, err := json.Marshal(exportMsg)
			if err != nil {
				return "", nil, nil, fmt.Errorf("failed to marshal message: %w", err)
			}

			offsets = append(offsets, pos)
			n, err := bw.Write(b)
			if err != nil {
				return "", nil, nil, err
			}
			pos += int64(n)
			if err := bw.WriteByte('\n'); err != nil {
				return "", nil, nil, err
			}
			pos++
			stats.messageCount++

			if msg.ReplyCount > 0 {
				stats.threadCount++
				threadsToExport = append(threadsToExport, msg)
			}
		}

		if !history.HasMore || history.ResponseMetaData.NextCursor == "" {
			break
		}
		cursor = history.ResponseMetaData.NextCursor
	}

	if err = bw.Flush(); err != nil {
		return "", nil, nil, fmt.Errorf("failed to flush temp file: %w", err)
	}

	return tmpFile.Name(), offsets, threadsToExport, nil
}

// reverseCopyLines copies lines from src to dst in reverse order using pre-recorded offsets.
// Each offset marks the start of a line in src; lines are written to dst from last to first.
func reverseCopyLines(src *os.File, dst *os.File, offsets []int64) error {
	bw := bufio.NewWriter(dst)
	for i := len(offsets) - 1; i >= 0; i-- {
		if _, err := src.Seek(offsets[i], 0); err != nil {
			return fmt.Errorf("failed to seek: %w", err)
		}
		scanner := bufio.NewScanner(src)
		scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("failed to read line: %w", err)
			}
			continue
		}
		if _, err := bw.Write(scanner.Bytes()); err != nil {
			return err
		}
		if err := bw.WriteByte('\n'); err != nil {
			return err
		}
	}
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("failed to flush: %w", err)
	}
	return nil
}
