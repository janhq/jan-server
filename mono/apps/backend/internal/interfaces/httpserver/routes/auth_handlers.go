package routes

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"jan-server/mono/apps/backend/internal/domain/user"
	"jan-server/mono/apps/backend/internal/infrastructure/config"
	"jan-server/mono/apps/backend/internal/infrastructure/database/repository"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// getUserService creates a user service instance.
func getUserService(cfg *config.Config, db *gorm.DB) *user.Service {
	repo := repository.NewUserRepository(db)
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

// ============================================
// Request/Response types
// ============================================

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type createAPIKeyRequest struct {
	Name      string     `json:"name" binding:"required,min=1,max=100"`
	ExpiresAt *time.Time `json:"expires_at"`
	Scopes    []string   `json:"scopes"`
}

type tokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// ============================================
// Auth Handlers Implementation
// ============================================

func localRegisterHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req registerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation_error",
				"message": err.Error(),
			})
			return
		}

		svc := getUserService(cfg, db)
		u, tokens, err := svc.Register(c.Request.Context(), user.CreateUserRequest{
			Email:    req.Email,
			Username: req.Username,
			Password: req.Password,
			Name:     req.Name,
		})

		if err != nil {
			if errors.Is(err, user.ErrUserAlreadyExists) {
				c.JSON(http.StatusConflict, gin.H{
					"error":   "user_exists",
					"message": "A user with this email or username already exists",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "registration_failed",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"user": u.ToResponse(),
			"tokens": tokenResponse{
				AccessToken:  tokens.AccessToken,
				RefreshToken: tokens.RefreshToken,
				TokenType:    tokens.TokenType,
				ExpiresIn:    tokens.ExpiresIn,
				ExpiresAt:    tokens.ExpiresAt,
			},
		})
	}
}

func localLoginHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation_error",
				"message": err.Error(),
			})
			return
		}

		svc := getUserService(cfg, db)
		u, tokens, err := svc.Login(c.Request.Context(), req.Email, req.Password)

		if err != nil {
			if errors.Is(err, user.ErrInvalidCredentials) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "invalid_credentials",
					"message": "Invalid email or password",
				})
				return
			}
			if errors.Is(err, user.ErrUserInactive) {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "account_inactive",
					"message": "Your account has been deactivated",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "login_failed",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user": u.ToResponse(),
			"tokens": tokenResponse{
				AccessToken:  tokens.AccessToken,
				RefreshToken: tokens.RefreshToken,
				TokenType:    tokens.TokenType,
				ExpiresIn:    tokens.ExpiresIn,
				ExpiresAt:    tokens.ExpiresAt,
			},
		})
	}
}

func localRefreshHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req refreshRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation_error",
				"message": err.Error(),
			})
			return
		}

		svc := getUserService(cfg, db)
		u, tokens, err := svc.RefreshTokens(c.Request.Context(), req.RefreshToken)

		if err != nil {
			if errors.Is(err, user.ErrInvalidToken) || errors.Is(err, user.ErrTokenExpired) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "invalid_token",
					"message": "Invalid or expired refresh token",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "refresh_failed",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user": u.ToResponse(),
			"tokens": tokenResponse{
				AccessToken:  tokens.AccessToken,
				RefreshToken: tokens.RefreshToken,
				TokenType:    tokens.TokenType,
				ExpiresIn:    tokens.ExpiresIn,
				ExpiresAt:    tokens.ExpiresAt,
			},
		})
	}
}

func localChangePasswordHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			return
		}

		var req changePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation_error",
				"message": err.Error(),
			})
			return
		}

		svc := getUserService(cfg, db)
		err := svc.ChangePassword(c.Request.Context(), principal.ID, req.OldPassword, req.NewPassword)

		if err != nil {
			if errors.Is(err, user.ErrInvalidCredentials) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "invalid_password",
					"message": "Current password is incorrect",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "change_password_failed",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Password changed successfully",
		})
	}
}

func refreshTokenHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return localRefreshHandler(cfg, db) // Alias to local refresh
}

func logoutHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// For stateless JWT, we just return success.
		// The client should discard the tokens.
		// If we had token blocklisting, we'd add the token to the blocklist here.
		c.JSON(http.StatusOK, gin.H{
			"message": "Logged out successfully",
		})
	}
}

func validateTokenHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"valid": false,
				"error": "missing_token",
			})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{
				"valid": false,
				"error": "invalid_format",
			})
			return
		}

		// Check if it's an API key
		if strings.HasPrefix(token, "sk_") {
			svc := getUserService(cfg, db)
			u, _, err := svc.ValidateAPIKey(c.Request.Context(), token)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"valid": false,
					"error": err.Error(),
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"valid": true,
				"user":  u.ToResponse(),
			})
			return
		}

		// For JWT, the auth middleware already validates
		// We can use the principal from context if auth middleware ran
		principal := middlewares.GetPrincipal(c)
		if principal != nil {
			c.JSON(http.StatusOK, gin.H{
				"valid": true,
				"user": gin.H{
					"id":       principal.ID,
					"email":    principal.Email,
					"username": principal.Username,
					"name":     principal.Name,
				},
			})
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{
			"valid": false,
			"error": "invalid_token",
		})
	}
}

func meHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			return
		}

		// Get full user from database
		svc := getUserService(cfg, db)
		u, err := svc.GetUserByID(c.Request.Context(), principal.ID)
		if err != nil {
			// Fall back to principal data
			c.JSON(http.StatusOK, gin.H{
				"id":       principal.ID,
				"email":    principal.Email,
				"username": principal.Username,
				"name":     principal.Name,
				"roles":    principal.Roles,
			})
			return
		}

		c.JSON(http.StatusOK, u.ToResponse())
	}
}

func createAPIKeyHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			return
		}

		var req createAPIKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation_error",
				"message": err.Error(),
			})
			return
		}

		svc := getUserService(cfg, db)
		keyResp, err := svc.CreateAPIKey(c.Request.Context(), principal.ID, user.CreateAPIKeyRequest{
			Name:      req.Name,
			ExpiresAt: req.ExpiresAt,
			Scopes:    req.Scopes,
		})

		if err != nil {
			if errors.Is(err, user.ErrMaxAPIKeysReached) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "max_keys_reached",
					"message": "Maximum number of API keys reached",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "create_key_failed",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, keyResp)
	}
}

func listAPIKeysHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			return
		}

		svc := getUserService(cfg, db)
		keys, err := svc.ListAPIKeys(c.Request.Context(), principal.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "list_keys_failed",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"api_keys": keys,
		})
	}
}

func deleteAPIKeyHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			return
		}

		keyID := c.Param("id")
		if keyID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation_error",
				"message": "API key ID is required",
			})
			return
		}

		svc := getUserService(cfg, db)
		err := svc.RevokeAPIKey(c.Request.Context(), principal.ID, keyID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "revoke_key_failed",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "API key revoked successfully",
		})
	}
}

func validateAPIKeyForKongHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"valid": false,
				"error": "missing_api_key",
			})
			return
		}

		svc := getUserService(cfg, db)
		u, _, err := svc.ValidateAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"valid": false,
				"error": err.Error(),
			})
			return
		}

		// Return user info for Kong to inject into headers
		c.JSON(http.StatusOK, gin.H{
			"valid":      true,
			"user_id":    u.ID,
			"user_email": u.Email,
			"username":   u.Username,
		})
	}
}

// ============================================
// Keycloak Handlers (Placeholder)
// ============================================

func keycloakLoginHandler(cfg *config.Config) gin.HandlerFunc {
	return notImplemented("keycloak login")
}

func keycloakCallbackHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return notImplemented("keycloak callback")
}

func guestLoginHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return notImplemented("guest login")
}

func upgradeGuestHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return notImplemented("upgrade guest")
}
