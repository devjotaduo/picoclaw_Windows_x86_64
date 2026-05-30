// Package providers adapts LLM backends behind a single interface, resolved
// from "protocol/model" entries in the config model_list.
package providers

import (
	"context"
	"fmt"

	"picoclaw/pkg/config"
	"picoclaw/pkg/llm"
)

// Provider performs a single chat completion (optionally with tool calling).
type Provider interface {
	// Name returns the protocol identifier, e.g. "openai".
	Name() string
	// Complete sends req and returns the model response.
	Complete(ctx context.Context, req llm.Request) (llm.Response, error)
}

// Constructor builds a Provider from a model_list entry.
type Constructor func(entry config.ModelEntry) (Provider, error)

// registry maps protocol -> constructor. Adapters register at init time.
var registry = map[string]Constructor{}

// Register binds a protocol name to its constructor.
func Register(protocol string, c Constructor) { registry[protocol] = c }

// Resolve builds the Provider for a model_list entry by its protocol.
func Resolve(entry config.ModelEntry) (Provider, error) {
	c, ok := registry[entry.Protocol()]
	if !ok {
		return nil, fmt.Errorf("unknown provider protocol %q (model %q)", entry.Protocol(), entry.Name)
	}
	return c(entry)
}
