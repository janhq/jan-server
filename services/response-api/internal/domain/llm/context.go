package llm

import "context"

type contextKey string

const authTokenKey contextKey = "llm-auth-token"

// ContextWithAuthToken stores an Authorization header value in context for downstream LLM calls.
func ContextWithAuthToken(ctx context.Context, authHeader string) context.Context {
	if ctx == nil || authHeader == "" {
		return ctx
	}
	return context.WithValue(ctx, authTokenKey, authHeader)
}

// AuthTokenFromContext extracts the Authorization header value if one was provided.
func AuthTokenFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if token, ok := ctx.Value(authTokenKey).(string); ok {
		return token
	}
	return ""
}
