package tools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxFileSize = 1 << 20 // 1 MB

// FileTool 提供工作区文件读写能力
type FileTool struct {
	WorkDir string
}

// NewFileTool 创建文件工具
func NewFileTool(workDir string) *FileTool {
	return &FileTool{
		WorkDir: workDir,
	}
}

// Name 工具名称
func (f *FileTool) Name() string {
	return "file"
}

// Description 工具描述
func (f *FileTool) Description() string {
	return "在 Agent 工作区中读取和写入文件"
}

// Execute 执行文件操作
func (f *FileTool) Execute(
	ctx context.Context,
	args map[string]any,
) (string, error) {

	if f == nil {
		return "", fmt.Errorf("file tool is nil")
	}

	if ctx == nil {
		return "", fmt.Errorf("context is nil")
	}

	if err := ctx.Err(); err != nil {
		return "", err
	}

	if args == nil {
		return "", fmt.Errorf("arguments cannot be nil")
	}

	action, err := requireString(args, "action")
	if err != nil {
		return "", err
	}

	path, err := requireString(args, "path")
	if err != nil {
		return "", err
	}

	fullPath, err := f.resolvePath(path)
	if err != nil {
		return "", err
	}

	switch action {
	case "read":
		return f.readFile(fullPath)

	case "write":
		content, err := requireString(args, "content")
		if err != nil {
			return "", err
		}

		overwrite, _ := args["overwrite"].(bool)

		return f.writeFile(fullPath, content, overwrite)

	default:
		return "", fmt.Errorf("unsupported file action: %s", action)
	}
}

// requireString 获取必填字符串参数
func requireString(args map[string]any, key string) (string, error) {
	if args == nil {
		return "", fmt.Errorf("arguments cannot be nil")
	}

	value, ok := args[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("parameter %q must be a non-empty string", key)
	}

	return value, nil
}

// resolvePath 检查并解析工作区路径
func (f *FileTool) resolvePath(path string) (string, error) {
	if strings.Contains(path, "\\") {
		return "", fmt.Errorf("backslash paths are not allowed")
	}

	if filepath.IsAbs(path) || filepath.VolumeName(path) != "" || strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("absolute paths are not allowed")
	}

	cleanPath := filepath.Clean(filepath.FromSlash(path))

	if cleanPath == "." || cleanPath == ".." ||
		strings.HasPrefix(cleanPath, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path traversal is not allowed")
	}

	root, err := filepath.Abs(f.WorkDir)
	if err != nil {
		return "", fmt.Errorf("resolve work directory: %w", err)
	}

	if err := os.MkdirAll(root, 0755); err != nil {
		return "", fmt.Errorf("create work directory: %w", err)
	}

	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve work directory symlinks: %w", err)
	}

	target := filepath.Join(root, cleanPath)

	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", fmt.Errorf("check path: %w", err)
	}

	if rel == ".." ||
		strings.HasPrefix(rel, ".."+string(os.PathSeparator)) ||
		filepath.IsAbs(rel) {
		return "", fmt.Errorf("path is outside workspace")
	}

	// 检查现有路径的每一级，拒绝符号链接。不存在的级别跳过，不中断后续检查。
	current := root
	parts := strings.Split(cleanPath, string(os.PathSeparator))

	for _, part := range parts {
		current = filepath.Join(current, part)

		info, err := os.Lstat(current)

		if os.IsNotExist(err) {
			continue
		}

		if err != nil {
			return "", fmt.Errorf("inspect path: %w", err)
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symbolic links are not allowed")
		}
	}

	return target, nil
}

// readFile 读取文件
func (f *FileTool) readFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("stat file: %w", err)
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("only regular files can be read")
	}

	if info.Size() > maxFileSize {
		return "", fmt.Errorf("file exceeds 1 MB limit")
	}

	data, err := io.ReadAll(io.LimitReader(file, maxFileSize+1))
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	if len(data) > maxFileSize {
		return "", fmt.Errorf("file exceeds 1 MB limit")
	}

	return string(data), nil
}

// writeFile 写入文件
func (f *FileTool) writeFile(
	path string,
	content string,
	overwrite bool,
) (string, error) {

	if len(content) > maxFileSize {
		return "", fmt.Errorf("content exceeds 1 MB limit")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("create parent directory: %w", err)
	}

	flags := os.O_WRONLY | os.O_CREATE

	if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}

	file, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("file already exists, set overwrite=true to replace: %w", err)
		}
		return "", fmt.Errorf("write file: %w", err)
	}
	defer file.Close()

	// 打开后复核：拒绝符号链接和非常规文件，挡住校验与打开之间的竞态。
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("stat file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("symbolic links are not allowed")
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("only regular files can be written")
	}

	if _, err := file.WriteString(content); err != nil {
		return "", fmt.Errorf("write content: %w", err)
	}

	return fmt.Sprintf("file written successfully: %s", path), nil
}
