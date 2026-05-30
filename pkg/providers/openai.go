package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"picoclaw/pkg/config"
	"picoclaw/pkg/llm"
)

// openAI implements Provider against the OpenAI Chat Completions API and any
// OpenAI-compatible endpoint (set base_url in the model entry).
type openAI struct {
	model   string
	apiKey  string
	baseURL string
	client  *http.Client
}

func init() {
	Register("openai", newOpenAI)
	// OpenAI-compatible backends reuse the same wire format.
	Register("openai_compat", newOpenAI)
}

func newOpenAI(entry config.ModelEntry) (Provider, error) {
	key := entry.APIKey
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	if key == "" {
		return nil, fmt.Errorf("openai: no api_key in config and OPENAI_API_KEY unset")
	}
	base := entry.BaseURL
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	return &openAI{
		model:   entry.Model(),
		apiKey:  key,
		baseURL: strings.TrimRight(base, "/"),
		client:  &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (o *openAI) Name() string { return "openai" }

// ---- wire types ----

type oaiMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	Name       string          `json:"name,omitempty"`
	ToolCalls  []oaiToolCall   `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type oaiToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function oaiFunctionCall `json:"function"`
}

type oaiFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type oaiTool struct {
	Type     string         `json:"type"`
	Function oaiToolFunc    `json:"function"`
}

type oaiToolFunc struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type oaiRequest struct {
	Model       string       `json:"model"`
	Messages    []oaiMessage `json:"messages"`
	Tools       []oaiTool    `json:"tools,omitempty"`
	Temperature float64      `json:"temperature,omitempty"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
}

type oaiResponse struct {
	Choices []struct {
		Message      oaiMessage `json:"message"`
		FinishReason string     `json:"finish_reason"`
	} `json:"choices"`
	Usage llm.Usage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (o *openAI) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	body := oaiRequest{
		Model:       o.model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	for _, m := range req.Messages {
		om := oaiMessage{
			Role:       string(m.Role),
			Content:    m.Content,
			Name:       m.Name,
			ToolCallID: m.ToolCallID,
		}
		for _, tc := range m.ToolCalls {
			om.ToolCalls = append(om.ToolCalls, oaiToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: oaiFunctionCall{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			})
		}
		body.Messages = append(body.Messages, om)
	}
	for _, t := range req.Tools {
		body.Tools = append(body.Tools, oaiTool{
			Type: "function",
			Function: oaiToolFunc{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return llm.Response{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return llm.Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return llm.Response{}, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var parsed oaiResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return llm.Response{}, fmt.Errorf("openai decode (status %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != nil {
		return llm.Response{}, fmt.Errorf("openai error: %s", parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return llm.Response{}, fmt.Errorf("openai status %d: %s", resp.StatusCode, string(data))
	}
	if len(parsed.Choices) == 0 {
		return llm.Response{}, fmt.Errorf("openai: no choices returned")
	}

	choice := parsed.Choices[0]
	out := llm.Response{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Usage:        parsed.Usage,
	}
	for _, tc := range choice.Message.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, llm.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	return out, nil
}
