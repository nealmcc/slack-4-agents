package slackapi

import (
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestNewClient_WithoutCookie(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := NewClient("xoxb-test-token", "", logger)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestNewClient_WithCookie(t *testing.T) {
	logger := zaptest.NewLogger(t)
	client := NewClient("xoxc-test-token", "test-cookie", logger)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
}
