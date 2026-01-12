package auth

import (
	"context"
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
	cfg  *config.Config
	log  zerolog.Logger
	jwks *keyfunc.JWKS
}

func NewValidator(ctx context.Context, cfg *config.Config, log zerolog.Logger) (*Validator, error) {
	if !cfg.AuthEnabled {
		return &Validator{cfg: cfg, log: log}, nil
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
	return &Validator{
		cfg:  cfg,
		log:  log,
		jwks: jwks,
	}, nil
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
