package slack

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/slack-go/slack"
)

// cookieTransport wraps an http.RoundTripper to add cookie headers
type cookieTransport struct {
	transport http.RoundTripper
	cookie    string
}

func (t *cookieTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Cookie", "d="+t.cookie)
	return t.transport.RoundTrip(req)
}

// newCookieTransport creates a transport with cookie authentication
func newCookieTransport(cookie string) *cookieTransport {
	return &cookieTransport{
		transport: http.DefaultTransport,
		cookie:    cookie,
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
				// Wait for the duration specified in Retry-After header
				select {
				case <-time.After(rateLimitErr.RetryAfter):
					// Retry the request
					continue
				case <-ctx.Done():
					return nil, "", ctx.Err()
				}
			}
			// Non-rate-limit error, return it
			return nil, "", err
		}

		// Success, return the results
		return channels, cursor, nil
	}
}
