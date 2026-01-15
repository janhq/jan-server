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

	"jan-server/services/response-api/internal/config"
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
// It supports three auth methods in order of precedence:
// 1. Kong gateway headers (X-User-ID, X-Auth-Method) - trusted gateway auth
// 2. Bearer JWT token - direct JWT auth
// 3. X-API-Key header - for internal service-to-service calls (requires validation URL)
func (v *Validator) Middleware() gin.HandlerFunc {
	if v == nil || !v.cfg.AuthEnabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		// 1. Check for Kong gateway-injected headers (trusted)
		// These headers are set by Kong after validating auth
		if userID := strings.TrimSpace(c.GetHeader("X-User-ID")); userID != "" {
			authMethod := strings.TrimSpace(c.GetHeader("X-Auth-Method"))
			if authMethod != "" {
				// Gateway has already validated - trust the headers
				c.Set("user_id", userID)
				c.Set("auth_method", authMethod)
				c.Set("user_subject", strings.TrimSpace(c.GetHeader("X-User-Subject")))
				c.Set("user_username", strings.TrimSpace(c.GetHeader("X-User-Username")))
				c.Set("user_email", strings.TrimSpace(c.GetHeader("X-User-Email")))
				c.Next()
				return
			}
		}

		// 2. Try Bearer JWT token
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString != "" {
			token, err := jwt.Parse(tokenString, v.jwks.Keyfunc,
				jwt.WithIssuer(v.cfg.AuthIssuer),
				jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}),
			)
			if err == nil && token.Valid {
				claims, ok := token.Claims.(jwt.MapClaims)
				if ok {
					issuerClaim, _ := claims["iss"].(string)
					if issuer := strings.TrimSpace(v.cfg.AuthIssuer); issuer != "" {
						allowed := map[string]struct{}{
							issuer:                             {},
							"http://localhost:8085/realms/jan": {},
							"http://keycloak:8085/realms/jan":  {},
						}
						if _, exists := allowed[issuerClaim]; !exists {
							abortUnauthorized(c, "invalid token issuer")
							return
						}
					}

					if audience := strings.TrimSpace(v.cfg.Account); audience != "" {
						audClaim, hasAud := claims["aud"]
						if hasAud {
							switch aud := audClaim.(type) {
							case string:
								if aud != audience {
									abortUnauthorized(c, "invalid token audience")
									return
								}
							case []any:
								found := false
								for _, entry := range aud {
									if s, ok := entry.(string); ok && s == audience {
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
					return
				}
			}
		}

		// No valid auth found
		abortUnauthorized(c, "missing or invalid authentication")
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

func abortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": message,
	})
}
