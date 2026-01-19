package slack

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

// cookieTransport wraps an http.RoundTripper to add cookie headers
type cookieTransport struct {
	transport http.RoundTripper
	cookie    string
	logger    *zap.Logger
}

func (t *cookieTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Cookie", "d="+t.cookie)
	t.logger.Debug("Making HTTP request with cookie authentication",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()))
	return t.transport.RoundTrip(req)
}

// newCookieTransport creates a transport with cookie authentication
func newCookieTransport(cookie string, logger *zap.Logger) *cookieTransport {
	return &cookieTransport{
		transport: http.DefaultTransport,
		cookie:    cookie,
		logger:    logger,
	}
}

// getConversationsWithRetry fetches conversations and handles rate limiting
// by respecting the Retry-After header and automatically retrying
func (c *Client) getConversationsWithRetry(ctx context.Context, params *slack.GetConversationsParameters) ([]slack.Channel, string, error) {
	for {
		channels, cursor, err := c.api.GetConversationsContext(ctx, params)
		// Check if this is a rate limit error
		if err != nil {
			var rateLimitErr *slack.RateLimitedError
			if errors.As(err, &rateLimitErr) {
				c.logger.Warn("Rate limit hit, waiting before retry",
					zap.Duration("retry_after", rateLimitErr.RetryAfter))
				// Wait for the duration specified in Retry-After header
				select {
				case <-time.After(rateLimitErr.RetryAfter):
					c.logger.Info("Retrying after rate limit wait")
					// Retry the request
					continue
				case <-ctx.Done():
					c.logger.Debug("Context cancelled during rate limit wait")
					return nil, "", ctx.Err()
				}
			}
			// Check if context was cancelled (expected when stopping pagination early)
			if errors.Is(err, context.Canceled) {
				c.logger.Debug("Context cancelled during API call")
				return nil, "", err
			}
			// Non-rate-limit error, return it
			c.logger.Error("API error fetching conversations", zap.Error(err))
			return nil, "", err
		}

		// Success, return the results
		c.logger.Debug("Successfully fetched conversations page",
			zap.Int("channel_count", len(channels)),
			zap.Bool("has_more", cursor != ""))
		return channels, cursor, nil
	}
}
