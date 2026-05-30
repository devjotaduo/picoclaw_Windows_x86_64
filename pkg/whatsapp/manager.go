// Package whatsapp connects a single WhatsApp number to the agent using
// whatsmeow (the unofficial WhatsApp Web multidevice protocol). Incoming
// direct text messages are answered by the agent and the reply is sent back.
// The session is persisted in a SQLite file (pure-Go modernc driver, so the
// build stays CGO-free) on the data volume, surviving restarts/redeploys.
package whatsapp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "modernc.org/sqlite"
)

// Connection states reported to the UI.
const (
	StateDisconnected = "disconnected"
	StateQR           = "qr"
	StateConnected    = "connected"
)

// ReplyFunc produces the agent's answer to an incoming message. Returning an
// empty string stays silent.
type ReplyFunc func(ctx context.Context, text string) string

// Manager owns one WhatsApp connection and its lifecycle.
type Manager struct {
	log       waLog.Logger
	reply     ReplyFunc
	container *sqlstore.Container

	mu     sync.Mutex
	client *whatsmeow.Client
	qr     string // latest QR code payload, empty once paired
	state  string
}

// New opens (or creates) the session store at dbPath and builds a client from
// any existing device. It does not connect — call Connect.
func New(ctx context.Context, dbPath string, reply ReplyFunc) (*Manager, error) {
	log := waLog.Noop
	dsn := "file:" + dbPath + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	container, err := sqlstore.New(ctx, "sqlite", dsn, log)
	if err != nil {
		return nil, fmt.Errorf("whatsapp store: %w", err)
	}
	dev, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("whatsapp device: %w", err)
	}
	m := &Manager{log: log, reply: reply, container: container, state: StateDisconnected}
	m.client = whatsmeow.NewClient(dev, log)
	m.client.AddEventHandler(m.onEvent)
	return m, nil
}

// Status returns the current state, the latest QR payload (when state is "qr"),
// and whether a paired session already exists.
func (m *Manager) Status() (state, qr string, registered bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state, m.qr, m.client.Store.ID != nil
}

// Connect brings the socket up. With no paired session it starts QR pairing
// (codes surface via Status); with an existing session it simply reconnects.
func (m *Manager) Connect(ctx context.Context) error {
	m.mu.Lock()
	client := m.client
	m.mu.Unlock()

	if client.IsConnected() {
		return nil
	}

	if client.Store.ID == nil {
		// Fresh login: the QR channel must be requested before connecting.
		qrChan, err := client.GetQRChannel(ctx)
		if err != nil {
			return fmt.Errorf("qr channel: %w", err)
		}
		if err := client.Connect(); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		m.set(StateQR, "")
		go func() {
			for item := range qrChan {
				switch item.Event {
				case whatsmeow.QRChannelEventCode:
					m.set(StateQR, item.Code)
				case "success":
					m.set(StateConnected, "")
				default:
					m.set(StateDisconnected, "")
				}
			}
		}()
		return nil
	}

	// Existing session — just reconnect.
	if err := client.Connect(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	return nil
}

// Logout drops the paired session (the user must scan a new QR to reconnect).
func (m *Manager) Logout(ctx context.Context) error {
	m.mu.Lock()
	client := m.client
	m.mu.Unlock()
	if client.Store.ID != nil {
		if err := client.Logout(ctx); err != nil {
			return err
		}
	} else {
		client.Disconnect()
	}
	m.set(StateDisconnected, "")
	return nil
}

// Close disconnects the socket and closes the session store (for graceful
// shutdown); the persisted session is kept on disk.
func (m *Manager) Close() {
	m.mu.Lock()
	client, container := m.client, m.container
	m.mu.Unlock()
	if client != nil {
		client.Disconnect()
	}
	if container != nil {
		_ = container.Close()
	}
}

func (m *Manager) set(state, qr string) {
	m.mu.Lock()
	m.state, m.qr = state, qr
	m.mu.Unlock()
}

func (m *Manager) onEvent(evt any) {
	switch v := evt.(type) {
	case *events.Connected:
		m.set(StateConnected, "")
	case *events.LoggedOut:
		m.set(StateDisconnected, "")
	case *events.Message:
		m.onMessage(v)
	}
}

func (m *Manager) onMessage(v *events.Message) {
	// Only answer direct (non-group) text messages from other people.
	if v.Info.IsFromMe || v.Info.Chat.Server == types.GroupServer {
		return
	}
	text := v.Message.GetConversation()
	if text == "" {
		if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			text = ext.GetText()
		}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	chat := v.Info.Chat
	go func() {
		ans := strings.TrimSpace(m.reply(context.Background(), text))
		if ans == "" {
			return
		}
		m.mu.Lock()
		client := m.client
		m.mu.Unlock()
		_, _ = client.SendMessage(context.Background(), chat, &waE2E.Message{
			Conversation: proto.String(ans),
		})
	}()
}
