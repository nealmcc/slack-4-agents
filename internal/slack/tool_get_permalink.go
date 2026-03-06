package slack

import (
	"context"
	"fmt"

	"github.com/slack-go/slack"
)

// GetPermalinkInput defines input for getting a message permalink
type GetPermalinkInput struct {
	Channel   string `json:"channel" jsonschema:"Channel ID (e.g., C1234567890)"`
	Timestamp string `json:"timestamp" jsonschema:"Message timestamp (e.g., 1234567890.123456)"`
}

// GetPermalinkOutput contains the permalink
type GetPermalinkOutput struct {
	Permalink string `json:"permalink"`
	Channel   string `json:"channel"`
	Timestamp string `json:"timestamp"`
}

// GetPermalink gets a permalink to a specific message
func (c *Service) GetPermalink(ctx context.Context, input GetPermalinkInput) (GetPermalinkOutput, error) {
	channelID, err := c.GetChannelID(input.Channel)
	if err != nil {
		return GetPermalinkOutput{}, err
	}

	permalink, err := c.api.GetPermalinkContext(ctx, &slack.PermalinkParameters{
		Channel: channelID,
		Ts:      input.Timestamp,
	})
	if err != nil {
		return GetPermalinkOutput{}, fmt.Errorf("failed to get permalink: %w", err)
	}

	return GetPermalinkOutput{
		Permalink: permalink,
		Channel:   channelID,
		Timestamp: input.Timestamp,
	}, nil
}
