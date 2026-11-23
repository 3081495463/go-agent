// Package mcp 通过官方 MCP Go SDK 连接外部 MCP Server（stdio 子进程），
// 负责握手、工具发现、工具调用，并把服务器返回的内容转成纯文本。
//
// 安全边界：服务器返回的内容只作为工具结果文本回传给模型，
// 不把其中的任意指令当作可信命令执行。
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Client 连接一个 MCP Server 子进程的客户端。
type Client struct {
	name    string
	command string
	args    []string
	opts    commandOptions

	client  *mcp.Client
	session *mcp.ClientSession
}

// NewClient 创建 MCP 客户端。name 用于日志与标识，command 为服务器可执行文件，
// args 为启动参数（例如 npx @playwright/mcp@latest --headless）。
func NewClient(name, command string, args []string) *Client {
	return &Client{
		name:    name,
		command: command,
		args:    append([]string{}, args...),
	}
}

// WithCommandOptions 注入启动命令的显式覆盖项（node / npx-cli 绝对路径）。
func (c *Client) WithCommandOptions(opts commandOptions) *Client {
	if c != nil {
		c.opts = opts
	}
	return c
}

// Initialize 启动子进程并完成 MCP 握手。
func (c *Client) Initialize(ctx context.Context) error {
	if c == nil || c.command == "" {
		return fmt.Errorf("mcp client is not configured")
	}

	c.client = mcp.NewClient(
		&mcp.Implementation{
			Name:    "go-agent",
			Version: "0.1.0",
		},
		nil,
	)

	command, err := buildCommand(ctx, c.command, c.args, c.opts)
	if err != nil {
		return fmt.Errorf("启动 MCP Server %q 失败: %w", c.name, err)
	}

	transport := &mcp.CommandTransport{
		Command:           command,
		TerminateDuration: 5 * time.Second,
	}

	session, err := c.client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("连接 MCP Server %q 失败: %w", c.name, err)
	}

	c.session = session
	return nil
}

// ListTools 拉取服务器提供的工具清单。
func (c *Client) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	if c == nil || c.session == nil {
		return nil, fmt.Errorf("mcp client %q 尚未初始化", c.name)
	}

	result, err := c.session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("列出 MCP 工具失败: %w", err)
	}

	return result.Tools, nil
}

// CallTool 调用服务器工具，把返回内容汇总为纯文本。
func (c *Client) CallTool(
	ctx context.Context,
	name string,
	args map[string]any,
) (string, error) {

	if c == nil || c.session == nil {
		return "", fmt.Errorf("mcp client %q 尚未初始化", c.name)
	}

	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return "", fmt.Errorf("调用 MCP 工具 %q 失败: %w", name, err)
	}

	if result.GetError() != nil {
		return "", result.GetError()
	}

	text := extractText(result)
	if result.IsError {
		return text, fmt.Errorf("MCP 工具 %q 返回错误: %s", name, text)
	}

	return text, nil
}

// Close 关闭会话并终止子进程。
func (c *Client) Close() error {
	if c == nil || c.session == nil {
		return nil
	}
	return c.session.Close()
}

// extractText 把工具结果里的文本、图片、结构化内容汇总成一段可读文本。
func extractText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}

	parts := make([]string, 0, len(result.Content))

	for _, content := range result.Content {
		switch v := content.(type) {
		case *mcp.TextContent:
			if v.Text != "" {
				parts = append(parts, v.Text)
			}
		case *mcp.ImageContent:
			parts = append(parts, fmt.Sprintf("[图片: %s, %d 字节]", v.MIMEType, len(v.Data)))
		default:
			if b, err := json.Marshal(v); err == nil {
				parts = append(parts, string(b))
			}
		}
	}

	if len(parts) == 0 && result.StructuredContent != nil {
		if b, err := json.Marshal(result.StructuredContent); err == nil {
			parts = append(parts, string(b))
		}
	}

	out := ""
	for _, p := range parts {
		if out == "" {
			out = p
		} else {
			out += "\n" + p
		}
	}

	return out
}
