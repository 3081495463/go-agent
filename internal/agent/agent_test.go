package agent

import (
	"context"
	"os"
	"testing"

	"go-agent/internal/llm"
	"go-agent/internal/memory"
	"go-agent/internal/tools"
)

func newTestAgent(t *testing.T) *Agent {
	t.Helper()
	registry := tools.NewRegistry()
	fileTool := tools.NewFileTool(t.TempDir())
	if err := registry.Register(fileTool); err != nil {
		t.Fatal(err)
	}
	return NewAgent(llm.NewMockModel("ok"), registry)
}

func TestAgentCallTool(t *testing.T) {
	a := newTestAgent(t)

	_, err := a.CallTool(context.Background(), "file", map[string]any{
		"action":  "write",
		"path":    "hello.txt",
		"content": "hello",
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestAgentUnknownTool(t *testing.T) {
	a := NewAgent(llm.NewMockModel("ok"), tools.NewRegistry())

	_, err := a.CallTool(context.Background(), "unknown", nil)
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
}

func TestChatNotReady(t *testing.T) {
	cases := []*Agent{
		NewAgent(nil, tools.NewRegistry()),
		NewAgent(llm.NewMockModel("ok"), nil),
	}

	for _, a := range cases {
		if _, err := a.Chat(context.Background(), "", "hi"); err != ErrAgentNotReady {
			t.Fatalf("expected ErrAgentNotReady, got %v", err)
		}
	}
}

func TestChatWithMockModel(t *testing.T) {
	a := NewAgent(llm.NewMockModel("hello from mock"), tools.NewRegistry())

	reply, err := a.Chat(context.Background(), "", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if reply != "hello from mock" {
		t.Fatalf("unexpected reply: %s", reply)
	}
}

func TestChatPersistsHistory(t *testing.T) {
	store := memory.NewStore(t.TempDir())
	a := NewAgent(llm.NewMockModel("记住了"), tools.NewRegistry())
	a.Store = store

	if _, err := a.Chat(context.Background(), "sess-1", "我叫小明"); err != nil {
		t.Fatal(err)
	}

	conv, err := store.Load("sess-1")
	if err != nil {
		t.Fatal(err)
	}

	if len(conv.Messages) != 2 {
		t.Fatalf("期望 2 条历史消息, 实际 %d", len(conv.Messages))
	}
	if conv.Messages[0].Role != "user" || conv.Messages[0].Content != "我叫小明" {
		t.Fatalf("首条历史不匹配: %+v", conv.Messages[0])
	}
	if conv.Messages[1].Role != "assistant" || conv.Messages[1].Content != "记住了" {
		t.Fatalf("次条历史不匹配: %+v", conv.Messages[1])
	}
}

func TestChatWithoutSessionNoPersist(t *testing.T) {
	dir := t.TempDir()
	store := memory.NewStore(dir)
	a := NewAgent(llm.NewMockModel("ok"), tools.NewRegistry())
	a.Store = store

	if _, err := a.Chat(context.Background(), "", "hi"); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("无会话 ID 不应落盘, 目录: %v", entries)
	}
}
