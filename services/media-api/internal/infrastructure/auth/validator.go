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

	"jan-server/services/media-api/internal/config"
)

// Validator validates JWTs using JWKS.
type Validator struct {
	cfg        *config.Config
	log        zerolog.Logger
	jwks       *keyfunc.JWKS
	httpClient *http.Client
}

// NewValidator initializes JWKS fetching when auth is enabled.
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

// Middleware enforces JWT auth when enabled.
func (v *Validator) Middleware() gin.HandlerFunc {
	if v == nil || !v.cfg.AuthEnabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		tokenString := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if tokenString == "" {
			tokenString = bearerToken(c.GetHeader("Authorization"))
		}
		if tokenString == "" {
			abortUnauthorized(c, "missing API key or bearer token")
			return
		}

		if strings.HasPrefix(tokenString, "sk_") {
			userInfo, err := v.validateAPIKey(c.Request.Context(), tokenString)
			if err != nil {
				v.log.Warn().Err(err).Msg("API key validation failed")
				abortUnauthorized(c, "invalid API key")
				return
			}
			c.Set("user_id", userInfo.UserID)
			c.Set("api_key_validated", true)
			c.Next()
			return
		}

		opts := []jwt.ParserOption{
			jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}),
		}
		token, err := jwt.Parse(tokenString, v.jwks.Keyfunc, opts...)
		if err != nil || !token.Valid {
			abortUnauthorized(c, "invalid token")
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			abortUnauthorized(c, "invalid token claims")
			return
		}

		if issuer := strings.TrimSpace(v.cfg.AuthIssuer); issuer != "" {
			allowedIssuers := map[string]struct{}{}
			allowedIssuers[issuer] = struct{}{}
			allowedIssuers["http://localhost:8085/realms/jan"] = struct{}{}
			allowedIssuers["http://keycloak:8085/realms/jan"] = struct{}{}
			issuerClaim, _ := claims["iss"].(string)
			if _, ok := allowedIssuers[issuerClaim]; !ok {
				abortUnauthorized(c, "invalid token issuer")
				return
			}
		}

		if audience := strings.TrimSpace(v.cfg.Account); audience != "" {
			audClaim, ok := claims["aud"]
			if ok {
				switch aud := audClaim.(type) {
				case string:
					if aud != audience {
						abortUnauthorized(c, "invalid token audience")
						return
					}
				case []any:
					found := false
					for _, item := range aud {
						if s, isStr := item.(string); isStr && s == audience {
							found = true
							break
						}
					}
					if !found {
						abortUnauthorized(c, "invalid token audience")
						return
					}
				default:
					abortUnauthorized(c, "invalid token audience")
					return
				}
			}
		}

		c.Set("auth_token", token)
		c.Next()
	}
}

// Ready indicates if the validator is prepared.
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

type APIKeyUserInfo struct {
	UserID   string `json:"user_id"`
	Subject  string `json:"subject"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func (v *Validator) validateAPIKey(ctx context.Context, apiKey string) (*APIKeyUserInfo, error) {
	if strings.TrimSpace(v.cfg.LLMAPIBaseURL) == "" {
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

func abortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": message,
	})
}
