// Package tools provides the built-in tool registry and implementations the
// agent exposes to the model (file access, shell, etc.).
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"picoclaw/pkg/config"
	"picoclaw/pkg/cron"
	"picoclaw/pkg/llm"
)

// Tool is a single capability the model can invoke by name.
type Tool interface {
	// Schema describes the tool and its JSON arguments for the model.
	Schema() llm.ToolSchema
	// Execute runs the tool with raw JSON args and returns a text result.
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

// Registry holds the tools available to an agent.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry builds an empty registry.
func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

// Default builds a registry with all built-in tools, bound to the sandbox.
// If sched is non-nil, the cron tool is registered too.
func Default(sandbox config.Sandbox, workspace string, sched *cron.Scheduler) *Registry {
	r := NewRegistry()
	r.Add(&readFile{sandbox: sandbox})
	r.Add(&readFileLines{sandbox: sandbox})
	r.Add(&writeFile{sandbox: sandbox})
	r.Add(&appendFile{sandbox: sandbox})
	r.Add(&listDir{sandbox: sandbox})
	r.Add(&shell{workspace: workspace})
	r.Add(&webFetch{})
	r.Add(&webSearch{})
	if sched != nil {
		r.Add(&cronTool{sched: sched})
	}
	return r
}

// Add registers a tool by its schema name.
func (r *Registry) Add(t Tool) { r.tools[t.Schema().Name] = t }

// Get returns the tool registered under name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// Schemas returns the schemas of all tools, sorted by name, optionally
// filtered by an allowlist (empty allow = all tools).
func (r *Registry) Schemas(allow []string) []llm.ToolSchema {
	allowed := func(name string) bool {
		if len(allow) == 0 {
			return true
		}
		for _, a := range allow {
			if a == name {
				return true
			}
		}
		return false
	}
	var out []llm.ToolSchema
	for name, t := range r.tools {
		if allowed(name) {
			out = append(out, t.Schema())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Run executes a tool call by name, returning a text result. Unknown tools and
// tool errors are returned as text so the model can react instead of aborting.
func (r *Registry) Run(ctx context.Context, call llm.ToolCall) string {
	t, ok := r.tools[call.Name]
	if !ok {
		return fmt.Sprintf("error: unknown tool %q", call.Name)
	}
	args := json.RawMessage(call.Arguments)
	if len(args) == 0 {
		args = json.RawMessage("{}")
	}
	out, err := t.Execute(ctx, args)
	if err != nil {
		return "error: " + err.Error()
	}
	return out
}
