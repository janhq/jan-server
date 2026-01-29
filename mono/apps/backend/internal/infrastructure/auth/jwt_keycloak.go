package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

// PrincipalClaims represent the subset of JWT claims we care about.
type PrincipalClaims struct {
	Subject           string
	Issuer            string
	Audience          []string
	PreferredUsername string
	Email             string
	Name              string
	Picture           string
	Roles             []string
	Groups            []string
	FeatureFlags      []string
	Scopes            []string
	Attributes        map[string]any
	ExpiresAt         time.Time
	IssuedAt          time.Time
	NotBefore         time.Time
	TokenID           string
	AuthorizedParty   string
}

// KeycloakValidator validates JWT tokens against Keycloak JWKS.
type KeycloakValidator struct {
	issuer          string
	audience        string
	authorizedParty string
	jwksURL         string
	logger          zerolog.Logger
	refreshEvery    time.Duration
	clockSkew       time.Duration
	jwks            atomic.Pointer[keyfunc.JWKS]
	lastErr         atomic.Value // stores lastErrWrap
}

// lastErrWrap is a sentinel wrapper to avoid storing bare nil in atomic.Value.
type lastErrWrap struct{ Err error }

const (
	jwksInitialRetryInterval   = time.Second
	jwksInitialRetryMaxBackoff = 10 * time.Second
	jwksInitialRetryTimeout    = 2 * time.Minute
)

// NewKeycloakValidator initialises JWKS fetching and returns a validator.
func NewKeycloakValidator(
	ctx context.Context,
	jwksURL,
	issuer,
	audience,
	authorizedParty string,
	refreshEvery,
	clockSkew time.Duration,
	logger zerolog.Logger,
) (*KeycloakValidator, error) {
	if jwksURL == "" {
		return nil, errors.New("jwks url is required")
	}

	validator := &KeycloakValidator{
		issuer:          issuer,
		audience:        audience,
		authorizedParty: authorizedParty,
		jwksURL:         jwksURL,
		logger:          logger,
		refreshEvery:    refreshEvery,
		clockSkew:       clockSkew,
	}
	// Initialize with a non-nil wrapper value
	validator.lastErr.Store(lastErrWrap{Err: nil})

	if err := validator.initJWKS(ctx); err != nil {
		return nil, err
	}

	return validator, nil
}

func (v *KeycloakValidator) initJWKS(ctx context.Context) error {
	options := keyfunc.Options{
		RefreshErrorHandler: func(err error) {
			// Always store non-nil wrapper type
			v.lastErr.Store(lastErrWrap{Err: err})
			if err != nil {
				v.logger.Error().Err(err).Msg("jwks refresh failed")
			}
		},
		RefreshInterval:   v.refreshEvery,
		RefreshUnknownKID: true,
	}

	if ctx != nil {
		options.Ctx = ctx
	}

	backoff := jwksInitialRetryInterval
	deadline := time.Now().Add(jwksInitialRetryTimeout)
	if ctx != nil {
		if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
			deadline = ctxDeadline
		}
	}

	for attempt := 1; ; attempt++ {
		jwks, err := keyfunc.Get(v.jwksURL, options)
		if err == nil {
			v.lastErr.Store(lastErrWrap{Err: nil})
			v.jwks.Store(jwks)
			return nil
		}

		v.logger.Warn().
			Err(err).
			Str("jwks_url", v.jwksURL).
			Int("attempt", attempt).
			Msg("initial jwks fetch failed, retrying")

		if ctx != nil {
			select {
			case <-ctx.Done():
				return fmt.Errorf("fetch jwks: %w", ctx.Err())
			case <-time.After(backoff):
			}
		} else {
			time.Sleep(backoff)
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("fetch jwks: %w", err)
		}

		if next := backoff * 2; next <= jwksInitialRetryMaxBackoff {
			backoff = next
		} else {
			backoff = jwksInitialRetryMaxBackoff
		}
	}
}

