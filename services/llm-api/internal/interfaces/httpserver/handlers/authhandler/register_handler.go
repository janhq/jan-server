package authhandler

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/llm-api/internal/infrastructure/keycloak"
)

// RegisterHandler handles user registration
type RegisterHandler struct {
	kc     *keycloak.Client
	logger zerolog.Logger
}

// NewRegisterHandler creates a new register handler
func NewRegisterHandler(kc *keycloak.Client, logger zerolog.Logger) *RegisterHandler {
	return &RegisterHandler{
		kc:     kc,
		logger: logger,
	}
}

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// RegisterResponse represents the registration response
type RegisterResponse struct {
	Message      string `json:"message"`
	UserID       string `json:"user_id"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
}

// Register godoc
// @Summary Register a new user
// @Description Creates a new user account with email, username, and password
// @Tags Authentication API
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} RegisterResponse "User created successfully"
// @Failure 400 {object} object "Invalid request - validation error"
// @Failure 409 {object} object "Conflict - user already exists"
// @Failure 500 {object} object "Internal server error"
// @Router /auth/register [post]
func (h *RegisterHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Additional validation
	if err := h.validateRegistration(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": err.Error(),
		})
		return
	}

	// Normalize input
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Username = strings.TrimSpace(req.Username)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	ctx := c.Request.Context()

	// Create user in Keycloak
	enabled := true
	userReq := keycloak.AdminUserRequest{
		Email:    req.Email,
		Username: req.Username,
		First:    req.FirstName,
		Last:     req.LastName,
		Enabled:  &enabled,
	}

	userID, err := h.kc.CreateUser(ctx, userReq)
	if err != nil {
		errMsg := err.Error()
		h.logger.Error().Err(err).Str("email", req.Email).Str("username", req.Username).Msg("Failed to create user")

		// Check for duplicate user error
		if strings.Contains(errMsg, "409") || strings.Contains(errMsg, "exists") || strings.Contains(errMsg, "duplicate") {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "User already exists",
				"message": "A user with this email or username already exists",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Registration failed",
			"message": "Failed to create user account",
		})
		return
	}

	// Set user password
	if err := h.kc.SetUserPassword(ctx, userID, req.Password, false); err != nil {
		h.logger.Error().Err(err).Str("user_id", userID).Msg("Failed to set user password")

		// Try to clean up the user if password setting fails
		if delErr := h.kc.DeleteUser(ctx, userID); delErr != nil {
			h.logger.Error().Err(delErr).Str("user_id", userID).Msg("Failed to delete user after password set failure")
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Registration failed",
			"message": "Failed to set user password",
		})
		return
	}

	h.logger.Info().
		Str("user_id", userID).
		Str("email", req.Email).
		Str("username", req.Username).
		Msg("User registered successfully")

	// Get tokens for the newly registered user to enable auto-login
	tokens, err := h.kc.GetUserTokens(ctx, req.Email, req.Password)
	if err != nil {
		h.logger.Warn().Err(err).Str("user_id", userID).Msg("Failed to get tokens for new user, user will need to login manually")
		// Return success without tokens - user can still login manually
		c.JSON(http.StatusCreated, RegisterResponse{
			Message: "User registered successfully",
			UserID:  userID,
		})
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		Message:      "User registered successfully",
		UserID:       userID,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// validateRegistration performs additional validation on the registration request
func (h *RegisterHandler) validateRegistration(req *RegisterRequest) error {
	// Username validation: alphanumeric, underscore, hyphen only
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !usernameRegex.MatchString(req.Username) {
		return &ValidationError{Field: "username", Message: "Username can only contain letters, numbers, underscores, and hyphens"}
	}

	// Password strength check
	if len(req.Password) < 8 {
		return &ValidationError{Field: "password", Message: "Password must be at least 8 characters long"}
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
