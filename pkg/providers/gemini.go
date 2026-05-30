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

// gemini implements Provider against Google's generateContent API, which uses
// "contents" with parts and functionCall/functionResponse for tool use.
type gemini struct {
	model   string
	apiKey  string
	baseURL string
	client  *http.Client
}

func init() {
	Register("gemini", newGemini)
	Register("google", newGemini)
}

func newGemini(entry config.ModelEntry) (Provider, error) {
	key := entry.APIKey
	if key == "" {
		key = os.Getenv("GEMINI_API_KEY")
	}
	if key == "" {
		return nil, fmt.Errorf("gemini: no api_key and GEMINI_API_KEY unset")
	}
	base := entry.BaseURL
	if base == "" {
		base = "https://generativelanguage.googleapis.com/v1beta"
	}
	return &gemini{
		model:   entry.Model(),
		apiKey:  key,
		baseURL: strings.TrimRight(base, "/"),
		client:  &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (g *gemini) Name() string { return "gemini" }

type gemPart struct {
	Text             string          `json:"text,omitempty"`
	FunctionCall     *gemFunctionCall `json:"functionCall,omitempty"`
	FunctionResponse *gemFunctionResp `json:"functionResponse,omitempty"`
}

type gemFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type gemFunctionResp struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type gemContent struct {
	Role  string    `json:"role,omitempty"`
	Parts []gemPart `json:"parts"`
}

type gemRequest struct {
	Contents          []gemContent    `json:"contents"`
	SystemInstruction *gemContent     `json:"systemInstruction,omitempty"`
	Tools             []gemToolWrapper `json:"tools,omitempty"`
}

type gemToolWrapper struct {
	FunctionDeclarations []gemFuncDecl `json:"functionDeclarations"`
}

type gemFuncDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type gemResponse struct {
	Candidates []struct {
		Content      gemContent `json:"content"`
		FinishReason string     `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (g *gemini) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	body := gemRequest{}
	for _, m := range req.Messages {
		switch m.Role {
		case llm.RoleSystem:
			body.SystemInstruction = &gemContent{Parts: []gemPart{{Text: m.Content}}}
		case llm.RoleUser:
			body.Contents = append(body.Contents, gemContent{Role: "user", Parts: []gemPart{{Text: m.Content}}})
		case llm.RoleAssistant:
			var parts []gemPart
			if m.Content != "" {
				parts = append(parts, gemPart{Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				args := map[string]any{}
				_ = json.Unmarshal([]byte(orEmptyObj(tc.Arguments)), &args)
				parts = append(parts, gemPart{FunctionCall: &gemFunctionCall{Name: tc.Name, Args: args}})
			}
			body.Contents = append(body.Contents, gemContent{Role: "model", Parts: parts})
		case llm.RoleTool:
			body.Contents = append(body.Contents, gemContent{
				Role:  "user",
				Parts: []gemPart{{FunctionResponse: &gemFunctionResp{Name: m.Name, Response: map[string]any{"result": m.Content}}}},
			})
		}
	}

	if len(req.Tools) > 0 {
		decls := make([]gemFuncDecl, 0, len(req.Tools))
		for _, t := range req.Tools {
			decls = append(decls, gemFuncDecl{Name: t.Name, Description: t.Description, Parameters: t.Parameters})
		}
		body.Tools = []gemToolWrapper{{FunctionDeclarations: decls}}
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return llm.Response{}, err
	}
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", g.baseURL, g.model, g.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return llm.Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return llm.Response{}, fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var parsed gemResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return llm.Response{}, fmt.Errorf("gemini decode (status %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != nil {
		return llm.Response{}, fmt.Errorf("gemini error: %s", parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return llm.Response{}, fmt.Errorf("gemini status %d: %s", resp.StatusCode, string(data))
	}
	if len(parsed.Candidates) == 0 {
		return llm.Response{}, fmt.Errorf("gemini: no candidates")
	}

	cand := parsed.Candidates[0]
	out := llm.Response{
		FinishReason: cand.FinishReason,
		Usage: llm.Usage{
			PromptTokens:     parsed.UsageMetadata.PromptTokenCount,
			CompletionTokens: parsed.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      parsed.UsageMetadata.TotalTokenCount,
		},
	}
	for _, p := range cand.Content.Parts {
		if p.Text != "" {
			out.Content += p.Text
		}
		if p.FunctionCall != nil {
			argsJSON, _ := json.Marshal(p.FunctionCall.Args)
			out.ToolCalls = append(out.ToolCalls, llm.ToolCall{
				ID:        "gem_" + p.FunctionCall.Name,
				Name:      p.FunctionCall.Name,
				Arguments: string(argsJSON),
			})
		}
	}
	return out, nil
}
