package slack

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/slack-go/slack"
)

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
func (c *Service) writeThreadFile(
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
func (c *Service) ExportChannel(ctx context.Context, input ExportChannelInput) (ExportChannelOutput, error) {
	channelID, err := c.GetChannelID(input.Channel)
	if err != nil {
		return ExportChannelOutput{}, err
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
		return ExportChannelOutput{}, err
	}

	return ExportChannelOutput{
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
func (c *Service) exportChannelTwoPass(
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
func (c *Service) writeHistoryToTempFile(
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
