package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// HTTPService implements webhook notifications via HTTP POST.
type HTTPService struct {
	httpClient *http.Client
	log        zerolog.Logger
	maxRetries int
	retryDelay time.Duration
}

// NewHTTPService creates a new HTTP-based webhook service.
func NewHTTPService(log zerolog.Logger) *HTTPService {
	return &HTTPService{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		log:        log.With().Str("component", "webhook").Logger(),
		maxRetries: 3,
		retryDelay: 2 * time.Second,
	}
}

// NotifyCompleted sends a webhook notification when a response completes.
func (s *HTTPService) NotifyCompleted(ctx context.Context, responseID string, output interface{}, metadata map[string]interface{}, completedAt *time.Time) error {
	webhookURL := extractWebhookURL(metadata)
	if webhookURL == "" {
		s.log.Debug().Str("response_id", responseID).Msg("no webhook URL configured, skipping notification")
		return nil
	}

	payload := WebhookPayload{
		ID:          responseID,
		Event:       "response.completed",
		Status:      "completed",
		Output:      output,
		Metadata:    metadata,
		CompletedAt: formatTime(completedAt),
	}

	return s.sendWebhook(ctx, webhookURL, payload, responseID)
}

// NotifyFailed sends a webhook notification when a response fails.
func (s *HTTPService) NotifyFailed(ctx context.Context, responseID string, errorCode string, errorMessage string, metadata map[string]interface{}) error {
	webhookURL := extractWebhookURL(metadata)
	if webhookURL == "" {
		s.log.Debug().Str("response_id", responseID).Msg("no webhook URL configured, skipping notification")
		return nil
	}

	payload := WebhookPayload{
		ID:       responseID,
		Event:    "response.failed",
		Status:   "failed",
		Error:    &ErrorDetails{Code: errorCode, Message: errorMessage},
		Metadata: metadata,
	}

	return s.sendWebhook(ctx, webhookURL, payload, responseID)
}

func (s *HTTPService) sendWebhook(ctx context.Context, url string, payload WebhookPayload, responseID string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= s.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("create webhook request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "jan-response-api/1.0")
		req.Header.Set("X-Jan-Event", payload.Event)
		req.Header.Set("X-Jan-Response-ID", responseID)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("send webhook (attempt %d/%d): %w", attempt, s.maxRetries, err)
			s.log.Warn().Err(err).Str("url", url).Int("attempt", attempt).Msg("webhook delivery failed")

			if attempt < s.maxRetries {
				time.Sleep(s.retryDelay)
				continue
			}
			break
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			s.log.Info().Str("url", url).Int("status", resp.StatusCode).Str("response_id", responseID).Msg("webhook delivered successfully")
			return nil
		}

		lastErr = fmt.Errorf("webhook returned status %d (attempt %d/%d)", resp.StatusCode, attempt, s.maxRetries)
		s.log.Warn().Int("status", resp.StatusCode).Str("url", url).Int("attempt", attempt).Msg("webhook delivery failed")

		if attempt < s.maxRetries {
			time.Sleep(s.retryDelay)
		}
	}

	return lastErr
}

func extractWebhookURL(metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}
	if url, ok := metadata["webhook_url"].(string); ok {
		return url
	}
	if url, ok := metadata["webhookUrl"].(string); ok {
		return url
	}
	return ""
}

func formatTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format(time.RFC3339)
	return &formatted
}
