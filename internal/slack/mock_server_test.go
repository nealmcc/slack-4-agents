package slack

import (
	"net/http"
	"net/http/httptest"
)

// mockSlackServer creates a test HTTP server that mocks Slack API responses
type mockSlackServer struct {
	server   *httptest.Server
	handlers map[string]http.HandlerFunc
}

func newMockSlackServer() *mockSlackServer {
	m := &mockSlackServer{
		handlers: make(map[string]http.HandlerFunc),
	}

	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Try exact match first
		if handler, ok := m.handlers[path]; ok {
			handler(w, r)
			return
		}

		// Not found
		http.Error(w, "mock not found: "+path, http.StatusNotFound)
	}))

	return m
}

func (m *mockSlackServer) close() {
	m.server.Close()
}

func (m *mockSlackServer) addHandler(path string, handler http.HandlerFunc) {
	m.handlers[path] = handler
}
