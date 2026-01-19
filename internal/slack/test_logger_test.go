package slack

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// testLogger wraps a zap logger with an observer for testing
type testLogger struct {
	*zap.Logger
	observer *observer.ObservedLogs
}

// newTestLogger creates a logger that captures all log entries for testing
func newTestLogger() *testLogger {
	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)
	return &testLogger{
		Logger:   logger,
		observer: logs,
	}
}

// LoggedMessages returns all messages logged at or above the given level
func (tl *testLogger) LoggedMessages(level zapcore.Level) []string {
	var messages []string
	for _, entry := range tl.observer.FilterLevelExact(level).All() {
		messages = append(messages, entry.Message)
	}
	return messages
}

// HasMessage checks if a specific message was logged at any level
func (tl *testLogger) HasMessage(msg string) bool {
	for _, entry := range tl.observer.All() {
		if entry.Message == msg {
			return true
		}
	}
	return false
}

// AllMessages returns all logged messages regardless of level
func (tl *testLogger) AllMessages() []string {
	var messages []string
	for _, entry := range tl.observer.All() {
		messages = append(messages, entry.Message)
	}
	return messages
}