// Validate parses and validates the given JWT returning principal claims.
func (v *KeycloakValidator) Validate(_ context.Context, rawToken string) (*PrincipalClaims, error) {
	jwks := v.jwks.Load()
	if jwks == nil {
		return nil, errors.New("jwks not initialised")
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256"}))
	token, err := parser.ParseWithClaims(rawToken, jwt.MapClaims{}, jwks.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	iss, _ := mapClaims["iss"].(string)
	if iss != v.issuer {
		return nil, fmt.Errorf("issuer mismatch %s", iss)
	}

	var audiences []string
	if audRaw, ok := mapClaims["aud"]; ok {
		switch val := audRaw.(type) {
		case string:
			if val != v.audience {
				return nil, fmt.Errorf("audience mismatch")
			}
			audiences = append(audiences, val)
		case []interface{}:
			found := false
			for _, item := range val {
				if s, ok := item.(string); ok {
					if s == v.audience {
						found = true
					}
					audiences = append(audiences, s)
				}
			}
			if !found {
				return nil, fmt.Errorf("audience mismatch")
			}
		default:
			return nil, fmt.Errorf("aud claim unsupported type %T", val)
		}
	}

	sub, _ := mapClaims["sub"].(string)
	if sub == "" {
		return nil, errors.New("sub claim missing")
	}

	preferredUsername, _ := mapClaims["preferred_username"].(string)
	email, _ := mapClaims["email"].(string)
	name, _ := mapClaims["name"].(string)
	picture, _ := mapClaims["picture"].(string)
	azp := claimString(mapClaims["azp"])
	if v.authorizedParty != "" && azp != "" && azp != v.authorizedParty {
		return nil, errors.New("authorized party mismatch")
	}

	var roles []string
	if realmAccess, ok := mapClaims["realm_access"].(map[string]any); ok {
		if rawRoles, ok := realmAccess["roles"].([]interface{}); ok {
			for _, role := range rawRoles {
				if s, ok := role.(string); ok {
					roles = appendUnique(roles, s)
				}
			}
		}
	}
	// Include client-level roles as well (resource_access.<client>.roles)
	if resourceAccess, ok := mapClaims["resource_access"].(map[string]any); ok {
		for _, entry := range resourceAccess {
			if clientMap, ok := entry.(map[string]any); ok {
				if rawRoles, ok := clientMap["roles"].([]interface{}); ok {
					for _, role := range rawRoles {
						if s, ok := role.(string); ok {
							roles = appendUnique(roles, s)
						}
					}
				}
			}
		}
	}

	var scopes []string
	if scopeStr, ok := mapClaims["scope"].(string); ok && scopeStr != "" {
		scopes = strings.Split(scopeStr, " ")
	}

	groups := extractStringSlice(mapClaims["groups"])
	featureFlags := extractStringSlice(mapClaims["feature_flags"])

	var attributes map[string]any
	if rawAttrs, ok := mapClaims["attributes"].(map[string]any); ok && len(rawAttrs) > 0 {
		attributes = rawAttrs
	}

	expires := jwtNumericTime(mapClaims["exp"])
	issued := jwtNumericTime(mapClaims["iat"])
	notBefore := jwtNumericTime(mapClaims["nbf"])

	now := time.Now().UTC()
	if !expires.IsZero() && now.After(expires.Add(v.clockSkew)) {
		return nil, errors.New("token expired")
	}
	if !notBefore.IsZero() && now.Add(v.clockSkew).Before(notBefore) {
		return nil, errors.New("token not yet valid")
	}

	return &PrincipalClaims{
		Subject:           sub,
		Issuer:            iss,
		PreferredUsername: preferredUsername,
		Email:             email,
		Name:              name,
		Picture:           picture,
		Roles:             roles,
		Scopes:            scopes,
		Groups:            groups,
		FeatureFlags:      featureFlags,
		Attributes:        attributes,
		ExpiresAt:         expires,
		IssuedAt:          issued,
		NotBefore:         notBefore,
		TokenID:           claimString(mapClaims["jti"]),
		Audience:          audiences,
		AuthorizedParty:   azp,
	}, nil
}

func extractStringSlice(value any) []string {
	if value == nil {
		return nil
	}

	switch raw := value.(type) {
	case []any:
		out := make([]string, 0, len(raw))
		for _, v := range raw {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return raw
	}
	return nil
}

func appendUnique(slice []string, item string) []string {
	for _, existing := range slice {
		if existing == item {
			return slice
		}
	}
	return append(slice, item)
}

// Ready indicates whether JWKS has been successfully loaded.
func (v *KeycloakValidator) Ready() bool {
	if v.jwks.Load() == nil {
		return false
	}
	if val := v.lastErr.Load(); val != nil {
		if wrap, ok := val.(lastErrWrap); ok && wrap.Err != nil {
			return false
		}
	}
	return true
}

func jwtNumericTime(value any) time.Time {
	switch timeValue := value.(type) {
	case float64:
		return time.Unix(int64(timeValue), 0).UTC()
	case int64:
		return time.Unix(timeValue, 0).UTC()
	case json.Number:
		if unixTime, err := timeValue.Int64(); err == nil {
			return time.Unix(unixTime, 0).UTC()
		}
	}
	return time.Time{}
}

func claimString(value any) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}
