package middlewares

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/llm-api/internal/domain"
	authvalidator "jan-server/services/llm-api/internal/infrastructure/auth"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses"
)

const principalContextKey = "principal"

// AuthMiddleware validates API key headers injected by Kong or JWT bearer tokens issued by Keycloak.
func AuthMiddleware(validator *authvalidator.KeycloakValidator, logger zerolog.Logger, fallbackIssuer string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiPrincipal, hasAPIKey := principalFromAPIKey(c, fallbackIssuer)
		jwtPrincipal, hasJWT, jwtErr := principalFromJWT(c, validator)

		if jwtErr != nil && !errors.Is(jwtErr, http.ErrNoCookie) {
			logger.Error().Err(jwtErr).Msg("jwt validation failed")
			responses.HandleErrorWithStatus(c, http.StatusUnauthorized, jwtErr, "unauthorized")
			return
		}

		switch {
		case hasAPIKey && hasJWT:
			merged, err := mergePrincipals(apiPrincipal, jwtPrincipal)
			if err != nil {
				logger.Warn().Err(err).Msg("principal mismatch between JWT and API key")
				responses.HandleErrorWithStatus(c, http.StatusUnauthorized, err, "conflicting credentials")
				return
			}
			setPrincipal(c, merged)
		case hasJWT:
			setPrincipal(c, jwtPrincipal)
		case hasAPIKey:
			setPrincipal(c, apiPrincipal)
		default:
			logger.Warn().
				Str("path", c.FullPath()).
				Str("method", c.Request.Method).
				Msg("unauthenticated request")
			responses.HandleErrorWithStatus(c, http.StatusUnauthorized, errors.New("authentication required"), "unauthorized")
			return
		}

		c.Next()
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

func principalFromAPIKey(c *gin.Context, fallbackIssuer string) (domain.Principal, bool) {
	headers := c.Request.Header

	// Check if there's actually an API key credential present
	// Kong sets X-Credential-Identifier only when a real API key is used
	if headers.Get("X-Credential-Identifier") == "" {
		return domain.Principal{}, false
	}

	consumerID := headers.Get("X-Consumer-ID")
	if consumerID == "" {
		return domain.Principal{}, false
	}

	username := headers.Get("X-Consumer-Username")
	customID := headers.Get("X-Consumer-Custom-ID")

	principalID := customID
	if principalID == "" {
		principalID = username
	}
	if principalID == "" {
		principalID = consumerID
	}
	scopes := parseScopes(headers.Get("X-Consumer-Groups"))
	credentials := map[string]string{
		"consumer_id":        consumerID,
		"consumer_custom_id": customID,
		"consumer_username":  username,
	}
	if credID := headers.Get("X-Credential-Identifier"); credID != "" {
		credentials["credential_identifier"] = credID
	}
	if route := headers.Get("X-Route-Id"); route != "" {
		credentials["route_id"] = route
	}
	return domain.Principal{
		ID:          principalID,
		AuthMethod:  domain.AuthMethodAPIKey,
		Subject:     principalID,
		Issuer:      fallbackIssuer,
		Username:    username,
		Scopes:      scopes,
		Credentials: credentials,
	}, true
}

func principalFromJWT(c *gin.Context, validator *authvalidator.KeycloakValidator) (domain.Principal, bool, error) {
	if validator == nil {
		return domain.Principal{}, false, http.ErrNoCookie
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return domain.Principal{}, false, http.ErrNoCookie
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return domain.Principal{}, false, http.ErrNoCookie
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return domain.Principal{}, false, http.ErrNoCookie
	}
	claims, err := validator.Validate(c.Request.Context(), token)
	if err != nil {
		return domain.Principal{}, false, err
	}
	credentials := map[string]string{
		"token_id": claims.TokenID,
	}
	if claims.Issuer != "" {
		credentials["issuer"] = claims.Issuer
	}
	if claims.Picture != "" {
		credentials["picture"] = claims.Picture
	}
	if claims.AuthorizedParty != "" {
		credentials["authorized_party"] = claims.AuthorizedParty
	}

	return domain.Principal{
		ID:              claims.Subject,
		AuthMethod:      domain.AuthMethodJWT,
		Subject:         claims.Subject,
		Issuer:          claims.Issuer,
		AuthorizedParty: claims.AuthorizedParty,
		Audience:        claims.Audience,
		Username:        claims.PreferredUsername,
		Email:           claims.Email,
		Name:            claims.Name,
		Scopes:          claims.Scopes,
		Credentials:     credentials,
	}, true, nil
}

func mergePrincipals(apiPrincipal, jwtPrincipal domain.Principal) (domain.Principal, error) {
	if apiPrincipal.Subject != "" && jwtPrincipal.Subject != "" && !strings.EqualFold(apiPrincipal.Subject, jwtPrincipal.Subject) {
		return domain.Principal{}, errors.New("principal subjects mismatch")
	}

	merged := jwtPrincipal
	merged.AuthMethod = domain.AuthMethodJWT
	merged.Credentials = map[string]string{}
	for k, v := range jwtPrincipal.Credentials {
		merged.Credentials[k] = v
	}
	for k, v := range apiPrincipal.Credentials {
		merged.Credentials[k] = v
	}
	merged.Credentials["authenticated_via"] = "jwt+api_key"
	merged.Credentials["api_key_subject"] = apiPrincipal.Subject
	merged.Credentials["api_key_consumer_id"] = apiPrincipal.Credentials["consumer_id"]
	merged.Credentials["api_key_username"] = apiPrincipal.Username

	if merged.Username == "" {
		merged.Username = apiPrincipal.Username
	}
	if merged.Email == "" {
		merged.Email = apiPrincipal.Email
	}
	if merged.Name == "" {
		merged.Name = apiPrincipal.Name
	}

	merged.Scopes = mergeScopes(jwtPrincipal.Scopes, apiPrincipal.Scopes)

	return merged, nil
}

func mergeScopes(primary, secondary []string) []string {
	if len(secondary) == 0 {
		return primary
	}
	seen := make(map[string]struct{}, len(primary)+len(secondary))
	var out []string
	for _, scope := range primary {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, exists := seen[scope]; !exists {
			out = append(out, scope)
			seen[scope] = struct{}{}
		}
	}
	for _, scope := range secondary {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		if _, exists := seen[scope]; !exists {
			out = append(out, scope)
			seen[scope] = struct{}{}
		}
	}
	return out
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
