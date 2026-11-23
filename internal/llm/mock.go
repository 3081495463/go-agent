package llm

import "context"

// MockModel 用于测试与离线联调的假模型，固定返回预设内容。
type MockModel struct {
	Reply string
	Calls int
}

// Chat 实现 Model 接口。
func (m *MockModel) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
) (Message, int, error) {
	m.Calls++
	return Message{Role: "assistant", Content: m.Reply}, 0, nil
}

// NewMockModel 创建固定回复的假模型。
func NewMockModel(reply string) *MockModel {
	return &MockModel{Reply: reply}
}
