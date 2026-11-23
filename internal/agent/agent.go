package agent

import (
	"context"
	"fmt"

	"go-agent/internal/llm"
	"go-agent/internal/memory"
	"go-agent/internal/tools"
)

// Agent 是智能体核心，持有模型、工具注册表，以及可选的记忆与技能提示。
type Agent struct {
	Name     string
	Model    llm.Model
	Registry *tools.Registry

	// Store 为可选长期记忆；为 nil 时本轮对话不落盘。
	Store *memory.Store
	// SkillPrompt 是注入系统提示词的技能说明，可为空。
	SkillPrompt string
}

// NewAgent 创建 Agent，注入模型与工具注册表。
func NewAgent(model llm.Model, registry *tools.Registry) *Agent {
	return &Agent{
		Name:     "go-agent",
		Model:    model,
		Registry: registry,
	}
}

// Chat 是对外对话入口：校验就绪状态后进入运行流程。
// sessionID 为空时不读写长期记忆。
func (a *Agent) Chat(
	ctx context.Context,
	sessionID string,
	input string,
) (string, error) {

	if a == nil || a.Model == nil || a.Registry == nil {
		return "", ErrAgentNotReady
	}

	return a.Run(ctx, sessionID, input)
}

// CallTool 直接调用指定工具（绕过模型，用于测试与确定性执行）。
func (a *Agent) CallTool(
	ctx context.Context,
	toolName string,
	args map[string]any,
) (string, error) {

	if a == nil {
		return "", fmt.Errorf("agent is nil")
	}

	if a.Registry == nil {
		return "", fmt.Errorf("tool registry is nil")
	}

	if ctx == nil {
		return "", fmt.Errorf("context is nil")
	}

	tool, exists := a.Registry.Get(toolName)
	if !exists {
		return "", fmt.Errorf("tool %q not found", toolName)
	}

	return tool.Execute(ctx, args)
}

// ListTools 获取工具列表。
func (a *Agent) ListTools() []string {
	if a == nil || a.Registry == nil {
		return []string{}
	}

	return a.Registry.List()
}
