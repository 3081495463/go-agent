package tools

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestShellToolRejectsUnknownAction(t *testing.T) {
	s := NewShellTool(t.TempDir())

	_, err := s.Execute(context.Background(), map[string]any{
		"action": "delete_everything",
	})

	if err == nil {
		t.Fatal("expected unsupported action error")
	}
}

func TestShellToolRejectsNilArguments(t *testing.T) {
	s := NewShellTool(t.TempDir())

	_, err := s.Execute(context.Background(), nil)

	if err == nil {
		t.Fatal("expected nil arguments error")
	}
}

func TestServePortAllowed(t *testing.T) {
	cases := map[string]bool{
		"8090":  true,
		"1024":  true,
		"65535": true,
		"80":    false,
		"999":   false,
		"70000": false,
		"abc":   false,
		"":      false,
	}

	for port, want := range cases {
		if got := servePortAllowed(port); got != want {
			t.Fatalf("servePortAllowed(%q) = %v, want %v", port, got, want)
		}
	}
}

func TestShellToolServeStartable(t *testing.T) {
	workDir := t.TempDir()
	if err := os.WriteFile(workDir+"/index.html", []byte("<h1>hello</h1>"), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewShellTool(workDir)

	out, err := s.Execute(context.Background(), map[string]any{
		"action": "serve",
		"port":   "18091",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "127.0.0.1:18091") {
		t.Fatalf("unexpected serve output: %s", out)
	}

	resp, err := http.Get("http://127.0.0.1:18091/index.html")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("状态码: %d", resp.StatusCode)
	}
}

func TestShellToolServeRejectsBadPort(t *testing.T) {
	s := NewShellTool(t.TempDir())

	_, err := s.Execute(context.Background(), map[string]any{
		"action": "serve",
		"port":   "80",
	})
	if err == nil {
		t.Fatal("expected bad port error")
	}
}
