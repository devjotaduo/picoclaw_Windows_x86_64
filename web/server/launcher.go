// Package server implements the PicoClaw Web UI Launcher: an HTTP server
// (default localhost:18800) that serves the embedded React frontend and a REST
// + SSE API backed by the agent. It guards access with a single password set
// on first run (launcher-setup) and a signed session cookie.
package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"picoclaw/pkg/agent"
	"picoclaw/pkg/config"
)

const cookieName = "picoclaw_session"

// Launcher is the Web UI HTTP server.
type Launcher struct {
	addr   string
	public bool

	// corsOrigins, when set (PICOCLAW_CORS_ORIGIN, comma-separated), enables
	// credentialed CORS for those exact origins and switches the session cookie
	// to SameSite=None;Secure so a frontend hosted on a different domain
	// (e.g. Vercel or a custom domain) can authenticate. Multiple origins let a
	// deployment migrate domains without downtime.
	corsOrigins []string
	crossSite   bool

	mu      sync.RWMutex
	cfg     *config.Config
	agent   *agent.Agent
	auth    authStore
	secret  []byte
	authDir string
}

// authStore is the persisted password record.
type authStore struct {
	Salt string `json:"salt"` // hex
	Hash string `json:"hash"` // hex sha256(salt+password)
}

func (a authStore) configured() bool { return a.Hash != "" }

// New builds a Launcher. addr is host:port; public exposes it beyond loopback.
func New(addr string, public bool, cfg *config.Config) (*Launcher, error) {
	ag, err := agent.New(cfg)
	if err != nil {
		// The UI must still load so the user can fix credentials; defer agent
		// errors to chat time rather than failing startup.
		ag = nil
	}
	authDir := cfg.Workspace
	corsOrigins := parseOrigins(os.Getenv("PICOCLAW_CORS_ORIGIN"))
	l := &Launcher{
		addr:        addr,
		public:      public,
		corsOrigins: corsOrigins,
		crossSite:   len(corsOrigins) > 0,
		cfg:         cfg,
		agent:       ag,
		authDir:     authDir,
	}
	if err := l.loadAuth(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *Launcher) authPath() string { return filepath.Join(l.authDir, ".launcher.json") }

func (l *Launcher) loadAuth() error {
	if err := os.MkdirAll(l.authDir, 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(l.authPath())
	if err == nil {
		_ = json.Unmarshal(data, &l.auth)
	}
	// A secret for signing cookies. In production set PICOCLAW_SECRET so
	// sessions survive restarts/redeploys; otherwise a random per-process
	// secret is used and rotating it on restart invalidates existing sessions.
	if s := os.Getenv("PICOCLAW_SECRET"); s != "" {
		l.secret = []byte(s)
		return nil
	}
	l.secret = make([]byte, 32)
	if _, err := rand.Read(l.secret); err != nil {
		return err
	}
	return nil
}

func (l *Launcher) saveAuth() error {
	data, err := json.MarshalIndent(l.auth, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.authPath(), data, 0o600)
}

// hashPassword returns hex sha256(salt || password).
func hashPassword(salt, password string) string {
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
}

// Run starts the HTTP server until ctx is cancelled.
func (l *Launcher) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	l.routes(mux)

	srv := &http.Server{
		Addr:         l.addr,
		Handler:      l.withSecurity(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 10 * time.Minute,
	}
	go func() {
		<-ctx.Done()
		sh, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sh)
	}()

	// Start the cron scheduler for the current agent, if any.
	if l.agent != nil {
		go func() { _ = l.agent.Scheduler().Run(ctx) }()
	}

	scheme := "http"
	log.Printf("web launcher on %s://%s (public=%v)", scheme, l.addr, l.public)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// withSecurity sets basic headers and, when PICOCLAW_CORS_ORIGIN is set,
// credentialed CORS for that origin (handling preflight requests).
func (l *Launcher) withSecurity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if origin := l.allowedOrigin(r.Header.Get("Origin")); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// parseOrigins splits PICOCLAW_CORS_ORIGIN (comma-separated) into trimmed,
// non-empty origins.
func parseOrigins(v string) []string {
	var out []string
	for _, o := range strings.Split(v, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}

// allowedOrigin returns the Access-Control-Allow-Origin value for a request
// carrying the given Origin header: the request origin when it is one of the
// configured origins, otherwise the first configured origin (a deterministic
// fallback). It returns "" when no origins are configured (CORS disabled).
func (l *Launcher) allowedOrigin(reqOrigin string) string {
	if len(l.corsOrigins) == 0 {
		return ""
	}
	for _, o := range l.corsOrigins {
		if o == reqOrigin {
			return o
		}
	}
	return l.corsOrigins[0]
}

func (l *Launcher) routes(mux *http.ServeMux) {
	// Public auth endpoints.
	mux.HandleFunc("/api/launcher/status", l.handleAuthStatus)
	mux.HandleFunc("/api/launcher/setup", l.handleSetup)
	mux.HandleFunc("/api/launcher/login", l.handleLogin)
	mux.HandleFunc("/api/launcher/logout", l.handleLogout)

	// Protected API.
	mux.HandleFunc("/api/system", l.requireAuth(l.handleSystem))
	mux.HandleFunc("/api/models", l.requireAuth(l.handleModels))
	mux.HandleFunc("/api/models/", l.requireAuth(l.handleModelByName))
	mux.HandleFunc("/api/credentials", l.requireAuth(l.handleCredentials))
	mux.HandleFunc("/api/credentials/", l.requireAuth(l.handleCredentialByName))
	mux.HandleFunc("/api/agents", l.requireAuth(l.handleAgents))
	mux.HandleFunc("/api/agents/", l.requireAuth(l.handleAgentByName))
	mux.HandleFunc("/api/chat/stream", l.requireAuth(l.handleChatStream))

	// Static SPA (catch-all).
	mux.HandleFunc("/", l.serveUI(uiFS()))
}

// --- auth handlers ---

func (l *Launcher) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	l.mu.RLock()
	configured := l.auth.configured()
	l.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"needs_setup": !configured,
		"authed":      l.isAuthed(r),
	})
}

