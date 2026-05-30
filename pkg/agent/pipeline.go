package agent

import (
	"context"
	"errors"

	"picoclaw/pkg/llm"
)

var errNoModel = errors.New("agent: no model configured (empty model_list)")

// turnState carries the evolving conversation and accounting across pipeline
// stages within a single Run.
type turnState struct {
	messages []llm.Message
	lastText string
	turns    int
	tokens   llm.Usage
}

// loop drives the pipeline until the model stops requesting tools or the turn
// budget is exhausted. Each iteration is: llm -> (execute) -> finalize.
func (a *Agent) loop(ctx context.Context, st *turnState) (string, error) {
	for st.turns < a.maxTurns {
		st.turns++

		resp, err := a.stageLLM(ctx, st)
		if err != nil {
			return "", err
		}

		// No tool calls -> this is the final assistant turn.
		if len(resp.ToolCalls) == 0 {
			return a.stageFinalize(st, resp), nil
		}

		a.stageExecute(ctx, st, resp)
	}
	// Budget exhausted: return whatever text we last produced.
	if st.lastText != "" {
		return st.lastText, nil
	}
	return "", errors.New("agent: max turns reached without a final answer")
}

// stageLLM calls the provider and records token usage. (The "setup" stage is
// folded into Run, which seeds the system prompt and user message.)
func (a *Agent) stageLLM(ctx context.Context, st *turnState) (llm.Response, error) {
	req := llm.Request{
		Messages: st.messages,
		Tools:    a.tools.Schemas(a.toolAllow),
	}
	resp, err := a.provider.Complete(ctx, req)
	if err != nil {
		return llm.Response{}, err
	}
	st.tokens.PromptTokens += resp.Usage.PromptTokens
	st.tokens.CompletionTokens += resp.Usage.CompletionTokens
	st.tokens.TotalTokens += resp.Usage.TotalTokens

	// Record the assistant message (text and/or tool calls) into history.
	st.messages = append(st.messages, llm.Message{
		Role:      llm.RoleAssistant,
		Content:   resp.Content,
		ToolCalls: resp.ToolCalls,
	})
	if resp.Content != "" {
		st.lastText = resp.Content
		if a.Observer != nil {
			a.Observer.OnAssistant(resp.Content)
		}
	}
	return resp, nil
}

// stageExecute runs each requested tool and appends results to history so the
// next llm stage can see them.
func (a *Agent) stageExecute(ctx context.Context, st *turnState, resp llm.Response) {
	for _, call := range resp.ToolCalls {
		if a.Observer != nil {
			a.Observer.OnToolCall(call.Name, call.Arguments)
		}
		result := a.tools.Run(ctx, call)
		if a.Observer != nil {
			a.Observer.OnToolResult(call.Name, result)
		}
		st.messages = append(st.messages, llm.Message{
			Role:       llm.RoleTool,
			Name:       call.Name,
			ToolCallID: call.ID,
			Content:    result,
		})
	}
}

// stageFinalize records and returns the final assistant text.
func (a *Agent) stageFinalize(st *turnState, resp llm.Response) string {
	if resp.Content != "" {
		st.lastText = resp.Content
	}
	return st.lastText
}

// Usage reports cumulative token usage from the most recent Run on st. Exposed
// for callers that wire their own state; the simple Run path discards it.
func (st *turnState) Usage() llm.Usage { return st.tokens }
