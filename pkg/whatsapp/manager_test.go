package whatsapp

import (
	"context"
	"path/filepath"
	"testing"
)

// TestNewOpensStore validates that the session store opens and migrates on the
// pure-Go SQLite driver (modernc) and that a client is built from an empty
// device, reporting the expected initial status.
func TestNewOpensStore(t *testing.T) {
	dir := t.TempDir()
	m, err := New(context.Background(), filepath.Join(dir, "wa.db"), func(context.Context, string) string { return "" })
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer m.Close()

	state, qr, registered := m.Status()
	if state != StateDisconnected {
		t.Errorf("state = %q, want %q", state, StateDisconnected)
	}
	if qr != "" {
		t.Errorf("qr = %q, want empty", qr)
	}
	if registered {
		t.Errorf("registered = true, want false (no paired session)")
	}
}
