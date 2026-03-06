package slack

import (
	"os"
	"testing"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

// newServiceWithIndex creates a client with a pre-populated channel index (for testing)
func newServiceWithIndex(api SlackAPI, index *channelIndex, logger *zap.Logger, responses ResponseWriter) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	if index == nil {
		index = newIndex()
	}
	return &Service{
		api:       api,
		index:     index,
		logger:    logger,
		responses: responses,
	}
}

// newTestClient creates a test client with a mocked HTTP server
func newTestClient(t *testing.T, mock *mockSlackServer) (*Service, *testLogger, string) {
	t.Helper()

	api := slack.New("xoxb-test-token",
		slack.OptionAPIURL(mock.server.URL+"/"),
	)

	outputDir, err := os.MkdirTemp("", "slack-4-agents-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	logger := newTestLogger()
	responses := NewFileResponseWriter(outputDir)
	return newServiceWithIndex(api, nil, logger.Logger, responses), logger, outputDir
}
