package slack

import (
	"context"
	"fmt"

	"github.com/slack-go/slack"
)

// GetUserInput defines input for getting user info
type GetUserInput struct {
	User  string `json:"user,omitempty" jsonschema:"User ID (e.g., U1234567890)"`
	Email string `json:"email,omitempty" jsonschema:"User email address"`
}

// UserInfo represents a Slack user
type UserInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	RealName    string `json:"real_name"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email,omitempty"`
	Title       string `json:"title,omitempty"`
	Status      string `json:"status,omitempty"`
	StatusEmoji string `json:"status_emoji,omitempty"`
	IsBot       bool   `json:"is_bot"`
	IsAdmin     bool   `json:"is_admin"`
	Timezone    string `json:"timezone,omitempty"`
}

// GetUserOutput contains user information
type GetUserOutput struct {
	User UserInfo `json:"user"`
}

// GetUser looks up user information by ID or email
func (c *Service) GetUser(ctx context.Context, input GetUserInput) (GetUserOutput, error) {
	var user *slack.User
	var err error

	if input.User != "" {
		user, err = c.api.GetUserInfoContext(ctx, input.User)
	} else if input.Email != "" {
		user, err = c.api.GetUserByEmailContext(ctx, input.Email)
	} else {
		return GetUserOutput{}, fmt.Errorf("either user ID or email is required")
	}

	if err != nil {
		return GetUserOutput{}, fmt.Errorf("failed to get user: %w", err)
	}

	output := GetUserOutput{
		User: UserInfo{
			ID:          user.ID,
			Name:        user.Name,
			RealName:    user.RealName,
			DisplayName: user.Profile.DisplayName,
			Email:       user.Profile.Email,
			Title:       user.Profile.Title,
			Status:      user.Profile.StatusText,
			StatusEmoji: user.Profile.StatusEmoji,
			IsBot:       user.IsBot,
			IsAdmin:     user.IsAdmin,
			Timezone:    user.TZ,
		},
	}

	return output, nil
}
