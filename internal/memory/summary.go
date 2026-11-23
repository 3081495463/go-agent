package memory

import (
	"strings"

	"go-agent/internal/llm"
)

func BuildSummary(messages []llm.Message) string {

	var builder strings.Builder

	count := 0

	for _, message := range messages {

		if message.Role != "user" &&
			message.Role != "assistant" {
			continue
		}

		if strings.TrimSpace(message.Content) == "" {
			continue
		}

		builder.WriteString(message.Role)
		builder.WriteString(": ")
		builder.WriteString(message.Content)
		builder.WriteString("\n")

		count++

		if count >= 8 {
			break
		}
	}

	return builder.String()
}
