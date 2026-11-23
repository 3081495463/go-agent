package mcp

import (
	"context"
	"errors"
	"testing"
)

func TestAdapterNameDescription(t *testing.T) {
	meta := DiscoveredTool{
		Name:        "browser_navigate",
		Description: "打开指定 URL",
		Server:      "playwright",
	}

	tool := NewToolAdapter(nil, meta)

	if tool.Name() != "browser_navigate" {
		t.Fatalf("Name() = %q, want browser_navigate", tool.Name())
	}

	desc := tool.Description()
	if desc == "" || desc == "打开指定 URL" {
		t.Fatalf("Description() 应附带来源 server: %q", desc)
	}
}

func TestAdapterExecuteNilClient(t *testing.T) {
	tool := NewToolAdapter(nil, DiscoveredTool{Name: "x"})

	_, err := tool.Execute(context.Background(), map[string]any{})
	if !errors.Is(err, errToolNotConfigured) {
		t.Fatalf("期望未绑定客户端错误, got %v", err)
	}
}
