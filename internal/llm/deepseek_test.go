package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, baseURL string) *DeepseekClient {
	t.Helper()
	c, err := NewDeepseekClient("test-key", "deepseek-chat", baseURL)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNewDeepseekClientEmptyKey(t *testing.T) {
	if _, err := NewDeepseekClient("", "", ""); err == nil {
		t.Fatal("空 API Key 应报错")
	}
}

func TestDeepseekChatSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("鉴权头不正确: %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{"message": {"role": "assistant", "content": "hello"}}],
			"usage": {"total_tokens": 12}
		}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	msg, tokens, err := c.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "hello" {
		t.Fatalf("内容不匹配: %s", msg.Content)
	}
	if tokens != 12 {
		t.Fatalf("token 数不匹配: %d", tokens)
	}
}

func TestDeepseekChatAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": {"message": "invalid key"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, _, err := c.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("非 2xx 响应应报错")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("错误应包含状态码 401: %v", err)
	}
}

func TestDeepseekChatEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices": [], "usage": {"total_tokens": 0}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	_, _, err := c.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("空 choices 应报错")
	}
}
