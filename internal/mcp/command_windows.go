//go:build windows

package mcp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// buildCommandPlatform 构造 Windows 下的 MCP 子进程命令。
//
// 启动优先级：
//  1. npx 命令：优先 node.exe + npx-cli.js 直接执行（绕开 .cmd/无扩展名脚本包装器）；
//  2. 其它命令：解析 .cmd/.bat 绝对路径后包一层 cmd /d /c；
//  3. 两条分支都失败时返回明确错误，不再静默退回裸命令（裸 npx 会被 ErrDot 拒绝）。
func buildCommandPlatform(ctx context.Context, command string, args []string, opts commandOptions) (*exec.Cmd, error) {
	if isNPXCommand(command) {
		if nodeCmd, err := buildNodeNPXCommand(ctx, args, opts); err == nil {
			return nodeCmd, nil
		} else {
			// 记录首选分支失败原因，供最后的错误汇总，避免静默退回裸 npx。
			log.Printf("MCP 启动: node+npx-cli 分支不可用: %v", err)
		}
	}

	if batch, err := resolveBatchCommand(command); err == nil {
		cmdArgs := append([]string{"/d", "/c", batch}, args...)
		return exec.CommandContext(ctx, "cmd.exe", cmdArgs...), nil
	} else {
		log.Printf("MCP 启动: cmd+npx.cmd 分支不可用: %v", err)
	}

	// 两条兼容分支都失败：不再静默退回裸命令（Windows 上会被 ErrDot 拒绝），
	// 直接返回明确错误，附带原始命令。
	return nil, fmt.Errorf("无法在 Windows 下解析命令 %q：node 运行时与 .cmd 批处理均不可用，请检查 PATH 或配置 MCP_NODE_PATH / MCP_NPX_CLI_PATH", command)
}

// isNPXCommand 判断原始命令是否为 npx。
func isNPXCommand(command string) bool {
	base := filepath.Base(command)
	return base == "npx" || base == "npx.cmd" || base == "npx.bat"
}

// resolveBatchCommand 在 PATH 中查找命令对应的 .cmd/.bat 绝对路径。
// ErrDot（相对路径）不绕过，直接返回可读错误。
func resolveBatchCommand(command string) (string, error) {
	var candidates []string

	ext := strings.ToLower(filepath.Ext(command))
	if ext == ".cmd" || ext == ".bat" {
		candidates = append(candidates, command)
	} else {
		candidates = append(candidates, command+".cmd", command+".bat")
	}

	for _, candidate := range candidates {
		resolved, err := exec.LookPath(candidate)
		if err != nil {
			if errors.Is(err, exec.ErrDot) {
				return "", fmt.Errorf("检测到相对路径命令 %q，请检查 PATH 并使用绝对路径", candidate)
			}
			continue
		}
		absolute, err := filepath.Abs(resolved)
		if err != nil {
			return "", fmt.Errorf("解析命令绝对路径失败: %w", err)
		}
		return absolute, nil
	}

	return "", fmt.Errorf("找不到命令 %q 的 .cmd/.bat，请确认 Node.js/npm 已安装", command)
}

// buildNodeNPXCommand：定位 node.exe 与 npx-cli.js，直接用 node 执行 npx。
// 显式配置（opts）优先，其次自动探测。
func buildNodeNPXCommand(ctx context.Context, args []string, opts commandOptions) (*exec.Cmd, error) {
	nodePath, err := resolveNodeExe(opts.nodePath)
	if err != nil {
		return nil, err
	}

	npxCLI, err := resolveNPXCLI(nodePath, opts.npxCLIPath)
	if err != nil {
		return nil, err
	}

	return exec.CommandContext(ctx, nodePath, append([]string{npxCLI}, args...)...), nil
}

// resolveNodeExe 定位 node.exe：先用显式配置路径，再在 PATH 中查找（只认 .exe）。
// 遇到 ErrDot（相对路径）继续尝试下一个候选，而不是直接失败。
func resolveNodeExe(explicit string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil || !fileExists(abs) {
			return "", fmt.Errorf("配置的 MCP_NODE_PATH 无效: %q", explicit)
		}
		return abs, nil
	}

	for _, candidate := range []string{"node.exe", "node"} {
		resolved, err := exec.LookPath(candidate)
		if err != nil {
			if errors.Is(err, exec.ErrDot) {
				continue
			}
			continue
		}
		absolute, err := filepath.Abs(resolved)
		if err != nil {
			continue
		}
		if fileExists(absolute) {
			return absolute, nil
		}
	}

	return "", fmt.Errorf(
		"未找到可用的 Node.js 运行时：请检查 Node.js 是否已安装、node.exe 是否在 PATH，或配置 MCP_NODE_PATH 为有效绝对路径")
}

// resolveNPXCLI 定位 npx-cli.js：先用显式配置，再从 node.exe 目录及其父目录探测。
func resolveNPXCLI(nodePath, explicit string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil || !fileExists(abs) {
			return "", fmt.Errorf("配置的 MCP_NPX_CLI_PATH 无效: %q", explicit)
		}
		return abs, nil
	}

	dir := filepath.Dir(nodePath)
	dirs := []string{dir}
	if parent := filepath.Dir(dir); parent != dir {
		dirs = append(dirs, parent)
	}

	for _, base := range dirs {
		candidate := filepath.Join(base, "node_modules", "npm", "bin", "npx-cli.js")
		if abs, err := filepath.Abs(candidate); err == nil && fileExists(abs) {
			return abs, nil
		}
	}

	return "", fmt.Errorf(
		"未找到 npx-cli.js（node 目录: %s）：请确认 npm 已随 Node 安装，或配置 MCP_NPX_CLI_PATH 为有效绝对路径", dir)
}

// fileExists 判断文件是否存在且为普通文件。
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
