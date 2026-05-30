// Package channels connects external messaging platforms to the agent. Each
// channel turns inbound platform messages into agent Runs and sends replies
// back. Phase 1 ships the Telegram channel.
package channels

import (
	"context"
	"net/http"
)

// Handler answers a user message and returns the reply text.
type Handler func(ctx context.Context, user, text string) (string, error)

// Channel is a long-running connector to one messaging platform (long polling
// or a persistent connection).
type Channel interface {
	// Name identifies the channel ("telegram", ...).
	Name() string
	// Run blocks, dispatching inbound messages to handle until ctx is done.
	Run(ctx context.Context, handle Handler) error
}

// WebhookChannel is a channel driven by inbound HTTP requests, mounted on the
// shared Gateway at Path. Webhook channels do not own a connection of their
// own; the gateway routes requests to Handler.
type WebhookChannel interface {
	// Name identifies the channel ("slack", "webhook", ...).
	Name() string
	// Path is the gateway-relative mount point, e.g. "/webhooks/slack".
	Path() string
	// Handler returns the HTTP handler, closed over the agent dispatcher.
	Handler(handle Handler) http.HandlerFunc
}
