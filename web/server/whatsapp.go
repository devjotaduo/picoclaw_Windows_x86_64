package server

import (
	"context"
	"net/http"
)

// handleWhatsAppStatus reports the connection state and the latest QR payload
// (when pairing) so the UI can render a QR code and poll for completion.
func (l *Launcher) handleWhatsAppStatus(w http.ResponseWriter, r *http.Request) {
	if l.wa == nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}
	state, qr, registered := l.wa.Status()
	writeJSON(w, http.StatusOK, map[string]any{
		"available":  true,
		"state":      state,
		"qr":         qr,
		"registered": registered,
	})
}

// handleWhatsAppConnect brings the WhatsApp socket up. Pairing must outlive the
// HTTP request, so it runs on a background context; the UI polls /status.
func (l *Launcher) handleWhatsAppConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	if l.wa == nil {
		writeErr(w, http.StatusServiceUnavailable, "whatsapp indisponível")
		return
	}
	if err := l.wa.Connect(context.Background()); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleWhatsAppLogout drops the paired session.
func (l *Launcher) handleWhatsAppLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	if l.wa == nil {
		writeErr(w, http.StatusServiceUnavailable, "whatsapp indisponível")
		return
	}
	if err := l.wa.Logout(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
