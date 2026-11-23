package mcp

import (
	"context"
	"os/exec"
)

// commandOptions 控制子进程命令的显式覆盖项，优先级高于自动探测。
type commandOptions struct {
	nodePath     string // 可选：node.exe 绝对路径
	npxCLIPath   string // 可选：npx-cli.js 绝对路径
}

// buildCommand 构造 MCP 子进程命令。
// 非 Windows 直接执行原始命令；Windows 的兼容逻辑在 command_windows.go 实现。
func buildCommand(ctx context.Context, command string, args []string, opts commandOptions) (*exec.Cmd, error) {
	return buildCommandPlatform(ctx, command, args, opts)
}
