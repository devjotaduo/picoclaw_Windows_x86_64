// Package config loads PicoClaw's non-sensitive config.json and exposes the
// path sandbox used by file/shell tools.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config is the top-level config.json document.
type Config struct {
	Version   int          `json:"version"`
	Workspace string       `json:"workspace"`
	ModelList []ModelEntry `json:"model_list"`
	Agents    Agents       `json:"agents"`
	Channels  Channels     `json:"channels"`
	Gateway   Gateway      `json:"gateway"`
	Sandbox   Sandbox      `json:"sandbox"`
	// Credentials maps a provider protocol ("openai") to an API key, used as a
	// fallback when a model entry has no api_key of its own.
	Credentials map[string]string `json:"credentials,omitempty"`

	// ActiveAgent, when set, is the name of a named agent (from agents.json)
	// that the default Chat and WhatsApp answer as — its prompt and model, with
	// its name injected. Empty means the generic default assistant.
	ActiveAgent string `json:"active_agent,omitempty"`

	// path records where this config was loaded from, for Save.
	path string
}

// ModelEntry is one "protocol/model" provider binding.
type ModelEntry struct {
	// Name is in the form "protocol/model", e.g. "openai/gpt-4o-mini".
	Name    string `json:"name"`
	APIKey  string `json:"api_key,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
}

// Protocol returns the part before the first slash ("openai").
func (m ModelEntry) Protocol() string {
	if i := strings.Index(m.Name, "/"); i >= 0 {
		return m.Name[:i]
	}
	return m.Name
}

// Model returns the part after the first slash ("gpt-4o-mini").
func (m ModelEntry) Model() string {
	if i := strings.Index(m.Name, "/"); i >= 0 {
		return m.Name[i+1:]
	}
	return m.Name
}

// Agents holds agent-level defaults.
type Agents struct {
	Defaults AgentDefaults `json:"defaults"`
}

// AgentDefaults configures the default agent.
type AgentDefaults struct {
	ModelName    string   `json:"model_name"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	Tools        []string `json:"tools,omitempty"` // allowlist; empty = all
	MaxTurns     int      `json:"max_turns,omitempty"`
}

// Channels holds per-channel settings.
type Channels struct {
	Telegram TelegramChannel `json:"telegram"`
	Slack    SlackChannel    `json:"slack"`
	Webhook  WebhookCfg      `json:"webhook"`
}

// TelegramChannel configures the Telegram bot channel.
type TelegramChannel struct {
	Enabled bool   `json:"enabled"`
	Token   string `json:"token"`
}

// SlackChannel configures the Slack Events API channel.
type SlackChannel struct {
	Enabled       bool   `json:"enabled"`
	BotToken      string `json:"bot_token"`
	SigningSecret string `json:"signing_secret"`
}

// WebhookCfg configures the generic inbound webhook channel.
type WebhookCfg struct {
	Enabled bool `json:"enabled"`
}

// Gateway configures the shared HTTP gateway.
type Gateway struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// Addr returns host:port with sane defaults.
func (g Gateway) Addr() string {
	host := g.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := g.Port
	if port == 0 {
		port = 18790
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// Sandbox restricts which paths tools may read or write.
type Sandbox struct {
	AllowReadPaths  []string `json:"allow_read_paths"`
	AllowWritePaths []string `json:"allow_write_paths"`
}

// Load reads and parses config.json from path, applying defaults.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	c.path = path
	if c.Workspace == "" {
		c.Workspace = "."
	}
	abs, err := filepath.Abs(c.Workspace)
	if err == nil {
		c.Workspace = abs
	}
	if c.Agents.Defaults.MaxTurns == 0 {
		c.Agents.Defaults.MaxTurns = 12
	}
	// Default the sandbox to the workspace when unset.
	if len(c.Sandbox.AllowReadPaths) == 0 {
		c.Sandbox.AllowReadPaths = []string{c.Workspace}
	}
	if len(c.Sandbox.AllowWritePaths) == 0 {
		c.Sandbox.AllowWritePaths = []string{c.Workspace}
	}
	return &c, nil
}

// ModelByName returns the ModelEntry whose Name matches, or false.
func (c *Config) ModelByName(name string) (ModelEntry, bool) {
	for _, m := range c.ModelList {
		if m.Name == name {
			return m, true
		}
	}
	return ModelEntry{}, false
}

// Path returns the file this config was loaded from.
func (c *Config) Path() string { return c.path }

// Save writes the config back to its source path as indented JSON.
func (c *Config) Save() error {
	if c.path == "" {
		return fmt.Errorf("config has no path to save to")
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0o644)
}

// CredentialFor returns the API key for a protocol from the credentials map.
func (c *Config) CredentialFor(protocol string) string {
	if c.Credentials == nil {
		return ""
	}
	return c.Credentials[protocol]
}

// AllowRead reports whether path is inside an allowed read root.
func (s Sandbox) AllowRead(path string) bool { return pathAllowed(path, s.AllowReadPaths) }

// AllowWrite reports whether path is inside an allowed write root.
func (s Sandbox) AllowWrite(path string) bool { return pathAllowed(path, s.AllowWritePaths) }

func pathAllowed(path string, roots []string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	for _, root := range roots {
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(rootAbs, abs)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)) {
			return true
		}
	}
	return false
}
