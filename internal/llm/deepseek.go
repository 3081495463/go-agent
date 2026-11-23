package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultDeepseekBaseURL = "https://api.deepseek.com"
	defaultDeepseekModel   = "deepseek-chat"
	defaultDeepseekTimeout = 90 * time.Second
	maxResponseBytes       = 8 << 20 // 8 MB
)

// DeepseekClient 调用 DeepSeek 兼容 chat/completions 接口的客户端。
type DeepseekClient struct {
	APIKey      string
	Model       string
	BaseURL     string
	Temperature float64
	Client      *http.Client
}

// NewDeepseekClient 创建客户端。model 为空时使用默认 deepseek-chat；
// baseURL 为空时使用 DeepSeek 官方地址，可传入兼容 OpenAI 接口的中转地址。
func NewDeepseekClient(apiKey, model, baseURL string) (*DeepseekClient, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("DeepSeek API Key 未配置")
	}
	if model == "" {
		model = defaultDeepseekModel
	}
	if baseURL == "" {
		baseURL = defaultDeepseekBaseURL
	}

	return &DeepseekClient{
		APIKey:      apiKey,
		Model:       model,
		BaseURL:     strings.TrimRight(baseURL, "/"),
		Temperature: 0.2,
		Client: &http.Client{
			Timeout: defaultDeepseekTimeout,
		},
	}, nil
}

type chatRequest struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	ToolChoice  string           `json:"tool_choice,omitempty"`
	Temperature float64          `json:"temperature"`
	Stream      bool             `json:"stream"`
}

// Chat 实现 Model 接口。
func (d *DeepseekClient) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
) (Message, int, error) {

	if d == nil {
		return Message{}, 0, fmt.Errorf("deepseek client is nil")
	}

	body := chatRequest{
		Model:       d.Model,
		Messages:    messages,
		Tools:       tools,
		Temperature: d.Temperature,
		Stream:      false,
	}

	if len(tools) > 0 {
		body.ToolChoice = "auto"
	}

	data, err := json.Marshal(body)
	if err != nil {
		return Message{}, 0, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(d.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(data),
	)
	if err != nil {
		return Message{}, 0, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.APIKey)

	resp, err := d.Client.Do(req)
	if err != nil {
		return Message{}, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return Message{}, 0, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Message{}, 0, fmt.Errorf(
			"deepseek API 状态码 %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(result)),
		)
	}

	var response Response
	if err := json.Unmarshal(result, &response); err != nil {
		return Message{}, 0, fmt.Errorf("decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		return Message{}, 0, fmt.Errorf("模型没有返回内容")
	}

	return response.Choices[0].Message,
		response.Usage.TotalTokens,
		nil
}
