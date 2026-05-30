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

// anthropic implements Provider against the Anthropic Messages API, whose wire
// format (content blocks, separate system field, tool_use/tool_result) differs
// from the OpenAI shape.
type anthropic struct {
	model   string
	apiKey  string
	baseURL string
	client  *http.Client
}

func init() {
	Register("anthropic", newAnthropic)
}

func newAnthropic(entry config.ModelEntry) (Provider, error) {
	key := entry.APIKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	if key == "" {
		return nil, fmt.Errorf("anthropic: no api_key and ANTHROPIC_API_KEY unset")
	}
	base := entry.BaseURL
	if base == "" {
		base = "https://api.anthropic.com/v1"
	}
	return &anthropic{
		model:   entry.Model(),
		apiKey:  key,
		baseURL: strings.TrimRight(base, "/"),
		client:  &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (a *anthropic) Name() string { return "anthropic" }

type antBlock struct {
	Type string `json:"type"`
	// text
	Text string `json:"text,omitempty"`
	// tool_use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
	// tool_result
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

type antMessage struct {
	Role    string     `json:"role"`
	Content []antBlock `json:"content"`
}

type antTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type antRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	System    string       `json:"system,omitempty"`
	Messages  []antMessage `json:"messages"`
	Tools     []antTool    `json:"tools,omitempty"`
}

type antResponse struct {
	Content    []antBlock `json:"content"`
	StopReason string     `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (a *anthropic) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	body := antRequest{Model: a.model, MaxTokens: maxTokens}

	for _, m := range req.Messages {
		switch m.Role {
		case llm.RoleSystem:
			if body.System != "" {
				body.System += "\n\n"
			}
			body.System += m.Content
		case llm.RoleUser:
			body.Messages = append(body.Messages, antMessage{
				Role:    "user",
				Content: []antBlock{{Type: "text", Text: m.Content}},
			})
		case llm.RoleAssistant:
			var blocks []antBlock
			if m.Content != "" {
				blocks = append(blocks, antBlock{Type: "text", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				blocks = append(blocks, antBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: json.RawMessage(orEmptyObj(tc.Arguments)),
				})
			}
			body.Messages = append(body.Messages, antMessage{Role: "assistant", Content: blocks})
		case llm.RoleTool:
			// Tool results are user-role messages with a tool_result block.
			body.Messages = append(body.Messages, antMessage{
				Role: "user",
				Content: []antBlock{{
					Type:      "tool_result",
					ToolUseID: m.ToolCallID,
					Content:   m.Content,
				}},
			})
		}
	}

	for _, t := range req.Tools {
		body.Tools = append(body.Tools, antTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return llm.Response{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/messages", bytes.NewReader(raw))
	if err != nil {
		return llm.Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return llm.Response{}, fmt.Errorf("anthropic request: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var parsed antResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return llm.Response{}, fmt.Errorf("anthropic decode (status %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != nil {
		return llm.Response{}, fmt.Errorf("anthropic error: %s", parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return llm.Response{}, fmt.Errorf("anthropic status %d: %s", resp.StatusCode, string(data))
	}

	out := llm.Response{
		FinishReason: parsed.StopReason,
		Usage: llm.Usage{
			PromptTokens:     parsed.Usage.InputTokens,
			CompletionTokens: parsed.Usage.OutputTokens,
			TotalTokens:      parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		},
	}
	for _, b := range parsed.Content {
		switch b.Type {
		case "text":
			out.Content += b.Text
		case "tool_use":
			out.ToolCalls = append(out.ToolCalls, llm.ToolCall{
				ID:        b.ID,
				Name:      b.Name,
				Arguments: string(b.Input),
			})
		}
	}
	return out, nil
}

func orEmptyObj(s string) string {
	if strings.TrimSpace(s) == "" {
		return "{}"
	}
	return s
}
