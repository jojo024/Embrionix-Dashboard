package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/embrionix/dashboard/internal/models"
	"github.com/embrionix/dashboard/pkg/logger"
	"go.uber.org/zap"
)

// Notifier delivers alert events to an outbound webhook. The payload is
// Slack-compatible (a top-level "text" field) while also carrying the full
// structured event for generic consumers.
type Notifier struct {
	webhookURL string
	on         map[models.DeviceStatus]bool
	httpClient *http.Client
}

func NewNotifier(webhookURL string, on []string) *Notifier {
	set := make(map[models.DeviceStatus]bool, len(on))
	for _, s := range on {
		set[models.DeviceStatus(s)] = true
	}
	return &Notifier{
		webhookURL: webhookURL,
		on:         set,
		httpClient: &http.Client{Timeout: 8 * time.Second},
	}
}

// Enabled reports whether a webhook URL is configured.
func (n *Notifier) Enabled() bool { return n.webhookURL != "" }

// ShouldNotify reports whether a transition into toStatus should fire a webhook.
func (n *Notifier) ShouldNotify(toStatus models.DeviceStatus) bool {
	return n.Enabled() && n.on[toStatus]
}

// Notify posts the event to the configured webhook. It is safe to call in a
// goroutine; failures are logged, not returned.
func (n *Notifier) Notify(ctx context.Context, event models.AlertEvent) {
	if !n.Enabled() {
		return
	}

	text := fmt.Sprintf("[%s] %s: %s → %s — %s",
		event.ToStatus, event.DeviceName, event.FromStatus, event.ToStatus, event.Message)

	n.post(ctx, map[string]interface{}{"text": text, "event": event})
}

// NotifyText posts a plain text message to the webhook (used for scheduled
// reports). Ignored when no webhook is configured.
func (n *Notifier) NotifyText(ctx context.Context, text string) {
	if !n.Enabled() {
		return
	}
	n.post(ctx, map[string]interface{}{"text": text})
}

func (n *Notifier) post(ctx context.Context, payload map[string]interface{}) {
	body, err := json.Marshal(payload)
	if err != nil {
		logger.Error("notifier: marshal failed", zap.Error(err))
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(body))
	if err != nil {
		logger.Error("notifier: build request failed", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		logger.Warn("notifier: webhook delivery failed", zap.Error(err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Warn("notifier: webhook returned non-2xx", zap.Int("status", resp.StatusCode))
		return
	}
	logger.Debug("notifier: webhook delivered")
}
