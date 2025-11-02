package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/llm-api/domain"
	authvalidator "jan-server/services/llm-api/infrastructure/auth"
	"jan-server/services/llm-api/interfaces/httpserver/responses"
)

const principalContextKey = "principal"

// AuthMiddleware validates API key headers injected by Kong or JWT bearer tokens issued by Keycloak.
func AuthMiddleware(validator *authvalidator.KeycloakValidator, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if principal, ok := principalFromAPIKey(c); ok {
			setPrincipal(c, principal)
			c.Next()
			return
		}

		principal, err := principalFromJWT(c, validator)
		if err == nil {
			setPrincipal(c, principal)
			c.Next()
			return
		}

		logger.Warn().
			Str("path", c.FullPath()).
			Str("method", c.Request.Method).
			Msg("unauthenticated request")
		c.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
			Type:      responses.ErrorTypeAuth,
			Code:      "unauthorized",
			Message:   "authentication required",
			RequestID: RequestIDFromContext(c),
		})
	}
}

// PrincipalFromContext returns the authenticated principal, if any.
func PrincipalFromContext(c *gin.Context) (domain.Principal, bool) {
	val, ok := c.Get(principalContextKey)
	if !ok {
		return domain.Principal{}, false
	}
	principal, ok := val.(domain.Principal)
	return principal, ok
}

func setPrincipal(c *gin.Context, principal domain.Principal) {
	c.Set(principalContextKey, principal)
	c.Request.Header.Set("X-Principal-Id", principal.ID)
	c.Request.Header.Set("X-Auth-Method", string(principal.AuthMethod))
	if len(principal.Scopes) > 0 {
		c.Request.Header.Set("X-Scopes", strings.Join(principal.Scopes, " "))
	}
	c.Writer.Header().Set("X-Principal-Id", principal.ID)
	c.Writer.Header().Set("X-Auth-Method", string(principal.AuthMethod))
	if len(principal.Scopes) > 0 {
		c.Writer.Header().Set("X-Scopes", strings.Join(principal.Scopes, " "))
	}
}

func principalFromAPIKey(c *gin.Context) (domain.Principal, bool) {
	headers := c.Request.Header
	consumerID := headers.Get("X-Consumer-ID")
	if consumerID == "" {
		return domain.Principal{}, false
	}
	customID := headers.Get("X-Consumer-Custom-ID")
	username := headers.Get("X-Consumer-Username")
	principalID := customID
	if principalID == "" {
		principalID = username
	}
	if principalID == "" {
		principalID = consumerID
	}
	scopes := parseScopes(headers.Get("X-Consumer-Groups"))
	return domain.Principal{
		ID:         principalID,
		AuthMethod: domain.AuthMethodAPIKey,
		Subject:    consumerID,
		Username:   username,
		Scopes:     scopes,
		Credentials: map[string]string{
			"consumer_id":        consumerID,
			"consumer_custom_id": customID,
			"consumer_username":  username,
		},
	}, true
}

func principalFromJWT(c *gin.Context, validator *authvalidator.KeycloakValidator) (domain.Principal, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return domain.Principal{}, http.ErrNoCookie
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return domain.Principal{}, http.ErrNoCookie
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return domain.Principal{}, http.ErrNoCookie
	}
	claims, err := validator.Validate(c.Request.Context(), token)
	if err != nil {
		return domain.Principal{}, err
	}
	return domain.Principal{
		ID:         claims.Subject,
		AuthMethod: domain.AuthMethodJWT,
		Subject:    claims.Subject,
		Username:   claims.PreferredUsername,
		Scopes:     claims.Scopes,
		Credentials: map[string]string{
			"token_id": claims.TokenID,
		},
	}, nil
}

func parseScopes(raw string) []string {
	if raw == "" {
		return nil
	}
	items := strings.Split(raw, ",")
	var out []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}
