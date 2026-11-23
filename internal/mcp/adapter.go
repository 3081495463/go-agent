package mcp

import (
	"context"
	"errors"

	"go-agent/internal/tools"
)

// errToolNotConfigured 适配工具未绑定客户端时返回。
var errToolNotConfigured = errors.New("mcp tool 未绑定客户端")

// MCPTool 把一个 MCP Server 提供的工具适配成 Agent 现有的 tools.Tool 接口。
// Agent 与工具注册表无需区分普通工具和 MCP 工具。
type MCPTool struct {
	client *Client
	meta   DiscoveredTool
}

// 编译期校验：*MCPTool 实现 tools.Tool。
var _ tools.Tool = (*MCPTool)(nil)

// NewToolAdapter 用发现到的工具元信息和对应客户端构造一个适配工具。
func NewToolAdapter(client *Client, meta DiscoveredTool) *MCPTool {
	return &MCPTool{client: client, meta: meta}
}

// Name 返回 MCP 工具名。
func (t *MCPTool) Name() string {
	if t == nil {
		return ""
	}
	return t.meta.Name
}

// Description 返回工具说明，附带来源 MCP Server 便于模型理解上下文。
func (t *MCPTool) Description() string {
	if t == nil {
		return ""
	}
	desc := t.meta.Description
	if t.meta.Server != "" {
		desc += "（来自 MCP Server: " + t.meta.Server + "）"
	}
	return desc
}

// Execute 转发参数到 MCP Server 并回传纯文本结果。
func (t *MCPTool) Execute(
	ctx context.Context,
	args map[string]any,
) (string, error) {
	if t == nil || t.client == nil {
		return "", errToolNotConfigured
	}
	return t.client.CallTool(ctx, t.meta.Name, args)
}

// Schema 返回该工具的参数 JSON Schema，供生成模型工具定义时使用。
func (t *MCPTool) Schema() any {
	if t == nil {
		return nil
	}
	return t.meta.InputSchema
}
