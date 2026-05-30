package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Telegram is a Bot API channel using long polling (getUpdates). No external
// dependencies: it speaks the HTTP Bot API directly.
type Telegram struct {
	token  string
	client *http.Client
}

// NewTelegram builds a Telegram channel for the given bot token.
func NewTelegram(token string) *Telegram {
	return &Telegram{
		token:  token,
		client: &http.Client{Timeout: 65 * time.Second},
	}
}

func (t *Telegram) Name() string { return "telegram" }

func (t *Telegram) api(method string) string {
	return "https://api.telegram.org/bot" + t.token + "/" + method
}

type tgUpdate struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		Text string `json:"text"`
		From struct {
			ID        int    `json:"id"`
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
		} `json:"from"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
}

type tgUpdatesResp struct {
	OK     bool       `json:"ok"`
	Result []tgUpdate `json:"result"`
}

// Run polls for updates and dispatches each text message to handle.
func (t *Telegram) Run(ctx context.Context, handle Handler) error {
	offset := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		updates, err := t.getUpdates(ctx, offset)
		if err != nil {
			// Transient: back off briefly and retry.
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(3 * time.Second):
			}
			continue
		}

		for _, u := range updates {
			offset = u.UpdateID + 1
			if u.Message == nil || strings.TrimSpace(u.Message.Text) == "" {
				continue
			}
			user := u.Message.From.Username
			if user == "" {
				user = u.Message.From.FirstName
			}
			reply, err := handle(ctx, user, u.Message.Text)
			if err != nil {
				reply = "error: " + err.Error()
			}
			if reply != "" {
				_ = t.sendMessage(ctx, u.Message.Chat.ID, reply)
			}
		}
	}
}

func (t *Telegram) getUpdates(ctx context.Context, offset int) ([]tgUpdate, error) {
	q := url.Values{}
	q.Set("timeout", "50")
	q.Set("offset", strconv.Itoa(offset))
	reqURL := t.api("getUpdates") + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed tgUpdatesResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if !parsed.OK {
		return nil, fmt.Errorf("telegram getUpdates not ok")
	}
	return parsed.Result, nil
}

func (t *Telegram) sendMessage(ctx context.Context, chatID int64, text string) error {
	q := url.Values{}
	q.Set("chat_id", strconv.FormatInt(chatID, 10))
	q.Set("text", text)
	reqURL := t.api("sendMessage") + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
