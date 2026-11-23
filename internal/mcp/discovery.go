package mcp

import (
	"context"
	"fmt"
)

// DiscoveredTool 是从 MCP Server 发现的一个工具的元信息快照。
type DiscoveredTool struct {
	// Name 工具名（在 Agent 注册表中作为唯一标识）。
	Name string
	// Description 工具的人类可读说明。
	Description string
	// InputSchema 工具参数的 JSON Schema（原样透传给模型）。
	InputSchema any
	// Server 提供该工具的 MCP Server 名称。
	Server string
}

// DiscoverTools 连接客户端并拉取工具清单，转成内部快照。
func DiscoverTools(ctx context.Context, client *Client) ([]DiscoveredTool, error) {
	if client == nil {
		return nil, fmt.Errorf("mcp client is nil")
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	discovered := make([]DiscoveredTool, 0, len(tools))
	for _, tool := range tools {
		if tool == nil || tool.Name == "" {
			continue
		}

		discovered = append(discovered, DiscoveredTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
			Server:      client.name,
		})
	}

	return discovered, nil
}

// ToolSchema 便于在 adapter 中访问发现结果的参数 Schema。
func (d DiscoveredTool) ToolSchema() any {
	return d.InputSchema
}
