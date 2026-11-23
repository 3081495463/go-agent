package tools

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const maxShellOutput = 1 << 20 // 1 MB

// ShellTool 提供受限制的命令执行能力，仅开放白名单内的固定动作。
type ShellTool struct {
	WorkDir string
	Timeout time.Duration

	// serveOnce 保证静态服务只启动一次，避免重复监听同一端口。
	serveOnce sync.Once
	serveAddr string
}

// NewShellTool 创建 Shell 工具，默认超时 15 秒。
func NewShellTool(workDir string) *ShellTool {
	return &ShellTool{
		WorkDir: workDir,
		Timeout: 15 * time.Second,
	}
}

// Name 工具名称
func (s *ShellTool) Name() string {
	return "shell"
}

// Description 工具描述
func (s *ShellTool) Description() string {
	return "执行预先允许的 Go、Git 项目操作，可在工作区启动静态网页服务，并用默认浏览器打开页面"
}

// limitedBuffer 限制命令输出大小，超出部分截断并打标记。
type limitedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := b.limit - b.buffer.Len()

	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
			b.truncated = true
		}
		_, _ = b.buffer.Write(p)
	} else if len(p) > 0 {
		b.truncated = true
	}

	return n, nil
}

func (b *limitedBuffer) String() string {
	result := b.buffer.String()
	if b.truncated {
		result += "\n[output truncated: 1 MB limit]"
	}
	return result
}

// Execute 执行预设的白名单动作。
func (s *ShellTool) Execute(
	ctx context.Context,
	args map[string]any,
) (string, error) {

	if s == nil {
		return "", fmt.Errorf("shell tool is nil")
	}

	if ctx == nil {
		return "", fmt.Errorf("context is nil")
	}

	if err := ctx.Err(); err != nil {
		return "", err
	}

	action, err := requireString(args, "action")
	if err != nil {
		return "", err
	}

	var program string
	var commandArgs []string

	// 只允许固定操作，不接受任意命令或外部拼接参数。
	switch action {
	case "go_version":
		program = "go"
		commandArgs = []string{"version"}

	case "go_fmt":
		program = "go"
		commandArgs = []string{"fmt", "./..."}

	case "git_status":
		program = "git"
		commandArgs = []string{"status", "--short"}

	case "serve":
		return s.startStaticServer(args)

	case "open":
		return s.openInBrowser(args)

	default:
		return "", fmt.Errorf("unsupported shell action: %s", action)
	}

	workDir, err := filepath.Abs(s.WorkDir)
	if err != nil {
		return "", fmt.Errorf("resolve work directory: %w", err)
	}

	info, err := os.Stat(workDir)
	if err != nil {
		return "", fmt.Errorf("inspect work directory: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("work directory is not a directory")
	}

	binary, err := exec.LookPath(program)
	if err != nil {
		return "", fmt.Errorf("program %q not found: %w", program, err)
	}

	timeout := s.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, binary, commandArgs...)
	cmd.Dir = workDir

	output := &limitedBuffer{
		limit: maxShellOutput,
	}

	cmd.Stdout = output
	cmd.Stderr = output

	err = cmd.Run()

	if runCtx.Err() != nil {
		return output.String(), fmt.Errorf(
			"command canceled or timed out: %w",
			runCtx.Err(),
		)
	}

	if err != nil {
		return output.String(), fmt.Errorf(
			"command failed: %w; output: %s",
			err,
			strings.TrimSpace(output.String()),
		)
	}

	return strings.TrimSpace(output.String()), nil
}

// startStaticServer 在工作区启动静态文件服务（标准库 FileServer）。
// 服务在后台 goroutine 常驻；端口默认 8090，可通过 port 参数指定 1024-65535。
func (s *ShellTool) startStaticServer(args map[string]any) (string, error) {
	root, err := filepath.Abs(s.WorkDir)
	if err != nil {
		return "", fmt.Errorf("resolve work directory: %w", err)
	}

	port := "8090"
	if raw, ok := args["port"].(string); ok && raw != "" {
		if !servePortAllowed(raw) {
			return "", fmt.Errorf("port 仅允许 1024-65535 之间的数字")
		}
		port = raw
	}

	addr := "127.0.0.1:" + port

	var startErr error
	s.serveOnce.Do(func() {
		handler := http.FileServer(http.Dir(root))
		server := &http.Server{Addr: addr, Handler: handler}

		go func() {
			_ = server.ListenAndServe()
		}()

		// 等待监听就绪或失败，最多 3 秒。
		probeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		startErr = waitListen(probeCtx, addr)
		if startErr == nil {
			s.serveAddr = addr
		}
	})

	if startErr != nil {
		return "", fmt.Errorf("静态服务启动失败: %w", startErr)
	}

	return fmt.Sprintf("静态服务已启动: http://%s/ (托管目录 %s)", addr, root), nil
}

// openInBrowser 用系统默认浏览器打开一个页面。
// target 可以是 http(s) URL，或工作区内的相对文件路径（自动走本地静态服务地址）。
func (s *ShellTool) openInBrowser(args map[string]any) (string, error) {
	target, err := requireString(args, "target")
	if err != nil {
		return "", err
	}

	url := target
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		// 工作区内相对路径：交给本地静态服务托管；若尚未启动则先在 8090 起一个。
		if _, err := s.startStaticServer(map[string]any{}); err != nil {
			return "", err
		}
		clean := filepath.Clean(filepath.FromSlash(target))
		if clean == "." || strings.HasPrefix(clean, "..") {
			return "", fmt.Errorf("path 必须是工作区内的相对路径")
		}
		root, err := filepath.Abs(s.WorkDir)
		if err != nil {
			return "", fmt.Errorf("resolve work directory: %w", err)
		}
		abs := filepath.Join(root, clean)
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("文件不存在: %s", target)
		}
		host := "127.0.0.1:8090"
		if s.serveAddr != "" {
			host = s.serveAddr
		}
		url = "http://" + host + "/" + filepath.ToSlash(clean)
	}

	// Windows: exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	// 会调用系统默认浏览器；其它平台用 xdg-open / open。
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("打开浏览器失败: %w", err)
	}
	_ = cmd.Process.Release()

	return fmt.Sprintf("已在默认浏览器打开: %s", url), nil
}

// servePortAllowed 校验端口为纯数字且在允许区间。
func servePortAllowed(port string) bool {
	if port == "" || len(port) > 5 {
		return false
	}
	n := 0
	for _, ch := range port {
		if ch < '0' || ch > '9' {
			return false
		}
		n = n*10 + int(ch-'0')
	}
	return n >= 1024 && n <= 65535
}

// waitListen 轮询探测端口是否可连接。
func waitListen(ctx context.Context, addr string) error {
	client := &http.Client{Timeout: time.Second}

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/", nil)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
}
