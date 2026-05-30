package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// NamedAgent is a user-defined agent ("template"): a name plus its response
// rules (system prompt) and an optional model + temperature. These are created
// and edited from the Agents tab and persisted to <workspace>/agents.json,
// independently of config.json so the launcher schema stays untouched.
type NamedAgent struct {
	Name         string  `json:"name"`
	Description  string  `json:"description,omitempty"`
	SystemPrompt string  `json:"system_prompt"`
	Model        string  `json:"model,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
	Enabled      bool    `json:"enabled"`
}

// agentsPath is where named agents are stored.
func (l *Launcher) agentsPath() string {
	return filepath.Join(l.cfg.Workspace, "agents.json")
}

func (l *Launcher) loadAgents() ([]NamedAgent, error) {
	data, err := os.ReadFile(l.agentsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return []NamedAgent{}, nil
		}
		return nil, err
	}
	var out []NamedAgent
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (l *Launcher) saveAgents(agents []NamedAgent) error {
	if err := os.MkdirAll(l.cfg.Workspace, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(agents, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.agentsPath(), data, 0o644)
}

// handleAgents serves GET (list + form choices) and POST (create/update).
func (l *Launcher) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		l.mu.RLock()
		defer l.mu.RUnlock()
		agents, err := l.loadAgents()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		models := make([]string, 0, len(l.cfg.ModelList))
		for _, m := range l.cfg.ModelList {
			models = append(models, m.Name)
		}
		sort.Strings(models)
		writeJSON(w, http.StatusOK, map[string]any{
			"agents":  agents,
			"models":  models,
			"default": l.cfg.Agents.Defaults.ModelName,
		})

	case http.MethodPost:
		var a NamedAgent
		if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request: "+err.Error())
			return
		}
		a.Name = strings.TrimSpace(a.Name)
		if a.Name == "" {
			writeErr(w, http.StatusBadRequest, "name is required")
			return
		}
		a.Enabled = true
		l.mu.Lock()
		defer l.mu.Unlock()
		agents, err := l.loadAgents()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		replaced := false
		for i := range agents {
			if strings.EqualFold(agents[i].Name, a.Name) {
				agents[i] = a
				replaced = true
				break
			}
		}
		if !replaced {
			agents = append(agents, a)
		}
		if err := l.saveAgents(agents); err != nil {
			writeErr(w, http.StatusInternalServerError, "save: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "agent": a})

	default:
		writeErr(w, http.StatusMethodNotAllowed, "GET or POST")
	}
}

// handleAgentByName serves DELETE /api/agents/{name}.
func (l *Launcher) handleAgentByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeErr(w, http.StatusMethodNotAllowed, "DELETE only")
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	if name == "" {
		writeErr(w, http.StatusBadRequest, "missing agent name")
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	agents, err := l.loadAgents()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	next := agents[:0]
	removed := false
	for _, a := range agents {
		if strings.EqualFold(a.Name, name) {
			removed = true
			continue
		}
		next = append(next, a)
	}
	if !removed {
		writeErr(w, http.StatusNotFound, "no such agent")
		return
	}
	if err := l.saveAgents(next); err != nil {
		writeErr(w, http.StatusInternalServerError, "save: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
