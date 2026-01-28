package slack

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

// SlackAPI defines the Slack API methods used by the client
//
//go:generate go tool mockgen -source=$GOFILE -destination=client_mocks.go -package=slack
type SlackAPI interface {
	GetConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) ([]slack.Channel, string, error)
	GetConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error)
	GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error)
	GetUserInfoContext(ctx context.Context, user string) (*slack.User, error)
	GetUserByEmailContext(ctx context.Context, email string) (*slack.User, error)
	SearchMessagesContext(ctx context.Context, query string, params slack.SearchParameters) (*slack.SearchMessages, error)
	GetPermalinkContext(ctx context.Context, params *slack.PermalinkParameters) (string, error)
}

// Config holds configuration for the Slack client
type Config struct {
	Token    string // Slack API token (required)
	Cookie   string // Slack cookie for xoxc token auth (optional)
	LogLevel string // "debug", "info", "warn", "error"
	WorkDir  string // the path to the working directory for this client
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
	Dir() string
}

type Client struct {
	api       SlackAPI
	channelID map[string]string // cache: name -> ID
	logger    *zap.Logger
	responses ResponseWriter
}

func NewClient(cfg Config, logger *zap.Logger, responses ResponseWriter) (*Client, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("slack token is required")
	}

	opts := []slack.Option{}

	if cfg.Cookie != "" {
		logger.Info("Using cookie authentication for Slack client")
		httpClient := &http.Client{
			Transport: newCookieTransport(cfg.Cookie, logger),
		}
		opts = append(opts, slack.OptionHTTPClient(httpClient))
	}

	api := slack.New(cfg.Token, opts...)

	logger.Info("Slack client initialized successfully")

	return &Client{
		api:       api,
		channelID: make(map[string]string),
		logger:    logger,
		responses: responses,
	}, nil
}

// newClientWithAPI creates a client with an existing Slack API client (for testing)
func newClientWithAPI(api SlackAPI, logger *zap.Logger, responses ResponseWriter) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Client{
		api:       api,
		channelID: make(map[string]string),
		logger:    logger,
		responses: responses,
	}
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

// GetChannelID accepts either a channel name or ID and returns the channel ID
// If given an ID (e.g., "CTKV7RT5Z"), validates it exists and returns it
// If given a name (e.g., "dogs" or "#general"), looks it up
func (c *Client) GetChannelID(ctx context.Context, channelOrName string) (string, error) {
	// If it's already an ID, validate it exists using conversations API
	if isChannelID(channelOrName) {
		c.logger.Debug("Validating channel ID", zap.String("channel_id", channelOrName))
		channel, err := c.api.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{
			ChannelID: channelOrName,
		})
		if err != nil {
			c.logger.Error("Failed to validate channel ID",
				zap.String("channel_id", channelOrName),
				zap.Error(err))
			return "", fmt.Errorf("invalid channel ID: %w", err)
		}
		// Cache the name -> ID mapping for future lookups
		c.channelID[strings.ToLower(channel.Name)] = channel.ID
		c.logger.Debug("Channel ID validated and cached",
			zap.String("channel_id", channel.ID),
			zap.String("channel_name", channel.Name))
		return channel.ID, nil
	}

	// Otherwise, resolve the name to an ID
	return c.findChannelID(ctx, channelOrName)
}

// channelPage represents a page of channels from the API
type channelPage struct {
	channels []slack.Channel
	err      error
}

// findChannelID converts a channel name to its ID
func (c *Client) findChannelID(ctx context.Context, name string) (string, error) {
	// Strip # prefix if present
	name = strings.TrimPrefix(name, "#")

	// Convert to lowercase for case-insensitive lookup
	name = strings.ToLower(name)

	// Check cache
	if id, ok := c.channelID[name]; ok {
		c.logger.Debug("Channel found in cache",
			zap.String("channel_name", name),
			zap.String("channel_id", id))
		return id, nil
	}

	c.logger.Info("Channel not in cache, starting pagination",
		zap.String("channel_name", name))

	// Create channels for communication between goroutines
	pages := make(chan channelPage, 1) // Buffer allows fetcher to send while processor works
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start fetcher goroutine to paginate through API results
	go func() {
		defer close(pages)
		cursor := ""
		for {
			// Check if context was cancelled (target found or error occurred)
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Fetch page with automatic rate limit handling
			var channels []slack.Channel
			var nextCursor string
			err := withRetry(ctx, c.logger, func() error {
				var err error
				channels, nextCursor, err = c.api.GetConversationsContext(ctx, &slack.GetConversationsParameters{
					Types:           []string{"public_channel", "private_channel"},
					ExcludeArchived: true,
					Limit:           1000,
					Cursor:          cursor,
				})
				return err
			})

			// Send page result (success or error)
			select {
			case pages <- channelPage{channels: channels, err: err}:
			case <-ctx.Done():
				return
			}

			if err != nil || nextCursor == "" {
				return
			}
			cursor = nextCursor
		}
	}()

	// Process pages as they arrive
	pageCount := 0
	channelsProcessed := 0
	for page := range pages {
		if page.err != nil {
			c.logger.Error("Failed to list channels",
				zap.String("channel_name", name),
				zap.Error(page.err))
			return "", fmt.Errorf("failed to list channels: %w", page.err)
		}

		pageCount++
		channelsProcessed += len(page.channels)
		c.logger.Debug("Processing channel page",
			zap.Int("page_number", pageCount),
			zap.Int("channels_in_page", len(page.channels)),
			zap.Int("total_processed", channelsProcessed))

		// Add all channels from this page to cache and check for target
		for _, ch := range page.channels {
			c.channelID[ch.Name] = ch.ID
			if ch.Name == name {
				cancel() // Stop the fetcher goroutine
				c.logger.Info("Channel found",
					zap.String("channel_name", name),
					zap.String("channel_id", ch.ID),
					zap.Int("pages_searched", pageCount),
					zap.Int("channels_processed", channelsProcessed))
				return ch.ID, nil
			}
		}
	}

	c.logger.Warn("Channel not found after pagination",
		zap.String("channel_name", name),
		zap.Int("pages_searched", pageCount),
		zap.Int("channels_processed", channelsProcessed))
	return "", fmt.Errorf("channel not found: %s", name)
}
