package slack

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

// SlackAPI defines the Slack API methods used by the client
//
//go:generate go tool mockgen -source=$GOFILE -destination=service_mocks.go -package=slack
type SlackAPI interface {
	GetConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) ([]slack.Channel, string, error)
	GetConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error)
	GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error)
	GetUserInfoContext(ctx context.Context, user string) (*slack.User, error)
	GetUserByEmailContext(ctx context.Context, email string) (*slack.User, error)
	SearchMessagesContext(ctx context.Context, query string, params slack.SearchParameters) (*slack.SearchMessages, error)
	GetPermalinkContext(ctx context.Context, params *slack.PermalinkParameters) (string, error)
	GetFileInfoContext(ctx context.Context, fileID string, count int, page int) (*slack.File, []slack.Comment, *slack.Paging, error)
	GetFileContext(ctx context.Context, downloadURL string, writer io.Writer) error
}

// FileRef describes a file written by ResponseWriter
type FileRef struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
	Lines int    `json:"lines"`
}

// JSONLineWriter provides streaming writes for JSON-lines format
type JSONLineWriter interface {
	WriteLine(data any) error
}

// ResponseWriter writes large response data to a file and returns a reference
type ResponseWriter interface {
	WriteJSON(name string, data any) (FileRef, error)
	WriteJSONLines(name string, writeFn func(w JSONLineWriter) error) (FileRef, error)
	WriteJSONLinesNamed(filename string, writeFn func(w JSONLineWriter) error) (FileRef, error)
	WriteText(name string, content string) (FileRef, error)
	Dir() string
}

type Service struct {
	api       SlackAPI
	index     *channelIndex
	logger    *zap.Logger
	responses ResponseWriter
}

// NewService creates a service-layer client with pre-built dependencies
func NewService(api SlackAPI, logger *zap.Logger, responses ResponseWriter) *Service {
	return &Service{
		api:       api,
		index:     newIndex(),
		logger:    logger,
		responses: responses,
	}
}

// GetChannelID accepts either a channel name or ID and returns the channel ID
func (c *Service) GetChannelID(channelOrName string) (string, error) {
	if isChannelID(channelOrName) {
		return channelOrName, nil
	}
	return c.findChannelID(channelOrName)
}

// isChannelID checks if a string looks like a Slack channel ID
// Channel IDs are uppercase alphanumeric strings starting with C, D, or G
// and are typically 9-11 characters long
func isChannelID(s string) bool {
	if len(s) < 9 {
		return false
	}

	// Must start with C, D, or G
	if s[0] != 'C' && s[0] != 'D' && s[0] != 'G' {
		return false
	}

	// Must be all uppercase alphanumeric
	for _, ch := range s {
		if !((ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
			return false
		}
	}

	return true
}

// listConversations wraps the Slack API call and feeds the channel index.
func (c *Service) listConversations(ctx context.Context, params *slack.GetConversationsParameters) ([]slack.Channel, string, error) {
	channels, cursor, err := c.api.GetConversationsContext(ctx, params)
	if err != nil {
		return nil, "", err
	}
	c.index.Add(channels)
	return channels, cursor, nil
}

// searchMessages wraps the Slack API call and feeds the channel index.
func (c *Service) searchMessages(ctx context.Context, query string, params slack.SearchParameters) (*slack.SearchMessages, error) {
	results, err := c.api.SearchMessagesContext(ctx, query, params)
	if err != nil {
		return nil, err
	}
	channels := make([]slack.Channel, 0, len(results.Matches))
	for _, match := range results.Matches {
		channels = append(channels, slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{
					ID:             match.Channel.ID,
					NameNormalized: match.Channel.Name,
				},
				Name: match.Channel.Name,
			},
		})
	}
	c.index.Add(channels)
	return results, nil
}

// getConversationInfo wraps the Slack API call and feeds the channel index.
func (c *Service) getConversationInfo(ctx context.Context, channelID string) (*slack.Channel, error) {
	var ch *slack.Channel
	err := withRetry(ctx, c.logger, func() error {
		var e error
		ch, e = c.api.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{
			ChannelID: channelID,
		})
		return e
	})
	if err != nil {
		return nil, err
	}
	c.index.Add([]slack.Channel{*ch})
	return ch, nil
}

// findChannelID looks up a channel name in the index
func (c *Service) findChannelID(name string) (string, error) {
	name = strings.TrimPrefix(name, "#")

	ch, ok := c.index.GetByName(name)
	if !ok {
		return "", fmt.Errorf("channel %q not found in index (%d entries); use a channel ID or call slack_list_channels first", name, c.index.Size())
	}

	c.logger.Debug("Channel found in index",
		zap.String("channel_name", ch.NameNormalized),
		zap.String("channel_id", ch.ID))

	return ch.ID, nil
}
