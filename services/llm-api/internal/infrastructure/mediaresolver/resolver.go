package mediaresolver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog"
	openai "github.com/sashabaranov/go-openai"

	"jan-server/services/llm-api/internal/config"
)

var placeholderPattern = regexp.MustCompile(`data:(image/[a-z0-9.+-]+);(jan_[A-Za-z0-9]+)`)

// Resolver resolves jan_* media placeholders embedded in chat messages.
type Resolver interface {
	ResolveMessages(ctx context.Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, bool, error)
}

type httpResolver struct {
	endpoint   string
	serviceKey string
	client     *http.Client
	log        zerolog.Logger
}

// NewResolver constructs an HTTP-backed resolver. Returns nil when MEDIA_RESOLVE_URL is empty.
func NewResolver(cfg *config.Config, log zerolog.Logger) Resolver {
	if cfg == nil {
		return nil
	}

	endpoint := strings.TrimSpace(cfg.MediaResolveURL)
	if endpoint == "" {
		return nil
	}

	timeout := cfg.MediaResolveTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &httpResolver{
		endpoint:   endpoint,
		serviceKey: strings.TrimSpace(cfg.MediaServiceKey),
		client: &http.Client{
			Timeout: timeout,
		},
		log: log.With().Str("component", "media-resolver").Logger(),
	}
}

func (r *httpResolver) ResolveMessages(ctx context.Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, bool, error) {
	if !r.hasPlaceholder(messages) {
		return messages, false, nil
	}

	requestBody := map[string]interface{}{
		"payload": map[string]interface{}{
			"messages": messages,
		},
	}

	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(requestBody); err != nil {
		return messages, false, fmt.Errorf("encode media resolve request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, buf)
	if err != nil {
		return messages, false, fmt.Errorf("build media resolve request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if r.serviceKey != "" {
		req.Header.Set("X-Media-Service-Key", r.serviceKey)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return messages, false, fmt.Errorf("call media resolve endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var body bytes.Buffer
		_, _ = body.ReadFrom(resp.Body)
		return messages, false, fmt.Errorf("media resolve error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(body.String()))
	}

	var envelope struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return messages, false, fmt.Errorf("decode media resolve response: %w", err)
	}
	if len(envelope.Payload) == 0 {
		return messages, false, errors.New("media resolve returned empty payload")
	}

	var output struct {
		Messages []openai.ChatCompletionMessage `json:"messages"`
	}
	if err := json.Unmarshal(envelope.Payload, &output); err != nil {
		return messages, false, fmt.Errorf("decode resolved messages: %w", err)
	}
	if len(output.Messages) == 0 {
		return messages, false, errors.New("media resolve returned no messages")
	}

	r.log.Debug().
		Int("message_count", len(output.Messages)).
		Msg("resolved media placeholders in chat request")

	return output.Messages, true, nil
}

func (r *httpResolver) hasPlaceholder(messages []openai.ChatCompletionMessage) bool {
	for _, msg := range messages {
		if matchesPlaceholder(msg.Content) {
			return true
		}
		for _, part := range msg.MultiContent {
			switch part.Type {
			case openai.ChatMessagePartTypeImageURL:
				if part.ImageURL != nil && matchesPlaceholder(part.ImageURL.URL) {
					return true
				}
			case openai.ChatMessagePartTypeText:
				if matchesPlaceholder(part.Text) {
					return true
				}
			}
		}
	}
	return false
}

func matchesPlaceholder(value string) bool {
	if value == "" {
		return false
	}
	return placeholderPattern.MatchString(value)
}
