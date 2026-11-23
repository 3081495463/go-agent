package memory

import (
	"testing"

	"go-agent/internal/llm"
)

func TestSaveAndLoad(t *testing.T) {
	store := NewStore(t.TempDir())

	conv := Conversation{
		ID:      "conv-001",
		Summary: "测试会话",
		Messages: []llm.Message{
			{Role: "user", Content: "你好"},
			{Role: "assistant", Content: "你好，有什么可以帮你"},
		},
	}

	if err := store.Save(conv); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load("conv-001")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Summary != conv.Summary {
		t.Fatalf("summary 不匹配: %s", loaded.Summary)
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("消息数不匹配: %d", len(loaded.Messages))
	}
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	store := NewStore(t.TempDir())
	conv, err := store.Load("not-exist")
	if err != nil {
		t.Fatal(err)
	}
	if conv.ID != "not-exist" {
		t.Fatalf("ID 不匹配: %s", conv.ID)
	}
}

func TestRejectTraversalID(t *testing.T) {
	store := NewStore(t.TempDir())

	for _, id := range []string{"../etc", "a/b", "..", "foo\\bar"} {
		if _, err := store.Load(id); err == nil {
			t.Fatalf("非法 ID %q 应被拒绝", id)
		}
	}
}
