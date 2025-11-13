package auth

import (
	"github.com/gin-gonic/gin"

	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/apikeyhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	guestauth "jan-server/services/llm-api/internal/interfaces/httpserver/handlers/guesthandler"
)

// AuthRoute handles authentication routes
type AuthRoute struct {
	guestHandler         *guestauth.GuestHandler
	upgradeHandler       *guestauth.UpgradeHandler
	tokenHandler         *authhandler.TokenHandler
	apiKeyHandler        *apikeyhandler.Handler
	authHandler          *authhandler.AuthHandler
	keycloakOAuthHandler *authhandler.KeycloakOAuthHandler
}

// NewAuthRoute creates a new auth route
func NewAuthRoute(
	guestHandler *guestauth.GuestHandler,
	upgradeHandler *guestauth.UpgradeHandler,
	tokenHandler *authhandler.TokenHandler,
	apiKeyHandler *apikeyhandler.Handler,
	authHandler *authhandler.AuthHandler,
	keycloakOAuthHandler *authhandler.KeycloakOAuthHandler,
) *AuthRoute {
	return &AuthRoute{
		guestHandler:         guestHandler,
		upgradeHandler:       upgradeHandler,
		tokenHandler:         tokenHandler,
		apiKeyHandler:        apiKeyHandler,
		authHandler:          authHandler,
		keycloakOAuthHandler: keycloakOAuthHandler,
	}
}

// RegisterRouter registers auth routes
func (a *AuthRoute) RegisterRouter(router gin.IRouter, protectedRouter gin.IRouter) {
	// Public routes - Guest login
	router.POST("/auth/guest-login", a.CreateGuestLogin)
	router.POST("/auth/refresh-token", a.RefreshToken)
	router.GET("/auth/logout", a.Logout)

	// Public routes - Keycloak OAuth2/OIDC (simplified)
	router.GET("/auth/login", a.KeycloakLogin)
	router.GET("/auth/callback", a.KeycloakCallback)
	router.POST("/auth/validate", a.ValidateKeycloakToken)
	router.POST("/auth/revoke", a.RevokeKeycloakToken)

	// API key validation endpoint (for Kong plugin)
	router.POST("/auth/validate-api-key", a.ValidateAPIKey)

	// Protected routes (require authentication)
	protectedRouter.POST("/auth/upgrade", a.UpgradeAccount)
	protectedRouter.GET("/auth/me", a.GetMe)
	protectedRouter.POST("/auth/api-keys", a.authHandler.WithAppUserAuthChain(a.CreateAPIKey)...)
	protectedRouter.GET("/auth/api-keys", a.authHandler.WithAppUserAuthChain(a.ListAPIKeys)...)
	protectedRouter.DELETE("/auth/api-keys/:id", a.authHandler.WithAppUserAuthChain(a.DeleteAPIKey)...)
}

