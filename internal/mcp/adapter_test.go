package mcp

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
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

// Windows 上 npx 应解析到 .cmd 批处理绝对路径，而不是无扩展名脚本。
func TestResolveBatchPrefersCmd(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("仅 Windows 验证批处理解析")
	}

	p, err := resolveBatchCommand("npx")
	if err != nil {
		t.Skip("PATH 中没有 npx.cmd")
	}
	if ext := filepath.Ext(p); ext != ".cmd" && ext != ".bat" {
		t.Fatalf("应解析到 .cmd/.bat, 实际 %q", p)
	}
	if _, err := exec.LookPath(p); err != nil {
		t.Fatalf("解析出的路径不可用: %v", err)
	}
}

// 显式配置的 node / npx-cli 路径应被原样采用，且不存在时给出可读错误。
func TestResolveNodeExeExplicit(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("仅 Windows 验证路径覆盖")
	}

	// 不存在的显式路径应报错，而不是退回 PATH。
	if _, err := resolveNodeExe(`C:\no\such\node.exe`); err == nil {
		t.Fatal("显式路径无效时应返回错误")
	}

	// 显式空路径走自动探测；若环境有 node 则应返回绝对路径。
	if p, err := resolveNodeExe(""); err == nil && p == "" {
		t.Fatal("自动探测不应返回空路径")
	}
}

// 显式 npx-cli 路径无效时应报错。
func TestResolveNPXCLIExplicitMissing(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("仅 Windows 验证路径覆盖")
	}
	if _, err := resolveNPXCLI(`C:\node.exe`, `C:\no\such\npx-cli.js`); err == nil {
		t.Fatal("显式 npx-cli 路径无效时应返回错误")
	}
}
