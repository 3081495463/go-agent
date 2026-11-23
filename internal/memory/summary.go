package memory

import (
	"strings"

	"go-agent/internal/llm"
)

// maxNotes 长期保留的关键信息条数上限，避免记忆无限膨胀。
const maxNotes = 20

// maxHistoryContent 单条历史消息落盘时保留的最大字符数，超出截断，
// 避免把长回复、过程性描述整段写进记忆。
const maxHistoryContent = 500

// CompactMessages 只保留最近 maxHistory 条非空对话，并压缩每条长度；
// 不做模型摘要（不额外调用 LLM），只做截断与清理。
func CompactMessages(messages []llm.Message, maxHistory int) []llm.Message {
	kept := make([]llm.Message, 0, len(messages))
	for _, m := range messages {
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		if len(content) > maxHistoryContent {
			content = content[:maxHistoryContent] + "…"
		}
		kept = append(kept, llm.Message{Role: m.Role, Content: content, ToolCallID: m.ToolCallID})
	}
	if len(kept) > maxHistory {
		kept = kept[len(kept)-maxHistory:]
	}
	return kept
}

// ExtractNotes 从一轮对话里挑出值得长期保留的关键信息（轻量规则版）。
// 目前只沉淀显式的用户偏好/约定；真正的语义抽取可后续接模型完成。
// 返回按 key 去重的 Note 列表。
func ExtractNotes(input string, existing []Note) []Note {
	notes := existing
	// 占位：规则抽取可在此扩展，例如识别“喜欢/偏好/要求”句式。
	_ = input
	if len(notes) > maxNotes {
		notes = notes[len(notes)-maxNotes:]
	}
	return notes
}
