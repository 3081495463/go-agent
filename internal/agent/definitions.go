package agent

import (
	"go-agent/internal/llm"
	"go-agent/internal/tools"
)

func BuildToolDefinitions(
	registry *tools.Registry,
) []llm.ToolDefinition {

	result := []llm.ToolDefinition{}

	for _, name := range registry.List() {

		tool, ok := registry.Get(name)
		if !ok {
			continue
		}

		var properties map[string]any
		var parameters map[string]any

		switch name {
		case "file":
			properties = map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": []string{"read", "write"},
				},
				"path": map[string]any{
					"type": "string",
				},
				"content": map[string]any{
					"type": "string",
				},
				"overwrite": map[string]any{
					"type": "boolean",
				},
			}

		case "shell":
			properties = map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": []string{"go_version", "go_fmt", "git_status", "serve", "open"},
				},
				"port": map[string]any{
					"type":        "string",
					"description": "serve 动作的监听端口，默认 8090",
				},
				"target": map[string]any{
					"type":        "string",
					"description": "open 动作要打开的页面 URL 或工作区内相对文件路径",
				},
			}

		default:
			properties = map[string]any{}
		}

		// MCP 适配工具自带参数 Schema，优先透传其原生 JSON Schema。
		if schemed, ok := tool.(interface{ Schema() any }); ok {
			if schema := schemed.Schema(); schema != nil {
				parameters = toParameterMap(schema)
			}
		}

		if parameters == nil {
			parameters = map[string]any{
				"type":                 "object",
				"properties":           properties,
				"additionalProperties": false,
			}
		}

		result = append(result, llm.ToolDefinition{
			Type: "function",
			Function: llm.Function{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  parameters,
			},
		})
	}

	return result
}

// toParameterMap 把 MCP 传来的 inputSchema 归一化成参数对象；
// 为空时回退到最小对象 Schema。
func toParameterMap(schema any) map[string]any {
	switch s := schema.(type) {
	case map[string]any:
		out := make(map[string]any, len(s)+1)
		for k, v := range s {
			out[k] = v
		}
		if _, hasType := out["type"]; !hasType {
			out["type"] = "object"
		}
		return out
	default:
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}
}
