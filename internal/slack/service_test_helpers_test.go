package slack

import "go.uber.org/zap"

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
