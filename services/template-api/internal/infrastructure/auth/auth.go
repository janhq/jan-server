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

	"jan-server/services/template-api/internal/config"
)

// Validator validates JWTs using JWKS.
type Validator struct {
	cfg  *config.Config
	log  zerolog.Logger
	jwks *keyfunc.JWKS
}

// NewValidator initializes JWKS fetching when auth is enabled.
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

// Middleware enforces JWT auth when enabled.
func (v *Validator) Middleware() gin.HandlerFunc {
	if v == nil || !v.cfg.AuthEnabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			abortUnauthorized(c, "missing bearer token")
			return
		}

		token, err := jwt.Parse(tokenString, v.jwks.Keyfunc,
			jwt.WithAudience(v.cfg.AuthAudience),
			jwt.WithIssuer(v.cfg.AuthIssuer),
			jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}),
		)
		if err != nil || !token.Valid {
			abortUnauthorized(c, "invalid token")
			return
		}

		c.Set("auth_token", token)
		c.Next()
	}
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