// CreateGuestLogin godoc
// @Summary Create guest user account
// @Description Creates a temporary guest user account and returns JWT tokens. Guest users have limited access and can be upgraded to full accounts later.
// @Tags Authentication API
// @Accept json
// @Produce json
// @Success 200 {object} object "Guest user created with access and refresh tokens"
// @Failure 500 {object} responses.ErrorResponse "Internal server error - failed to create guest user"
// @Router /auth/guest-login [post]
func (a *AuthRoute) CreateGuestLogin(c *gin.Context) {
	a.guestHandler.CreateGuest(c)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Exchanges a valid refresh token for a new access token. Refresh token must be provided in Authorization header or refresh_token cookie.
// @Tags Authentication API
// @Accept json
// @Produce json
// @Param refresh_token body string false "Refresh token (can also be in Authorization header)"
// @Success 200 {object} object "New access token and refresh token"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or expired refresh token"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /auth/refresh-token [post]
func (a *AuthRoute) RefreshToken(c *gin.Context) {
	a.tokenHandler.RefreshToken(c)
}

// Logout godoc
// @Summary Logout user
// @Description Revokes the current access token and clears authentication cookies. After logout, the user must re-authenticate.
// @Tags Authentication API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} object "Successfully logged out"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid token"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /auth/logout [get]
func (a *AuthRoute) Logout(c *gin.Context) {
	a.tokenHandler.Logout(c)
}

// UpgradeAccount godoc
// @Summary Upgrade guest to permanent account
// @Description Converts a guest user account to a permanent account with email/password credentials. Guest flag is removed and user gains full access.
// @Tags Authentication API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object true "Upgrade request with email and password"
// @Success 200 {object} object "Account upgraded successfully with new tokens"
// @Failure 400 {object} responses.ErrorResponse "Invalid request - missing email or password"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - not a guest user or invalid token"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /auth/upgrade [post]
func (a *AuthRoute) UpgradeAccount(c *gin.Context) {
	a.upgradeHandler.Upgrade(c)
}

// GetMe godoc
// @Summary Get current user information
// @Description Returns the authenticated user's profile information including user ID, email, roles, and guest status.
// @Tags Authentication API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} object "User profile information"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or expired token"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /auth/me [get]
func (a *AuthRoute) GetMe(c *gin.Context) {
	a.tokenHandler.GetMe(c)
}

// CreateAPIKey godoc
// @Summary Create API key
// @Description Creates a new API key for the authenticated user. API keys provide programmatic access without requiring user credentials.
// @Tags Authentication API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object true "API key creation request with name and optional scopes"
// @Success 201 {object} object "API key created successfully with key value"
// @Failure 400 {object} responses.ErrorResponse "Invalid request - missing required fields"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or expired token"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /auth/api-keys [post]
func (a *AuthRoute) CreateAPIKey(c *gin.Context) {
	a.apiKeyHandler.Create(c)
}

// ListAPIKeys godoc
// @Summary List user's API keys
// @Description Returns all API keys created by the authenticated user. Key values are not returned, only metadata.
// @Tags Authentication API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} object "List of API keys with metadata"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or expired token"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /auth/api-keys [get]
func (a *AuthRoute) ListAPIKeys(c *gin.Context) {
	a.apiKeyHandler.List(c)
}

// DeleteAPIKey godoc
// @Summary Delete API key
// @Description Revokes and deletes an API key by ID. Deleted keys can no longer be used for authentication.
// @Tags Authentication API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "API key ID"
// @Success 204 "API key deleted successfully"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - invalid or expired token"
// @Failure 404 {object} responses.ErrorResponse "API key not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /auth/api-keys/{id} [delete]
func (a *AuthRoute) DeleteAPIKey(c *gin.Context) {
	a.apiKeyHandler.Delete(c)
}

// ValidateAPIKey godoc
// @Summary Validate API key (Kong Plugin)
// @Description Internal endpoint used by Kong API Gateway to validate API keys. Not intended for direct client use.
// @Tags Authentication API
// @Accept json
// @Produce json
// @Param request body object true "API key validation request"
// @Success 200 {object} object "API key is valid with user information"
// @Failure 401 {object} responses.ErrorResponse "Invalid API key"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /auth/validate-api-key [post]
func (a *AuthRoute) ValidateAPIKey(c *gin.Context) {
	a.apiKeyHandler.Validate(c)
}

// KeycloakLogin godoc
// @Summary Initiate Keycloak OAuth2 login
// @Description Returns the Keycloak authorization URL for frontend to redirect users. Supports OAuth2 authorization code flow with PKCE.
// @Tags Authentication API
// @Accept json
// @Produce json
// @Param redirect_url query string false "URL to redirect after successful login"
// @Success 200 {object} object{authorization_url=string,state=string} "Authorization URL and state parameter"
// @Failure 500 {object} responses.ErrorResponse "Failed to initiate login"
// @Router /auth/login [get]
func (a *AuthRoute) KeycloakLogin(c *gin.Context) {
	if a.keycloakOAuthHandler != nil {
		a.keycloakOAuthHandler.InitiateLogin(c)
	} else {
		c.JSON(500, gin.H{"error": "Keycloak OAuth is not configured"})
	}
}

// KeycloakCallback godoc
// @Summary Handle Keycloak OAuth2 callback
// @Description Handles the OAuth2 callback from Keycloak, exchanges authorization code for JWT tokens
// @Tags Authentication API
// @Accept json
// @Produce json
// @Param code query string true "Authorization code from Keycloak"
// @Param state query string true "State parameter for CSRF protection"
// @Success 200 {object} object{access_token=string,refresh_token=string,expires_in=int,token_type=string} "JWT tokens"
// @Failure 400 {object} responses.ErrorResponse "Missing code or state"
// @Failure 401 {object} responses.ErrorResponse "Invalid state parameter"
// @Failure 500 {object} responses.ErrorResponse "Failed to exchange code for tokens"
// @Router /auth/callback [get]
func (a *AuthRoute) KeycloakCallback(c *gin.Context) {
	if a.keycloakOAuthHandler != nil {
		a.keycloakOAuthHandler.HandleCallback(c)
	} else {
		c.JSON(500, gin.H{"error": "Keycloak OAuth is not configured"})
	}
}

// ValidateKeycloakToken godoc
// @Summary Validate Keycloak access token
// @Description Validates an access token against Keycloak's userinfo endpoint
// @Tags Authentication API
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} object{valid=bool,user_info=object} "Token is valid with user information"
// @Failure 401 {object} responses.ErrorResponse "Invalid or expired token"
// @Failure 500 {object} responses.ErrorResponse "Keycloak OAuth is not configured"
// @Router /auth/validate [post]
func (a *AuthRoute) ValidateKeycloakToken(c *gin.Context) {
	if a.keycloakOAuthHandler != nil {
		a.keycloakOAuthHandler.ValidateAccessToken(c)
	} else {
		c.JSON(500, gin.H{"error": "Keycloak OAuth is not configured"})
	}
}

// RevokeKeycloakToken godoc
// @Summary Revoke Keycloak refresh token
// @Description Revokes a refresh token to invalidate it
// @Tags Authentication API
// @Accept json
// @Produce json
// @Param request body object{refresh_token=string} true "Token to revoke"
// @Success 200 {object} object{message=string} "Token revoked successfully"
// @Failure 400 {object} responses.ErrorResponse "Invalid request body"
// @Failure 500 {object} responses.ErrorResponse "Keycloak OAuth is not configured"
// @Router /auth/revoke [post]
func (a *AuthRoute) RevokeKeycloakToken(c *gin.Context) {
	if a.keycloakOAuthHandler != nil {
		a.keycloakOAuthHandler.RevokeKeycloakToken(c)
	} else {
		c.JSON(500, gin.H{"error": "Keycloak OAuth is not configured"})
	}
}
