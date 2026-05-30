// Package gateway exposes a small shared HTTP server. In Phase 1 it offers a
// health endpoint and a synchronous /agent endpoint; channel webhooks will
// mount here in later phases.
package gateway

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"picoclaw/pkg/agent"
)

// Gateway is the shared HTTP front door for the runtime.
type Gateway struct {
	addr  string
	agent *agent.Agent
}

// New builds a Gateway bound to addr that serves the given agent.
func New(addr string, ag *agent.Agent) *Gateway {
	return &Gateway{addr: addr, agent: ag}
}

// Run starts the HTTP server and blocks until ctx is cancelled.
func (g *Gateway) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", g.handleHealth)
	mux.HandleFunc("/agent", g.handleAgent)

	srv := &http.Server{
		Addr:         g.addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute,
	}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	log.Printf("gateway listening on http://%s", g.addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (g *Gateway) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (g *Gateway) handleAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "POST only"})
		return
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing message"})
		return
	}
	reply, err := g.agent.Run(r.Context(), body.Message)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reply": reply})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
