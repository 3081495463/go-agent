package llm

import "context"

// Message 一条对话消息。
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall 模型发起的一次工具调用。
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall 调用的函数名与参数（Arguments 为 JSON 字符串）。
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDefinition 注册给模型的工具定义。
type ToolDefinition struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function 工具的元信息与参数 Schema。
type Function struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// Response 模型原始响应结构。
type Response struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`

	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// Model 统一的大模型对话接口，更换具体模型时无需改动 Agent 主体。
type Model interface {
	Chat(
		ctx context.Context,
		messages []Message,
		tools []ToolDefinition,
	) (Message, int, error)
}
