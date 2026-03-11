package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"osg/internal/config"
)

const httpTimeout = 10 * time.Second

// Dispatch sends a webhook event to all configured endpoints that subscribe
// to the given event name. Payload is serialized as JSON in the request body.
// Non-nil errors from individual webhooks are logged but do not stop other
// deliveries; the returned error is nil unless no webhooks could be sent.
func Dispatch(ctx context.Context, cfg config.Config, event string, payload map[string]any, logger *slog.Logger) {
	if len(cfg.Webhooks) == 0 {
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		if logger != nil {
			logger.Warn("webhook marshal failed", "event", event, "error", err)
		}
		return
	}

	client := &http.Client{Timeout: httpTimeout}

	for _, wh := range cfg.Webhooks {
		if !matchesEvent(wh.Events, event) {
			continue
		}
		if err := send(ctx, client, wh, event, body); err != nil {
			if logger != nil {
				logger.Warn("webhook delivery failed", "url", wh.URL, "event", event, "error", err)
			}
		} else if logger != nil {
			logger.Debug("webhook delivered", "url", wh.URL, "event", event)
		}
	}
}

func matchesEvent(subscribed []string, event string) bool {
	if len(subscribed) == 0 {
		return true // empty list means all events
	}
	for _, e := range subscribed {
		if e == event {
			return true
		}
	}
	return false
}

func send(ctx context.Context, client *http.Client, wh config.WebhookConfig, event string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OSG-Event", event)

	if wh.Secret != "" {
		mac := hmac.New(sha256.New, []byte(wh.Secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-OSG-Signature", "sha256="+sig)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}
