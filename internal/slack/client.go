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

// cookieTransport wraps an http.RoundTripper to add cookie headers
type cookieTransport struct {
	transport http.RoundTripper
	cookie    string
}

func (t *cookieTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Cookie", "d="+t.cookie)
	return t.transport.RoundTrip(req)
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
			Transport: &cookieTransport{
				transport: http.DefaultTransport,
				cookie:    cookie,
			},
		}
		opts = append(opts, slack.OptionHTTPClient(httpClient))
	}

	api := slack.New(token, opts...)

	return &Client{
		api:       api,
		channelID: make(map[string]string),
	}, nil
}

// ResolveChannelID converts a channel name to its ID, or returns the ID if already provided
func (c *Client) ResolveChannelID(ctx context.Context, channel string) (string, error) {
	// If it looks like an ID (starts with C, D, or G), return as-is
	if strings.HasPrefix(channel, "C") || strings.HasPrefix(channel, "D") || strings.HasPrefix(channel, "G") {
		return channel, nil
	}

	// Strip # prefix if present
	channel = strings.TrimPrefix(channel, "#")

	// Check cache
	if id, ok := c.channelID[channel]; ok {
		return id, nil
	}

	// Search for the channel
	channels, _, err := c.api.GetConversationsContext(ctx, &slack.GetConversationsParameters{
		Types:           []string{"public_channel", "private_channel"},
		ExcludeArchived: true,
		Limit:           1000,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list channels: %w", err)
	}

	for _, ch := range channels {
		c.channelID[ch.Name] = ch.ID
		if ch.Name == channel {
			return ch.ID, nil
		}
	}

	return "", fmt.Errorf("channel not found: %s", channel)
}

// API returns the underlying slack client for direct access
func (c *Client) API() *slack.Client {
	return c.api
}
