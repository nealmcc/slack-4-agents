package slack

import (
	"errors"
	"testing"

	"go.uber.org/zap"
)

func TestMatchAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode string
		wantMsg  string
	}{
		{
			name:     "invalid_auth error",
			err:      errors.New("invalid_auth"),
			wantCode: "invalid_auth",
			wantMsg:  "Authentication token is invalid. Please refresh your SLACK_TOKEN and SLACK_COOKIE.",
		},
		{
			name:     "token_expired error",
			err:      errors.New("token_expired"),
			wantCode: "token_expired",
			wantMsg:  "Authentication token has expired. Please refresh your SLACK_TOKEN and SLACK_COOKIE.",
		},
		{
			name:     "token_revoked error",
			err:      errors.New("token_revoked"),
			wantCode: "token_revoked",
			wantMsg:  "Authentication token has been revoked. Please generate new credentials.",
		},
		{
			name:     "not_authed error",
			err:      errors.New("not_authed"),
			wantCode: "not_authed",
			wantMsg:  "No authentication token provided. Please set SLACK_TOKEN and SLACK_COOKIE.",
		},
		{
			name:     "wrapped auth error",
			err:      errors.New("slack api: invalid_auth"),
			wantCode: "invalid_auth",
			wantMsg:  "Authentication token is invalid. Please refresh your SLACK_TOKEN and SLACK_COOKIE.",
		},
		{
			name:     "non-auth error",
			err:      errors.New("channel_not_found"),
			wantCode: "",
			wantMsg:  "",
		},
		{
			name:     "nil error",
			err:      nil,
			wantCode: "",
			wantMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchAuthError(tt.err)
			if tt.wantCode == "" {
				if got != nil {
					t.Errorf("matchAuthError() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("matchAuthError() = nil, want AuthError")
			}
			if got.Code != tt.wantCode {
				t.Errorf("matchAuthError().Code = %q, want %q", got.Code, tt.wantCode)
			}
			if got.Message != tt.wantMsg {
				t.Errorf("matchAuthError().Message = %q, want %q", got.Message, tt.wantMsg)
			}
		})
	}
}

func TestWrapError_AuthError(t *testing.T) {
	logger := zap.NewNop()
	err := errors.New("invalid_auth")

	wrapped := WrapError(logger, "test operation", err)

	var authErr *AuthError
	if !errors.As(wrapped, &authErr) {
		t.Fatalf("expected AuthError, got %T", wrapped)
	}

	if authErr.Code != "invalid_auth" {
		t.Errorf("Code: got %q, want %q", authErr.Code, "invalid_auth")
	}

	wantMsg := "Authentication token is invalid. Please refresh your SLACK_TOKEN and SLACK_COOKIE."
	if authErr.Message != wantMsg {
		t.Errorf("Message: got %q, want %q", authErr.Message, wantMsg)
	}
}

func TestWrapError_NonAuthError(t *testing.T) {
	logger := zap.NewNop()
	originalErr := errors.New("channel_not_found")

	wrapped := WrapError(logger, "test operation", originalErr)

	var authErr *AuthError
	if errors.As(wrapped, &authErr) {
		t.Fatalf("expected non-AuthError, got AuthError")
	}

	wantErrStr := "test operation: channel_not_found"
	if wrapped.Error() != wantErrStr {
		t.Errorf("error string: got %q, want %q", wrapped.Error(), wantErrStr)
	}
}

func TestWrapError_NilError(t *testing.T) {
	logger := zap.NewNop()

	wrapped := WrapError(logger, "test operation", nil)

	if wrapped != nil {
		t.Errorf("expected nil, got %v", wrapped)
	}
}

func TestAuthError_Error(t *testing.T) {
	err := &AuthError{
		Code:    "invalid_auth",
		Message: "Test message",
	}

	want := "SLACK AUTHENTICATION ERROR: Test message (code: invalid_auth)"
	if got := err.Error(); got != want {
		t.Errorf("Error(): got %q, want %q", got, want)
	}
}
