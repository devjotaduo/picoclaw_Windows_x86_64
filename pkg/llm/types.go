// Package llm defines the provider-agnostic message and tool types shared
// across the agent loop, providers, and tools.
package llm

// Role identifies who produced a message in a conversation.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is a single entry in a conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
	// Name optionally identifies the author (e.g. a tool name).
	Name string `json:"name,omitempty"`
	// ToolCalls is set on assistant messages that request tool execution.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// ToolCallID links a RoleTool message back to the call it answers.
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ToolCall is a model request to invoke a named tool with JSON arguments.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // raw JSON object
}

// ToolSchema advertises a tool to the model.
type ToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema object
}

// Request is a single completion call.
type Request struct {
	Model       string
	Messages    []Message
	Tools       []ToolSchema
	Temperature float64
	MaxTokens   int
}

// Usage reports token accounting for a response.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Response is the model's reply to a Request.
type Response struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason string
	Usage        Usage
}