func (l *Launcher) handleSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.auth.configured() {
		writeErr(w, http.StatusConflict, "already set up")
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Password) < 4 {
		writeErr(w, http.StatusBadRequest, "password must be at least 4 chars")
		return
	}
	salt := randHex(16)
	l.auth = authStore{Salt: salt, Hash: hashPassword(salt, body.Password)}
	if err := l.saveAuth(); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	l.setSession(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (l *Launcher) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "bad request")
		return
	}
	l.mu.RLock()
	a := l.auth
	l.mu.RUnlock()
	if !a.configured() {
		writeErr(w, http.StatusConflict, "not set up")
		return
	}
	want := hashPassword(a.Salt, body.Password)
	if subtle.ConstantTimeCompare([]byte(want), []byte(a.Hash)) != 1 {
		writeErr(w, http.StatusUnauthorized, "wrong password")
		return
	}
	l.setSession(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (l *Launcher) handleLogout(w http.ResponseWriter, _ *http.Request) {
	clear := &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	if l.crossSite {
		clear.SameSite = http.SameSiteNoneMode
		clear.Secure = true
	}
	http.SetCookie(w, clear)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// --- session cookie ---

func (l *Launcher) setSession(w http.ResponseWriter) {
	token := l.signToken("authed")
	c := &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600,
	}
	if l.crossSite {
		// Cross-origin frontend (e.g. Vercel) requires SameSite=None;Secure.
		c.SameSite = http.SameSiteNoneMode
		c.Secure = true
	}
	http.SetCookie(w, c)
}

func (l *Launcher) signToken(payload string) string {
	mac := hmac.New(sha256.New, l.secret)
	mac.Write([]byte(payload))
	return payload + "." + hex.EncodeToString(mac.Sum(nil))
}

func (l *Launcher) isAuthed(r *http.Request) bool {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	const prefix = "authed."
	if len(c.Value) <= len(prefix) || c.Value[:len(prefix)] != prefix {
		return false
	}
	expected := l.signToken("authed")
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(expected)) == 1
}

func (l *Launcher) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !l.isAuthed(r) {
			writeErr(w, http.StatusUnauthorized, "not authenticated")
			return
		}
		next(w, r)
	}
}

// --- static UI ---

func (l *Launcher) serveUI(fsys fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(fsys))
	return func(w http.ResponseWriter, r *http.Request) {
		// Serve the asset if it exists; otherwise fall back to index.html so
		// client-side routes (TanStack Router) resolve.
		p := r.URL.Path
		if p == "/" {
			p = "/index.html"
		}
		if f, err := fsys.Open(trimLeadingSlash(p)); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		serveIndex(w, fsys)
	}
}

func serveIndex(w http.ResponseWriter, fsys fs.FS) {
	f, err := fsys.Open("index.html")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "ui not built")
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, f)
}

func trimLeadingSlash(s string) string {
	if len(s) > 0 && s[0] == '/' {
		return s[1:]
	}
	return s
}

// --- helpers ---

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
