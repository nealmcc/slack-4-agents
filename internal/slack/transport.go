package slack

import (
	"context"
	"errors"
	"net/http"
	"path"
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
	t.logger.Debug("Slack API request",
		zap.String("api_method", path.Base(req.URL.Path)),
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

// withRetry executes fn and automatically retries on Slack rate limit errors.
// The fn closure should perform the API call and return any error.
// Results should be captured in variables in the outer scope.
func withRetry(ctx context.Context, logger *zap.Logger, fn func() error) error {
	for {
		err := fn()
		if err == nil {
			return nil
		}

		var rateLimitErr *slack.RateLimitedError
		if errors.As(err, &rateLimitErr) {
			logger.Warn("Rate limit hit, waiting before retry",
				zap.Duration("retry_after", rateLimitErr.RetryAfter))
			select {
			case <-time.After(rateLimitErr.RetryAfter):
				logger.Info("Retrying after rate limit wait")
				continue
			case <-ctx.Done():
				logger.Debug("Context cancelled during rate limit wait")
				return ctx.Err()
			}
		}

		if errors.Is(err, context.Canceled) {
			logger.Debug("Context cancelled during API call")
		}
		return err
	}
}
