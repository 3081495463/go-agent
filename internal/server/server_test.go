package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-agent/internal/agent"
	"go-agent/internal/llm"
	"go-agent/internal/tools"
)

func newTestServer(t *testing.T, token string) *Server {
	t.Helper()
	registry := tools.NewRegistry()
	a := agent.NewAgent(llm.NewMockModel("mock reply"), registry)
	return New(a, token)
}

func TestHealth(t *testing.T) {
	s := newTestServer(t, "")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码不匹配: %d", w.Code)
	}
}

func TestChat(t *testing.T) {
	s := newTestServer(t, "")
	body := `{"message":"你好","session_id":"demo"}`
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码不匹配: %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["reply"] != "mock reply" {
		t.Fatalf("回复不匹配: %s", resp["reply"])
	}
	if resp["session_id"] != "demo" {
		t.Fatalf("session_id 未回传: %s", resp["session_id"])
	}
}

func TestChatUnauthorized(t *testing.T) {
	s := newTestServer(t, "secret")
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"message":"hi"}`))
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无令牌应返回 401，实际 %d", w.Code)
	}
}

func TestChatWrongMethod(t *testing.T) {
	s := newTestServer(t, "")
	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET 应返回 405，实际 %d", w.Code)
	}
}
