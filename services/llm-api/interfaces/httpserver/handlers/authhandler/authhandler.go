package authhandler

import (
	"net/http"
	"strings"

	"menlo.ai/menlo-platform/config/envs"
	"menlo.ai/menlo-platform/internal/domain/apikey"
	"menlo.ai/menlo-platform/internal/domain/auth"
	"menlo.ai/menlo-platform/internal/domain/user"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/responses"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	authService   *auth.AuthService
	apiKeyService *apikey.APIKeyService
}

func NewAuthHandler(
	authService *auth.AuthService,
	apiKeyService *apikey.APIKeyService) *AuthHandler {
	return &AuthHandler{
		authService,
		apiKeyService,
	}
}

func (m *AuthHandler) RegisteredUserMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		userPublicId, ok := GetUserIDFromContext(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "3296ce86-783b-4c05-9fdb-930d3713024e",
			})
			return
		}
		if userPublicId == "" {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "80e1017d-038a-48c1-9de7-c3cdffdddb95",
			})
			return
		}
		user, err := m.authService.FindUserByPublicID(ctx, userPublicId)
		if err != nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "6272df83-f538-421b-93ba-c2b6f6d39f39",
			})
			return
		}
		if user == nil {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "b1ef40e7-9db9-477d-bb59-f3783585195d",
			})
			return
		}
		if !user.Enabled {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "b5fd2fa7-afd4-4c07-b70a-995fdb7906af",
			})
			return
		}
		SetUserToContext(reqCtx, user)
		reqCtx.Next()
	}
}

func (m *AuthHandler) JWTMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		userClaim, ok := GetUserClaimFromBearer(reqCtx)
		if ok {
			SetUserIDToContext(reqCtx, userClaim.ID)
		}
		reqCtx.Next()
	}
}

func (m *AuthHandler) APIKeyMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		userPublicID, ok := m.getUserPublicIDFromApikey(reqCtx)
		if ok {
			SetUserIDToContext(reqCtx, userPublicID)
		}
		reqCtx.Next()
	}
}

func (handler *AuthHandler) getUserPublicIDFromApikey(reqCtx *gin.Context) (string, bool) {
	tokenString, ok := GetTokenFromBearer(reqCtx)
	if !ok {
		return "", false
	}
	if !apikey.IsValidAPIKeyPrefix(tokenString) {
		return "", false
	}
	ctx := reqCtx.Request.Context()
	hashed := apikey.HashAPIKey(tokenString)
	apikeyEntity, err := handler.apiKeyService.FindByKeyHash(ctx, hashed)
	SetAPIKeyToContext(reqCtx, apikeyEntity)
	if err != nil {
		return "", false
	}
	if !apikeyEntity.IsValid() {
		return "", false
	}
	return apikeyEntity.UserPublicID, true
}

func (handler *AuthHandler) WithAppUserAuthChain(handleFuncs ...gin.HandlerFunc) []gin.HandlerFunc {
	middlewareChain := []gin.HandlerFunc{
		handler.JWTMiddleware(),
		handler.APIKeyMiddleware(),
		handler.RegisteredUserMiddleware(),
	}

	// Use append to correctly add the variadic arguments (the route handlers)
	return append(middlewareChain, handleFuncs...)
}

type UserContextKey string

const (
	UserContextKeyEntity   UserContextKey = "UserContextKeyEntity"
	UserContextKeyID       UserContextKey = "UserContextKeyID"
	APIKeyContextKeyEntity UserContextKey = "APIKeyContextKeyEntity"
)

func GetUserFromContext(reqCtx *gin.Context) (*user.User, bool) {
	v, ok := reqCtx.Get(string(UserContextKeyEntity))
	if !ok {
		return nil, false
	}
	return v.(*user.User), true
}

func SetUserToContext(reqCtx *gin.Context, user *user.User) {
	reqCtx.Set(string(UserContextKeyEntity), user)
}

func GetAPIKeyFromContext(reqCtx *gin.Context) (*apikey.APIKey, bool) {
	v, ok := reqCtx.Get(string(APIKeyContextKeyEntity))
	if !ok {
		return nil, false
	}
	return v.(*apikey.APIKey), true
}

func SetAPIKeyToContext(reqCtx *gin.Context, apiKey *apikey.APIKey) {
	reqCtx.Set(string(APIKeyContextKeyEntity), apiKey)
}

func GetUserIDFromContext(reqCtx *gin.Context) (string, bool) {
	userId, ok := reqCtx.Get(string(UserContextKeyID))
	if !ok {
		return "", false
	}
	v, ok := userId.(string)
	if !ok {
		return "", false
	}
	return v, true
}

func SetUserIDToContext(reqCtx *gin.Context, v string) {
	reqCtx.Set(string(UserContextKeyID), v)
}

func GetUserClaimFromBearer(reqCtx *gin.Context) (*auth.UserClaim, bool) {
	authHeader := reqCtx.GetHeader("Authorization")
	if authHeader == "" {
		return nil, false
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, false
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	return getUserClaimFromTokenString(tokenString)
}

func GetUserClaimFromRefreshToken(reqCtx *gin.Context) (*auth.UserClaim, bool) {
	refreshTokenString, err := reqCtx.Cookie(auth.RefreshTokenKey)
	if err != nil {
		return nil, false
	}
	return getUserClaimFromTokenString(refreshTokenString)
}

func getUserClaimFromTokenString(tokenString string) (*auth.UserClaim, bool) {
	token, err := jwt.ParseWithClaims(tokenString, &auth.UserClaim{}, func(token *jwt.Token) (interface{}, error) {
		return envs.ENV.OAUTH2_JWT_SECRET, nil
	})
	if err != nil {
		return nil, false
	}
	if !token.Valid {
		return nil, false
	}

	claims, ok := token.Claims.(*auth.UserClaim)
	if !ok {
		return nil, false
	}
	if claims.ID == "" {
		return nil, false
	}
	return claims, true
}

func GetTokenFromBearer(c *gin.Context) (string, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", false
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", false
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	return token, true
}
