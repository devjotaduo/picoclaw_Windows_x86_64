// Package agent implements the PicoClaw agent loop as a small pipeline:
// setup -> llm -> execute -> finalize, repeated until the model stops calling
// tools or the turn budget is exhausted.
package agent

import (
	"context"
	"log"

	"picoclaw/pkg/config"
	"picoclaw/pkg/cron"
	"picoclaw/pkg/llm"
	"picoclaw/pkg/providers"
	"picoclaw/pkg/tools"
)

// Agent runs a single conversational agent over a provider and tool registry.
type Agent struct {
	provider     providers.Provider
	tools        *tools.Registry
	systemPrompt string
	toolAllow    []string
	maxTurns     int
	sched        *cron.Scheduler
	// Observer, if set, receives lifecycle events for logging/UX.
	Observer Observer
}

// Scheduler returns the agent's cron scheduler. Callers (gateway/web) start it
// with Scheduler().Run(ctx) to fire scheduled prompts.
func (a *Agent) Scheduler() *cron.Scheduler { return a.sched }

// Observer receives runtime events from the loop. A nil Observer is ignored.
type Observer interface {
	OnAssistant(text string)
	OnToolCall(name, args string)
	OnToolResult(name, result string)
}

// New builds an Agent for the default agent defined in cfg.
func New(cfg *config.Config) (*Agent, error) {
	entry, ok := cfg.ModelByName(cfg.Agents.Defaults.ModelName)
	if !ok {
		if len(cfg.ModelList) == 0 {
			return nil, errNoModel
		}
		entry = cfg.ModelList[0]
	}
	// Fall back to the shared credentials map when the entry has no key.
	if entry.APIKey == "" {
		entry.APIKey = cfg.CredentialFor(entry.Protocol())
	}
	prov, err := providers.Resolve(entry)
	if err != nil {
		return nil, err
	}

	sched := cron.New()
	reg := tools.Default(cfg.Sandbox, cfg.Workspace, sched)

	prompt := cfg.Agents.Defaults.SystemPrompt
	if prompt == "" {
		prompt = defaultSystemPrompt
	}
	a := &Agent{
		provider:     prov,
		tools:        reg,
		systemPrompt: prompt,
		toolAllow:    cfg.Agents.Defaults.Tools,
		maxTurns:     cfg.Agents.Defaults.MaxTurns,
		sched:        sched,
	}
	// Fired jobs run their prompt as a fresh turn and log the result.
	sched.SetHandler(func(ctx context.Context, job cron.Job) {
		out, err := a.Run(ctx, job.Prompt)
		if err != nil {
			log.Printf("cron %s error: %v", job.ID, err)
			return
		}
		log.Printf("cron %s fired: %s", job.ID, out)
	})
	return a, nil
}

// Run executes a single user message to completion and returns the final
// assistant text. History is seeded with the system prompt each call; full
// session persistence arrives in a later phase.
func (a *Agent) Run(ctx context.Context, userMessage string) (string, error) {
	st := &turnState{
		messages: []llm.Message{
			{Role: llm.RoleSystem, Content: a.systemPrompt},
			{Role: llm.RoleUser, Content: userMessage},
		},
	}
	return a.loop(ctx, st)
}

const defaultSystemPrompt = "You are PicoClaw, an ultra-lightweight personal AI assistant. " +
	"You can read and write files, list directories, and run shell commands via tools. " +
	"Use tools when they help; keep answers concise and act directly."
