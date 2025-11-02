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
	PreferredUsername string
	Roles             []string
	Scopes            []string
	ExpiresAt         time.Time
	IssuedAt          time.Time
	NotBefore         time.Time
	TokenID           string
}

// KeycloakValidator validates JWT tokens against Keycloak JWKS.
type KeycloakValidator struct {
	issuer       string
	audience     string
	jwksURL      string
	logger       zerolog.Logger
	refreshEvery time.Duration
	jwks         atomic.Pointer[keyfunc.JWKS]
	lastErr      atomic.Value // stores lastErrWrap
}

// lastErrWrap is a sentinel wrapper to avoid storing bare nil in atomic.Value.
type lastErrWrap struct{ Err error }

const (
	jwksInitialRetryInterval   = time.Second
	jwksInitialRetryMaxBackoff = 10 * time.Second
	jwksInitialRetryTimeout    = 2 * time.Minute
)

// NewKeycloakValidator initialises JWKS fetching and returns a validator.
func NewKeycloakValidator(ctx context.Context, jwksURL, issuer, audience string, refreshEvery time.Duration, logger zerolog.Logger) (*KeycloakValidator, error) {
	if jwksURL == "" {
		return nil, errors.New("jwks url is required")
	}

	validator := &KeycloakValidator{
		issuer:       issuer,
		audience:     audience,
		jwksURL:      jwksURL,
		logger:       logger,
		refreshEvery: refreshEvery,
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

	if audRaw, ok := mapClaims["aud"]; ok {
		switch val := audRaw.(type) {
		case string:
			if val != v.audience {
				return nil, fmt.Errorf("audience mismatch")
			}
		case []interface{}:
			found := false
			for _, item := range val {
				if s, ok := item.(string); ok && s == v.audience {
					found = true
					break
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

	var roles []string
	if realmAccess, ok := mapClaims["realm_access"].(map[string]any); ok {
		if rawRoles, ok := realmAccess["roles"].([]interface{}); ok {
			for _, role := range rawRoles {
				if s, ok := role.(string); ok {
					roles = append(roles, s)
				}
			}
		}
	}

	var scopes []string
	if scopeStr, ok := mapClaims["scope"].(string); ok && scopeStr != "" {
		scopes = strings.Split(scopeStr, " ")
	}

	expires := jwtNumericTime(mapClaims["exp"])
	issued := jwtNumericTime(mapClaims["iat"])
	notBefore := jwtNumericTime(mapClaims["nbf"])

	now := time.Now().UTC()
	if !expires.IsZero() && now.After(expires) {
		return nil, errors.New("token expired")
	}
	if !notBefore.IsZero() && now.Before(notBefore) {
		return nil, errors.New("token not yet valid")
	}

	return &PrincipalClaims{
		Subject:           sub,
		PreferredUsername: preferredUsername,
		Roles:             roles,
		Scopes:            scopes,
		ExpiresAt:         expires,
		IssuedAt:          issued,
		NotBefore:         notBefore,
		TokenID:           claimString(mapClaims["jti"]),
	}, nil
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

func jwtNumericTime(raw any) time.Time {
	switch t := raw.(type) {
	case float64:
		return time.Unix(int64(t), 0).UTC()
	case int64:
		return time.Unix(t, 0).UTC()
	case json.Number:
		if v, err := t.Int64(); err == nil {
			return time.Unix(v, 0).UTC()
		}
	}
	return time.Time{}
}

func claimString(raw any) string {
	if s, ok := raw.(string); ok {
		return s
	}
	return ""
}
