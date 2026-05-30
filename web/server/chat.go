package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"picoclaw/pkg/agent"
)

// chatEvent is one Server-Sent Event emitted during a chat turn.
type chatEvent struct {
	Type   string `json:"type"` // tool_call | tool_result | assistant | done | error
	Name   string `json:"name,omitempty"`
	Args   string `json:"args,omitempty"`
	Text   string `json:"text,omitempty"`
	Result string `json:"result,omitempty"`
}

// streamObserver forwards agent loop events onto a channel as chatEvents.
type streamObserver struct{ ch chan chatEvent }

func (o *streamObserver) OnAssistant(text string) {
	o.ch <- chatEvent{Type: "assistant", Text: text}
}
func (o *streamObserver) OnToolCall(name, args string) {
	o.ch <- chatEvent{Type: "tool_call", Name: name, Args: args}
}
func (o *streamObserver) OnToolResult(name, result string) {
	o.ch <- chatEvent{Type: "tool_result", Name: name, Result: result}
}

// handleChatStream runs a single agent turn and streams events as SSE. The
// frontend reads this with fetch + ReadableStream (EventSource is GET-only).
func (l *Launcher) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var body struct {
		Message string `json:"message"`
		// Agent, when set, targets a named agent (the isolated agent page) so the
		// reply comes from that agent — its own name, prompt and model.
		Agent string `json:"agent,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Message == "" {
		writeErr(w, http.StatusBadRequest, "missing message")
		return
	}

	l.mu.RLock()
	ag := l.agent
	l.mu.RUnlock()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	send := func(e chatEvent) {
		data, _ := json.Marshal(e)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Targeting a named agent (isolated agent page): build it on the fly so the
	// reply uses that agent's own name, prompt and model.
	if body.Agent != "" {
		named, err := l.buildNamedAgent(body.Agent)
		if err != nil {
			send(chatEvent{Type: "error", Text: err.Error()})
			send(chatEvent{Type: "done"})
			return
		}
		ag = named
	}

	if ag == nil {
		send(chatEvent{Type: "error", Text: "agent not ready — set a valid model/credential"})
		send(chatEvent{Type: "done"})
		return
	}

	events := make(chan chatEvent, 16)
	obs := &streamObserver{ch: events}

	// Run the agent in a goroutine so we can stream events as they arrive.
	// A fresh Agent value shares the provider/tools but gets its own observer.
	runAgent := *ag
	runAgent.Observer = obs

	done := make(chan struct{})
	var finalText string
	var runErr error
	go func() {
		finalText, runErr = runAgent.Run(r.Context(), body.Message)
		close(done)
	}()

	for {
		select {
		case e := <-events:
			// Skip streaming intermediate assistant text; the final one is sent
			// after the run completes to avoid duplicates.
			if e.Type != "assistant" {
				send(e)
			}
		case <-done:
			// Drain any buffered events.
			for {
				select {
				case e := <-events:
					if e.Type != "assistant" {
						send(e)
					}
					continue
				default:
				}
				break
			}
			if runErr != nil {
				send(chatEvent{Type: "error", Text: runErr.Error()})
			} else {
				send(chatEvent{Type: "assistant", Text: finalText})
			}
			send(chatEvent{Type: "done"})
			return
		case <-r.Context().Done():
			return
		}
	}
}

// ensure agent.Observer interface is satisfied at compile time.
var _ agent.Observer = (*streamObserver)(nil)
