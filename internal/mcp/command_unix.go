//go:build !windows

package mcp

import (
	"context"
	"os/exec"
)

// buildCommandPlatform 非 Windows：直接执行原始命令。
func buildCommandPlatform(ctx context.Context, command string, args []string, _ commandOptions) (*exec.Cmd, error) {
	return exec.CommandContext(ctx, command, args...), nil
}
