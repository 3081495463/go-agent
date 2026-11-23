package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-agent/internal/llm"
	"go-agent/internal/memory"
)

var ErrAgentNotReady = errors.New("Agent 尚未初始化")

const maxToolSteps = 30

// stepsBeforeHint 在剩余步数不足时提醒模型停止调度工具、给出最终回复。
const stepsBeforeHint = 4

// maxHistory 参与下一轮对话的最近用户/助手消息条数，避免上下文无限膨胀。
const maxHistory = 20

func (a *Agent) Run(
	ctx context.Context,
	sessionID string,
	input string,
) (string, error) {

	systemContent := SystemPrompt
	if a.SkillPrompt != "" {
		systemContent += a.SkillPrompt
	}

	// 从长期记忆恢复历史（无 Store 或无会话 ID 时为空）。
	history := a.loadHistory(sessionID)

	messages := []llm.Message{
		{Role: "system", Content: systemContent},
	}
	messages = append(messages, history...)
	messages = append(messages, llm.Message{Role: "user", Content: input})

	definitions := BuildToolDefinitions(a.Registry)

	var answer llm.Message

	for step := 0; step < maxToolSteps; step++ {

		// 剩余步数不足时，提醒模型不要再发起新的工具调用，直接汇总结果回复。
		if remaining := maxToolSteps - step; remaining == stepsBeforeHint {
			messages = append(messages, llm.Message{
				Role: "system",
				Content: fmt.Sprintf("剩余工具调用次数有限（还剩约 %d 次）。若关键文件已生成，请停止调用工具，直接向用户总结结果与文件位置；不要继续试探性调用。", remaining),
			})
		}

		var err error
		answer, _, err = a.Model.Chat(ctx, messages, definitions)
		if err != nil {
			return "", fmt.Errorf("模型调用失败: %w", err)
		}

		messages = append(messages, answer)

		// 没有工具调用，说明模型已经给出最终回答
		if len(answer.ToolCalls) == 0 {
			a.saveHistory(sessionID, input, answer)
			return answer.Content, nil
		}

		for _, call := range answer.ToolCalls {

			var args map[string]any

			if err := json.Unmarshal(
				[]byte(call.Function.Arguments),
				&args,
			); err != nil {

				messages = append(messages, llm.Message{
					Role:       "tool",
					ToolCallID: call.ID,
					Content:    "工具参数不是有效 JSON",
				})

				continue
			}

			tool, ok := a.Registry.Get(call.Function.Name)

			if !ok {
				messages = append(messages, llm.Message{
					Role:       "tool",
					ToolCallID: call.ID,
					Content:    "工具不存在",
				})

				continue
			}

			result, err := tool.Execute(ctx, args)

			if err != nil {
				result = "工具执行失败: " + err.Error()
			}

			if len(result) > 12000 {
				result = result[:12000] + "\n[结果已截断]"
			}

			messages = append(messages, llm.Message{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    result,
			})
		}
	}

	return "", fmt.Errorf("Agent 达到最大工具调用次数")
}

// loadHistory 读取指定会话的历史对话消息；不可用时返回空切片。
func (a *Agent) loadHistory(sessionID string) []llm.Message {
	if a.Store == nil || sessionID == "" {
		return nil
	}

	conv, err := a.Store.Load(sessionID)
	if err != nil {
		return nil
	}

	history := conv.Messages
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}

	return history
}

// saveHistory 把本轮用户输入与助手最终回复追加到长期记忆并落盘。
func (a *Agent) saveHistory(sessionID, input string, answer llm.Message) {
	if a.Store == nil || sessionID == "" {
		return
	}

	conv, err := a.Store.Load(sessionID)
	if err != nil {
		return
	}

	conv.ID = sessionID
	conv.Messages = append(conv.Messages,
		llm.Message{Role: "user", Content: input},
		llm.Message{Role: "assistant", Content: answer.Content},
	)

	if len(conv.Messages) > maxHistory {
		conv.Messages = conv.Messages[len(conv.Messages)-maxHistory:]
	}

	conv.Summary = memory.BuildSummary(conv.Messages)

	_ = a.Store.Save(conv)
}
