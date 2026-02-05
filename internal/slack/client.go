package slack

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/slack-go/slack"
)

type Client struct {
	api       *slack.Client
	channelID map[string]string // cache: name -> ID
}

func NewClient() (*Client, error) {
	token := os.Getenv("SLACK_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("SLACK_TOKEN environment variable required")
	}

	opts := []slack.Option{}

	// Support xoxc tokens with cookie authentication
	if cookie := os.Getenv("SLACK_COOKIE"); cookie != "" {
		httpClient := &http.Client{
			Transport: newCookieTransport(cookie),
		}
		opts = append(opts, slack.OptionHTTPClient(httpClient))
	}

	api := slack.New(token, opts...)

	return &Client{
		api:       api,
		channelID: make(map[string]string),
	}, nil
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
		channel, err := c.api.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{
			ChannelID: channelOrName,
		})
		if err != nil {
			return "", fmt.Errorf("invalid channel ID: %w", err)
		}
		// Cache the name -> ID mapping for future lookups
		c.channelID[strings.ToLower(channel.Name)] = channel.ID
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
		return id, nil
	}

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
			channels, nextCursor, err := c.getConversationsWithRetry(ctx, &slack.GetConversationsParameters{
				Types:           []string{"public_channel", "private_channel"},
				ExcludeArchived: true,
				Limit:           1000,
				Cursor:          cursor,
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
	for page := range pages {
		if page.err != nil {
			return "", fmt.Errorf("failed to list channels: %w", page.err)
		}

		// Add all channels from this page to cache and check for target
		for _, ch := range page.channels {
			c.channelID[ch.Name] = ch.ID
			if ch.Name == name {
				cancel() // Stop the fetcher goroutine
				return ch.ID, nil
			}
		}
	}

	return "", fmt.Errorf("channel not found: %s", name)
}
