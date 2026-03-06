package slackapi

import (
	"net/http"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

// NewClient creates a new Slack API client with optional cookie authentication
func NewClient(token, cookie string, logger *zap.Logger) *slack.Client {
	opts := []slack.Option{}

	if cookie != "" {
		logger.Info("Using cookie authentication for Slack client")
		httpClient := &http.Client{
			Transport: newCookieTransport(cookie, logger),
		}
		opts = append(opts, slack.OptionHTTPClient(httpClient))
	}

	return slack.New(token, opts...)
}
