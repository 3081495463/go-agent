package tools

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func writeTempFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

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

func TestShellToolRejectsCanceledContext(t *testing.T) {
	s := NewShellTool(t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.Execute(ctx, map[string]any{
		"action": "go_version",
	})

	if err == nil {
		t.Fatal("expected canceled context error")
	}
}

func TestShellToolGoVersion(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	s := NewShellTool(t.TempDir())

	output, err := s.Execute(context.Background(), map[string]any{
		"action": "go_version",
	})

	if err != nil {
		t.Fatal(err)
	}

	if output == "" {
		t.Fatal("expected command output")
	}

	if !strings.Contains(output, "go version") {
		t.Fatalf("unexpected go version output: %s", output)
	}
}

func TestShellToolRejectsMissingWorkDir(t *testing.T) {
	s := NewShellTool(t.TempDir() + "/not-exist")

	_, err := s.Execute(context.Background(), map[string]any{
		"action": "go_version",
	})

	if err == nil {
		t.Fatal("expected missing work directory error")
	}
}

func TestShellToolRejectsFileAsWorkDir(t *testing.T) {
	file := t.TempDir() + "/a.txt"
	if err := writeTempFile(file, "x"); err != nil {
		t.Fatal(err)
	}

	s := NewShellTool(file)

	_, err := s.Execute(context.Background(), map[string]any{
		"action": "go_version",
	})

	if err == nil {
		t.Fatal("expected not-a-directory error")
	}
}

func TestShellToolCommandFailure(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// 临时目录不是 git 仓库，git status 应返回非零退出码。
	s := NewShellTool(t.TempDir())

	_, err := s.Execute(context.Background(), map[string]any{
		"action": "git_status",
	})

	if err == nil {
		t.Fatal("expected command failure error")
	}
}

func TestShellToolTimeout(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}

	// 超时设得极短，命令来不及完成，应报超时/取消。
	s := NewShellTool(t.TempDir())
	s.Timeout = 1 * time.Millisecond

	_, err := s.Execute(context.Background(), map[string]any{
		"action": "go_version",
	})

	if err == nil {
		t.Fatal("expected timeout error")
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
