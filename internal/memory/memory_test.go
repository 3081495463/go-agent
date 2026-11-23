package memory

import (
	"testing"

	"go-agent/internal/llm"
)

func TestSaveAndLoad(t *testing.T) {
	store := NewStore(t.TempDir())

	conv := Conversation{
		ID: "conv-001",
		Notes: []Note{
			{Key: "style", Content: "偏好简约现代风"},
		},
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
	if len(loaded.Notes) != 1 || loaded.Notes[0].Content != "偏好简约现代风" {
		t.Fatalf("notes 不匹配: %+v", loaded.Notes)
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

func TestUpsertNoteDedup(t *testing.T) {
	conv := Conversation{ID: "x"}
	conv.UpsertNote("style", "第一版")
	conv.UpsertNote("style", "第二版")
	conv.UpsertNote("project", "go-agent")

	if len(conv.Notes) != 2 {
		t.Fatalf("按 key 去重后期望 2 条, 实际 %d", len(conv.Notes))
	}
	if conv.Notes[0].Content != "第二版" {
		t.Fatalf("同 key 应覆盖: %+v", conv.Notes)
	}
}
