package tools

import (
	"context"
	"testing"
)

type testTool struct {
	name string
}

func (t *testTool) Name() string {
	return t.name
}

func (t *testTool) Description() string {
	return "test tool"
}

func (t *testTool) Execute(
	ctx context.Context,
	args map[string]any,
) (string, error) {
	return "ok", nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	tool := &testTool{name: "test"}

	if err := r.Register(tool); err != nil {
		t.Fatal(err)
	}

	got, ok := r.Get("test")
	if !ok || got.Name() != "test" {
		t.Fatal("tool was not registered correctly")
	}
}

func TestRegistryDuplicate(t *testing.T) {
	r := NewRegistry()

	if err := r.Register(&testTool{name: "test"}); err != nil {
		t.Fatal(err)
	}

	err := r.Register(&testTool{name: "test"})
	if err == nil {
		t.Fatal("expected duplicate registration error")
	}
}

func TestRegistryNilTool(t *testing.T) {
	r := NewRegistry()

	var tool *testTool

	if err := r.Register(tool); err == nil {
		t.Fatal("expected nil tool error")
	}
}

func TestRegistryListSorted(t *testing.T) {
	r := NewRegistry()

	for _, name := range []string{"z", "a", "m"} {
		if err := r.Register(&testTool{name: name}); err != nil {
			t.Fatal(err)
		}
	}

	got := r.List()
	expected := []string{"a", "m", "z"}

	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	}
}
