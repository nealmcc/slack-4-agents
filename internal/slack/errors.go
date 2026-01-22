package slack

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// authErrorCodes are Slack API error codes that indicate authentication problems
var authErrorCodes = map[string]string{
	"invalid_auth":     "Authentication token is invalid. Please refresh your SLACK_TOKEN and SLACK_COOKIE.",
	"token_expired":    "Authentication token has expired. Please refresh your SLACK_TOKEN and SLACK_COOKIE.",
	"token_revoked":    "Authentication token has been revoked. Please generate new credentials.",
	"account_inactive": "The Slack account is inactive or disabled.",
	"not_authed":       "No authentication token provided. Please set SLACK_TOKEN and SLACK_COOKIE.",
}

// AuthError represents a Slack authentication error with guidance for resolution
type AuthError struct {
	Code    string
	Message string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("SLACK AUTHENTICATION ERROR: %s (code: %s)", e.Message, e.Code)
}

// matchAuthError checks if an error contains an auth error code.
// Returns nil if no auth error is found.
func matchAuthError(err error) *AuthError {
	if err == nil {
		return nil
	}
	errStr := err.Error()
	for code, message := range authErrorCodes {
		if strings.Contains(errStr, code) {
			return &AuthError{Code: code, Message: message}
		}
	}
	return nil
}

// WrapError checks for auth errors and returns an enhanced error with logging.
// This should be called at the API boundary (e.g., MCP layer) to provide
// clear error messages to callers.
func WrapError(logger *zap.Logger, operation string, err error) error {
	if err == nil {
		return nil
	}

	if authErr := matchAuthError(err); authErr != nil {
		logger.Error("Slack authentication failed",
			zap.String("operation", operation),
			zap.String("guidance", authErr.Message),
			zap.Error(err))
		return authErr
	}

	return fmt.Errorf("%s: %w", operation, err)
}
