package slack

import (
	"context"
	"errors"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

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
		} else if errors.Is(err, context.Canceled) {
			logger.Debug("Context cancelled during API call")
		}
		return err
	}
}
