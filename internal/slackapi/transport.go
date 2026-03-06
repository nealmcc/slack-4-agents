package slackapi

import (
	"net/http"
	"path"

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
