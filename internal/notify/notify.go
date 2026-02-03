package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rowjay/db-backup-utility/internal/config"
)

type Event struct {
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	Database  string    `json:"database"`
	DBType    string    `json:"db_type"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Duration  string    `json:"duration"`
	Key       string    `json:"key"`
	Error     string    `json:"error,omitempty"`
}

type Notifier interface {
	Notify(ctx context.Context, event Event) error
}

type Multi struct {
	Targets []Notifier
}

func (m Multi) Notify(ctx context.Context, event Event) error {
	var err error
	for _, target := range m.Targets {
		if target == nil {
			continue
		}
		if nerr := target.Notify(ctx, event); nerr != nil {
			err = nerr
		}
	}
	return err
}

type Webhook struct {
	Name    string
	URL     string
	Headers map[string]string
}

func (w Webhook) Notify(ctx context.Context, event Event) error {
	body, _ := json.Marshal(event)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.Headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook %s returned %s", w.Name, resp.Status)
	}
	return nil
}

type Mattermost struct {
	Name string
	URL  string
}

func (m Mattermost) Notify(ctx context.Context, event Event) error {
	payload := map[string]string{"text": fmt.Sprintf("[%s] %s", event.Status, event.Message)}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("mattermost %s returned %s", m.Name, resp.Status)
	}
	return nil
}

type Matrix struct {
	Name        string
	ServerURL   string
	AccessToken string
	RoomID      string
}

func (m Matrix) Notify(ctx context.Context, event Event) error {
	endpoint := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/send/m.room.message/%d?access_token=%s", m.ServerURL, m.RoomID, time.Now().UnixNano(), m.AccessToken)
	payload := map[string]any{
		"msgtype": "m.text",
		"body":    fmt.Sprintf("[%s] %s", event.Status, event.Message),
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("matrix %s returned %s", m.Name, resp.Status)
	}
	return nil
}

func FromConfig(cfg config.NotificationsConfig) Multi {
	var targets []Notifier
	for _, w := range cfg.Webhooks {
		targets = append(targets, Webhook{Name: w.Name, URL: w.URL, Headers: w.Headers})
	}
	for _, mm := range cfg.Mattermost {
		targets = append(targets, Mattermost{Name: mm.Name, URL: mm.URL})
	}
	for _, mx := range cfg.Matrix {
		targets = append(targets, Matrix{Name: mx.Name, ServerURL: mx.ServerURL, AccessToken: mx.AccessToken, RoomID: mx.RoomID})
	}
	return Multi{Targets: targets}
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}
