package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestFileTool(t *testing.T) *FileTool {
	t.Helper()
	return NewFileTool(t.TempDir())
}

func TestFileToolWriteAndRead(t *testing.T) {
	f := newTestFileTool(t)
	ctx := context.Background()

	_, err := f.Execute(ctx, map[string]any{
		"action":  "write",
		"path":    "hello.txt",
		"content": "hello agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := f.Execute(ctx, map[string]any{
		"action": "read",
		"path":   "hello.txt",
	})
	if err != nil {
		t.Fatal(err)
	}

	if got != "hello agent" {
		t.Fatalf("unexpected content: %s", got)
	}
}

func TestFileToolRejectsTraversal(t *testing.T) {
	f := newTestFileTool(t)

	_, err := f.Execute(context.Background(), map[string]any{
		"action":  "write",
		"path":    "../outside.txt",
		"content": "bad",
	})

	if err == nil {
		t.Fatal("expected traversal error")
	}
}

func TestFileToolRejectsAbsolutePath(t *testing.T) {
	f := newTestFileTool(t)

	_, err := f.Execute(context.Background(), map[string]any{
		"action": "read",
		"path":   filepath.Join(t.TempDir(), "outside.txt"),
	})

	if err == nil {
		t.Fatal("expected absolute path error")
	}
}

func TestFileToolRejectsBackslash(t *testing.T) {
	f := newTestFileTool(t)

	_, err := f.Execute(context.Background(), map[string]any{
		"action": "read",
		"path":   `folder\secret.txt`,
	})

	if err == nil {
		t.Fatal("expected backslash path error")
	}
}

func TestFileToolRejectsOverwrite(t *testing.T) {
	f := newTestFileTool(t)
	ctx := context.Background()

	args := map[string]any{
		"action":  "write",
		"path":    "hello.txt",
		"content": "first",
	}

	if _, err := f.Execute(ctx, args); err != nil {
		t.Fatal(err)
	}

	args["content"] = "second"

	if _, err := f.Execute(ctx, args); err == nil {
		t.Fatal("expected overwrite rejection")
	}

	args["overwrite"] = true

	if _, err := f.Execute(ctx, args); err != nil {
		t.Fatal(err)
	}

	got, err := f.Execute(context.Background(), map[string]any{
		"action": "read",
		"path":   "hello.txt",
	})
	if err != nil {
		t.Fatal(err)
	}

	if got != "second" {
		t.Fatalf("expected second, got %s", got)
	}
}

func TestFileToolRejectsLargeContent(t *testing.T) {
	f := newTestFileTool(t)

	_, err := f.Execute(context.Background(), map[string]any{
		"action":  "write",
		"path":    "large.txt",
		"content": strings.Repeat("a", maxFileSize+1),
	})

	if err == nil {
		t.Fatal("expected file size error")
	}
}

func TestFileToolRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	f := NewFileTool(root)

	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable on this system: %v", err)
	}

	_, err := f.Execute(context.Background(), map[string]any{
		"action": "read",
		"path":   "link/secret.txt",
	})

	if err == nil {
		t.Fatal("expected symlink rejection")
	}
}
