package slack

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap/zaptest"
)

func TestWithRetry_SuccessOnFirstTry(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	callCount := 0
	err := withRetry(ctx, logger, func() error {
		callCount++
		return nil
	})
	if err != nil {
		t.Errorf("WithRetry returned error: %v", err)
	}

	wantCalls := 1
	if callCount != wantCalls {
		t.Errorf("call count: got %d, want %d", callCount, wantCalls)
	}
}

func TestWithRetry_NonRateLimitError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	expectedErr := errors.New("some other error")
	callCount := 0
	err := withRetry(ctx, logger, func() error {
		callCount++
		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Errorf("error: got %v, want %v", err, expectedErr)
	}

	wantCalls := 1
	if callCount != wantCalls {
		t.Errorf("call count: got %d, want %d", callCount, wantCalls)
	}
}

func TestWithRetry_RateLimitThenSuccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()

	callCount := 0
	err := withRetry(ctx, logger, func() error {
		callCount++
		if callCount == 1 {
			return &slack.RateLimitedError{RetryAfter: 1 * time.Millisecond}
		}
		return nil
	})
	if err != nil {
		t.Errorf("WithRetry returned error: %v", err)
	}

	wantCalls := 2
	if callCount != wantCalls {
		t.Errorf("call count: got %d, want %d", callCount, wantCalls)
	}
}

func TestWithRetry_ContextCancelledDuringWait(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	err := withRetry(ctx, logger, func() error {
		callCount++
		if callCount == 1 {
			cancel()
			return &slack.RateLimitedError{RetryAfter: 1 * time.Hour}
		}
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error: got %v, want context.Canceled", err)
	}

	wantCalls := 1
	if callCount != wantCalls {
		t.Errorf("call count: got %d, want %d", callCount, wantCalls)
	}
}

func TestWithRetry_ContextAlreadyCancelled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	callCount := 0
	err := withRetry(ctx, logger, func() error {
		callCount++
		return context.Canceled
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error: got %v, want context.Canceled", err)
	}

	wantCalls := 1
	if callCount != wantCalls {
		t.Errorf("call count: got %d, want %d", callCount, wantCalls)
	}
}
