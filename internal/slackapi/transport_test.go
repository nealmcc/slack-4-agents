package slackapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestCookieTransport_RoundTrip(t *testing.T) {
	var capturedCookie string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t)
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
	logger := zaptest.NewLogger(t)
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
