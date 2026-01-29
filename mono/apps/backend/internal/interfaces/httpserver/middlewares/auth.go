package middlewares

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"jan-server/mono/apps/backend/internal/domain/user"
	"jan-server/mono/apps/backend/internal/infrastructure/config"
	"jan-server/mono/apps/backend/internal/infrastructure/database/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// Helper functions for creating service instances
func newUserRepository(db *gorm.DB) user.Repository {
	return repository.NewUserRepository(db)
}

func newUserService(repo user.Repository, cfg *config.Config) *user.Service {
	return user.NewService(repo, user.ServiceConfig{
		JWTSecret:        cfg.LocalJWTSecret,
		JWTIssuer:        cfg.LocalJWTIssuer,
		JWTExpiration:    cfg.LocalJWTExpiration,
		RefreshTokenTTL:  cfg.LocalJWTRefreshTTL,
		BcryptCost:       cfg.BcryptCost,
		APIKeyPrefix:     cfg.APIKeyPrefix,
		APIKeyMaxPerUser: cfg.APIKeyMaxPerUser,
		APIKeyDefaultTTL: cfg.APIKeyDefaultTTL,
	})
}

const (
	// Context keys
	PrincipalKey = "principal"

	// Header keys
	AuthorizationHeader = "Authorization"
	APIKeyHeader        = "X-API-Key"

	// Kong gateway headers (when Kong is enabled)
	KongUserIDHeader     = "X-User-ID"
	KongUserSubject      = "X-User-Subject"
	KongUserEmail        = "X-User-Email"
	KongUserUsername     = "X-User-Username"
	KongAuthMethod       = "X-Auth-Method"
)

// AuthMethod represents the authentication method used.
type AuthMethod string

const (
	AuthMethodJWT       AuthMethod = "jwt"
	AuthMethodAPIKey    AuthMethod = "apikey"
	AuthMethodLocalJWT  AuthMethod = "local_jwt"
	AuthMethodKong      AuthMethod = "kong"
)

// Principal represents the authenticated user.
type Principal struct {
	ID           string            `json:"id"`
	Subject      string            `json:"subject"`
	Email        string            `json:"email"`
	Username     string            `json:"username"`
	Name         string            `json:"name"`
	Roles        []string          `json:"roles"`
	Groups       []string          `json:"groups"`
	AuthMethod   AuthMethod        `json:"auth_method"`
	AuthProvider string            `json:"auth_provider"` // "keycloak" or "local"
	Issuer       string            `json:"issuer"`
	Attributes   map[string]any    `json:"attributes,omitempty"`
}

// Auth middleware validates authentication tokens.
// Supports: Keycloak JWT, Local JWT, API Keys, and Kong gateway headers.
func Auth(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var principal *Principal
		var err error

		// 1. Check Kong gateway headers first (if Kong is enabled and headers present)
		if cfg.KongEnabled && c.GetHeader(KongAuthMethod) != "" {
			principal, err = extractFromKongHeaders(c)
			if err == nil {
				setPrincipal(c, principal)
				c.Next()
				return
			}
		}

		// 2. Check API Key header
		apiKey := c.GetHeader(APIKeyHeader)
		if apiKey != "" {
			principal, err = validateAPIKey(c.Request.Context(), cfg, db, apiKey)
			if err == nil {
				setPrincipal(c, principal)
				c.Next()
				return
			}
		}

		// 3. Check Authorization header (Bearer token)
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader != "" {
			// Check for Bearer token
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")

				// Check if it's an API key (starts with sk_)
				if strings.HasPrefix(token, "sk_") {
					principal, err = validateAPIKey(c.Request.Context(), cfg, db, token)
					if err == nil {
						setPrincipal(c, principal)
						c.Next()
						return
					}
				} else {
					// Try to validate as JWT
					principal, err = validateJWT(c.Request.Context(), cfg, db, token)
					if err == nil {
						setPrincipal(c, principal)
						c.Next()
						return
					}
				}
			}
		}

		// No valid authentication found
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
			"message": "valid authentication required",
		})
	}
}

