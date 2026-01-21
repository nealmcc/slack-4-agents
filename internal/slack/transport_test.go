package slack

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

func TestCookieTransport_RoundTrip(t *testing.T) {
	var capturedCookie string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zap.NewNop()
	transport := newCookieTransport("test-cookie-value", logger)

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	defer resp.Body.Close()

	want := "d=test-cookie-value"
	if capturedCookie != want {
		t.Errorf("Cookie header: got %q, want %q", capturedCookie, want)
	}
}

func TestNewCookieTransport(t *testing.T) {
	logger := zap.NewNop()
	transport := newCookieTransport("my-cookie", logger)

	if transport.cookie != "my-cookie" {
		t.Errorf("cookie: got %q, want %q", transport.cookie, "my-cookie")
	}

	if transport.transport != http.DefaultTransport {
		t.Error("transport: expected http.DefaultTransport")
	}

	if transport.logger != logger {
		t.Error("logger: expected provided logger")
	}
}

func TestWithRetry_SuccessOnFirstTry(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	callCount := 0
	err := withRetry(ctx, logger, func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("withRetry returned error: %v", err)
	}

	wantCalls := 1
	if callCount != wantCalls {
		t.Errorf("call count: got %d, want %d", callCount, wantCalls)
	}
}

func TestWithRetry_NonRateLimitError(t *testing.T) {
	logger := zap.NewNop()
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
	logger := zap.NewNop()
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
		t.Errorf("withRetry returned error: %v", err)
	}

	wantCalls := 2
	if callCount != wantCalls {
		t.Errorf("call count: got %d, want %d", callCount, wantCalls)
	}
}

func TestWithRetry_ContextCancelledDuringWait(t *testing.T) {
	logger := zap.NewNop()
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
	logger := zap.NewNop()
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
