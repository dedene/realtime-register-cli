package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockServer creates a test server with predefined responses.
type MockServer struct {
	*httptest.Server
	t        *testing.T
	Handlers map[string]http.HandlerFunc
}

// NewMockServer creates a mock API server.
func NewMockServer(t *testing.T) *MockServer {
	m := &MockServer{
		t:        t,
		Handlers: make(map[string]http.HandlerFunc),
	}
	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if h, ok := m.Handlers[key]; ok {
			h(w, r)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	return m
}

// On registers a handler for method + path.
func (m *MockServer) On(method, path string, handler http.HandlerFunc) {
	m.Handlers[method+" "+path] = handler
}

// OnJSON registers a handler that returns JSON.
func (m *MockServer) OnJSON(method, path string, status int, body any) {
	m.On(method, path, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			if err := json.NewEncoder(w).Encode(body); err != nil {
				m.t.Errorf("failed to encode response: %v", err)
			}
		}
	})
}

// Client returns a Client configured to use the mock server.
func (m *MockServer) Client() *Client {
	c := NewClient("test-api-key")
	c.SetBaseURL(m.URL)
	return c
}