// RequireAdmin middleware ensures the user has admin role.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := GetPrincipal(c)
		if principal == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			return
		}

		// Check for admin role
		for _, role := range principal.Roles {
			if role == "admin" {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "forbidden",
			"message": "admin role required",
		})
	}
}

// GetPrincipal retrieves the authenticated principal from the context.
func GetPrincipal(c *gin.Context) *Principal {
	if p, exists := c.Get(PrincipalKey); exists {
		if principal, ok := p.(*Principal); ok {
			return principal
		}
	}
	return nil
}

func setPrincipal(c *gin.Context, p *Principal) {
	c.Set(PrincipalKey, p)
}

func extractFromKongHeaders(c *gin.Context) (*Principal, error) {
	userID := c.GetHeader(KongUserIDHeader)
	if userID == "" {
		return nil, errors.New("missing X-User-ID header")
	}

	authMethod := AuthMethod(c.GetHeader(KongAuthMethod))
	if authMethod == "" {
		authMethod = AuthMethodKong
	}

	return &Principal{
		ID:           userID,
		Subject:      c.GetHeader(KongUserSubject),
		Email:        c.GetHeader(KongUserEmail),
		Username:     c.GetHeader(KongUserUsername),
		AuthMethod:   authMethod,
		AuthProvider: "keycloak", // Kong validates Keycloak tokens
	}, nil
}

func validateJWT(ctx context.Context, cfg *config.Config, db *gorm.DB, tokenString string) (*Principal, error) {
	// Try local JWT validation first (if enabled)
	if cfg.LocalAuthEnabled {
		principal, err := validateLocalJWT(cfg, tokenString)
		if err == nil {
			return principal, nil
		}
	}

	// Try Keycloak JWT validation (if enabled)
	if cfg.KeycloakEnabled {
		principal, err := validateKeycloakJWT(ctx, cfg, tokenString)
		if err == nil {
			return principal, nil
		}
	}

	return nil, errors.New("invalid token")
}

func validateLocalJWT(cfg *config.Config, tokenString string) (*Principal, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(cfg.LocalJWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Check issuer
	iss, _ := claims["iss"].(string)
	if iss != cfg.LocalJWTIssuer {
		return nil, errors.New("invalid issuer")
	}

	// Extract principal from claims
	principal := &Principal{
		ID:           getStringClaim(claims, "sub"),
		Subject:      getStringClaim(claims, "sub"),
		Email:        getStringClaim(claims, "email"),
		Username:     getStringClaim(claims, "preferred_username"),
		Name:         getStringClaim(claims, "name"),
		Issuer:       iss,
		AuthMethod:   AuthMethodLocalJWT,
		AuthProvider: "local",
	}

	// Extract roles
	if roles, ok := claims["roles"].([]interface{}); ok {
		for _, r := range roles {
			if role, ok := r.(string); ok {
				principal.Roles = append(principal.Roles, role)
			}
		}
	}

	return principal, nil
}

func validateKeycloakJWT(ctx context.Context, cfg *config.Config, tokenString string) (*Principal, error) {
	// TODO: Implement Keycloak JWT validation using JWKS
	// This requires fetching the JWKS from Keycloak and validating the signature
	// For now, return an error to indicate not implemented
	return nil, errors.New("keycloak jwt validation not yet implemented")
}

func validateAPIKey(ctx context.Context, cfg *config.Config, db *gorm.DB, apiKey string) (*Principal, error) {
	// Use the user service to validate the API key
	repo := newUserRepository(db)
	svc := newUserService(repo, cfg)

	user, _, err := svc.ValidateAPIKey(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	roles := []string{"user"}
	if user.IsAdmin {
		roles = append(roles, "admin")
	}

	return &Principal{
		ID:           user.ID,
		Subject:      user.ID,
		Email:        user.Email,
		Username:     user.Username,
		Name:         user.Name,
		Roles:        roles,
		AuthMethod:   AuthMethodAPIKey,
		AuthProvider: "local",
	}, nil
}

func getStringClaim(claims jwt.MapClaims, key string) string {
	if v, ok := claims[key].(string); ok {
		return v
	}
	return ""
}
