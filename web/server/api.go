package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"picoclaw/pkg/agent"
	"picoclaw/pkg/config"
)

// handleSystem returns a non-sensitive runtime summary for the UI.
func (l *Launcher) handleSystem(w http.ResponseWriter, _ *http.Request) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"version":       "0.1.0-phase2",
		"workspace":     l.cfg.Workspace,
		"default_model": l.cfg.Agents.Defaults.ModelName,
		"agent_ready":   l.agent != nil,
	})
}

// --- models ---

type modelDTO struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url,omitempty"`
	HasKey  bool   `json:"has_key"`
}

func (l *Launcher) handleModels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		l.mu.RLock()
		defer l.mu.RUnlock()
		out := make([]modelDTO, 0, len(l.cfg.ModelList))
		for _, m := range l.cfg.ModelList {
			out = append(out, modelDTO{
				Name:    m.Name,
				BaseURL: m.BaseURL,
				HasKey:  m.APIKey != "" || l.cfg.CredentialFor(m.Protocol()) != "",
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"models":        out,
			"default_model": l.cfg.Agents.Defaults.ModelName,
		})
	case http.MethodPost:
		var body struct {
			Name      string `json:"name"`
			BaseURL   string `json:"base_url"`
			APIKey    string `json:"api_key"`
			SetDefault bool  `json:"set_default"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !strings.Contains(body.Name, "/") {
			writeErr(w, http.StatusBadRequest, `name must be "protocol/model"`)
			return
		}
		l.mu.Lock()
		defer l.mu.Unlock()
		// Upsert by name.
		updated := false
		for i := range l.cfg.ModelList {
			if l.cfg.ModelList[i].Name == body.Name {
				l.cfg.ModelList[i].BaseURL = body.BaseURL
				if body.APIKey != "" {
					l.cfg.ModelList[i].APIKey = body.APIKey
				}
				updated = true
				break
			}
		}
		if !updated {
			l.cfg.ModelList = append(l.cfg.ModelList, config.ModelEntry{
				Name:    body.Name,
				BaseURL: body.BaseURL,
				APIKey:  body.APIKey,
			})
		}
		if body.SetDefault || l.cfg.Agents.Defaults.ModelName == "" {
			l.cfg.Agents.Defaults.ModelName = body.Name
		}
		l.persistAndReload(w)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "GET or POST")
	}
}

func (l *Launcher) handleModelByName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/models/")
	if name == "" {
		writeErr(w, http.StatusBadRequest, "missing model name")
		return
	}
	switch r.Method {
	case http.MethodDelete:
		l.mu.Lock()
		defer l.mu.Unlock()
		next := l.cfg.ModelList[:0]
		removed := false
		for _, m := range l.cfg.ModelList {
			if m.Name == name {
				removed = true
				continue
			}
			next = append(next, m)
		}
		l.cfg.ModelList = next
		if !removed {
			writeErr(w, http.StatusNotFound, "no such model")
			return
		}
		if l.cfg.Agents.Defaults.ModelName == name {
			l.cfg.Agents.Defaults.ModelName = ""
			if len(l.cfg.ModelList) > 0 {
				l.cfg.Agents.Defaults.ModelName = l.cfg.ModelList[0].Name
			}
		}
		l.persistAndReload(w)
	case http.MethodPut:
		// Set as default model.
		l.mu.Lock()
		defer l.mu.Unlock()
		if _, ok := l.cfg.ModelByName(name); !ok {
			writeErr(w, http.StatusNotFound, "no such model")
			return
		}
		l.cfg.Agents.Defaults.ModelName = name
		l.persistAndReload(w)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "PUT or DELETE")
	}
}

// --- credentials ---

func (l *Launcher) handleCredentials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		l.mu.RLock()
		defer l.mu.RUnlock()
		type credDTO struct {
			Protocol string `json:"protocol"`
			HasKey   bool   `json:"has_key"`
		}
		// Surface every protocol referenced by a model, plus any stored cred.
		seen := map[string]bool{}
		out := []credDTO{}
		add := func(proto string) {
			if proto == "" || seen[proto] {
				return
			}
			seen[proto] = true
			out = append(out, credDTO{Protocol: proto, HasKey: l.cfg.CredentialFor(proto) != ""})
		}
		for _, m := range l.cfg.ModelList {
			add(m.Protocol())
		}
		for proto := range l.cfg.Credentials {
			add(proto)
		}
		writeJSON(w, http.StatusOK, map[string]any{"credentials": out})
	case http.MethodPost:
		var body struct {
			Protocol string `json:"protocol"`
			APIKey   string `json:"api_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Protocol == "" || body.APIKey == "" {
			writeErr(w, http.StatusBadRequest, "protocol and api_key required")
			return
		}
		l.mu.Lock()
		defer l.mu.Unlock()
		if l.cfg.Credentials == nil {
			l.cfg.Credentials = map[string]string{}
		}
		l.cfg.Credentials[body.Protocol] = body.APIKey
		l.persistAndReload(w)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "GET or POST")
	}
}

func (l *Launcher) handleCredentialByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeErr(w, http.StatusMethodNotAllowed, "DELETE only")
		return
	}
	proto := strings.TrimPrefix(r.URL.Path, "/api/credentials/")
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cfg.Credentials != nil {
		delete(l.cfg.Credentials, proto)
	}
	l.persistAndReload(w)
}

// persistAndReload saves the config and rebuilds the agent. The caller must
// hold l.mu (write lock). It writes the HTTP response.
func (l *Launcher) persistAndReload(w http.ResponseWriter) {
	if err := l.cfg.Save(); err != nil {
		writeErr(w, http.StatusInternalServerError, "save: "+err.Error())
		return
	}
	if ag, err := agent.New(l.cfg); err == nil {
		l.agent = ag
	} else {
		l.agent = nil
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
