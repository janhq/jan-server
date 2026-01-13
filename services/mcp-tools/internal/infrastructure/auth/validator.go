package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"

	"jan-server/services/mcp-tools/internal/infrastructure/config"
)

type Validator struct {
	cfg        *config.Config
	log        zerolog.Logger
	jwks       *keyfunc.JWKS
	httpClient *http.Client
}

func NewValidator(ctx context.Context, cfg *config.Config, log zerolog.Logger) (*Validator, error) {
	validator := &Validator{
		cfg: cfg,
		log: log,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	if !cfg.AuthEnabled {
		return validator, nil
	}

	options := keyfunc.Options{
		Ctx:               ctx,
		RefreshInterval:   time.Hour,
		RefreshUnknownKID: true,
		RefreshErrorHandler: func(err error) {
			log.Error().Err(err).Msg("jwks refresh error")
		},
	}
	jwks, err := keyfunc.Get(cfg.AuthJWKSURL, options)
	if err != nil {
		return nil, err
	}
	validator.jwks = jwks
	return validator, nil
}

func (v *Validator) Middleware() gin.HandlerFunc {
	if v == nil || !v.cfg.AuthEnabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		// Skip auth for health check and metrics endpoints
		path := c.Request.URL.Path
		if path == "/healthz" || path == "/readyz" || path == "/health/auth" || path == "/metrics" {
			c.Next()
			return
		}

		// Try X-API-Key first, then fall back to Authorization Bearer
		tokenString := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if tokenString == "" {
			tokenString = bearerToken(c.GetHeader("Authorization"))
		}
		if tokenString == "" {
			abortUnauthorized(c, "missing API key or bearer token")
			return
		}

		// Check if this is an API key (starts with sk_)
		if strings.HasPrefix(tokenString, "sk_") {
			// Validate API key via LLM-API
			userInfo, err := v.validateAPIKey(c.Request.Context(), tokenString)
			if err != nil {
				v.log.Warn().Err(err).Msg("API key validation failed")
				abortUnauthorized(c, "invalid API key")
				return
			}
			// Set user context from validated API key
			c.Set("user_id", userInfo.UserID)
			c.Set("api_key_validated", true)
			c.Next()
			return
		}

		// Otherwise, validate as JWT
		token, err := jwt.Parse(tokenString, v.jwks.Keyfunc,
			jwt.WithIssuer(v.cfg.AuthIssuer),
			jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}),
		)
		if err != nil || !token.Valid {
			abortUnauthorized(c, "invalid token")
			return
		}

		if !audienceMatches(token, v.cfg.Account) {
			abortUnauthorized(c, "invalid token")
			return
		}

		c.Set("auth_token", token)
		c.Next()
	}
}

// APIKeyUserInfo represents the response from LLM-API's validate-api-key endpoint
type APIKeyUserInfo struct {
	UserID   string `json:"user_id"`
	Subject  string `json:"subject"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// validateAPIKey validates an API key by calling LLM-API's validate-api-key endpoint
func (v *Validator) validateAPIKey(ctx context.Context, apiKey string) (*APIKeyUserInfo, error) {
	if v.cfg.LLMAPIBaseURL == "" {
		return nil, fmt.Errorf("LLM_API_BASE_URL not configured for API key validation")
	}

	endpoint := v.cfg.LLMAPIBaseURL + "/auth/validate-api-key"

	reqBody := map[string]string{"api_key": apiKey}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call LLM-API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM-API returned %d: %s", resp.StatusCode, string(body))
	}

	var userInfo APIKeyUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	v.log.Debug().
		Str("user_id", userInfo.UserID).
		Str("username", userInfo.Username).
		Msg("API key validated via LLM-API")

	return &userInfo, nil
}

func audienceMatches(token *jwt.Token, expected string) bool {
	if strings.TrimSpace(expected) == "" {
		return true
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}
	rawAud, ok := claims["aud"]
	if !ok {
		// Tokens without aud are accepted for backward compatibility.
		return true
	}
	switch aud := rawAud.(type) {
	case string:
		return strings.EqualFold(aud, expected)
	case []any:
		for _, entry := range aud {
			if s, ok := entry.(string); ok && strings.EqualFold(s, expected) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (v *Validator) Ready() bool {
	if v == nil || !v.cfg.AuthEnabled {
		return true
	}
	return v.jwks != nil
}

func bearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func abortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": message,
	})
}
