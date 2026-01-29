package authhandler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// RefreshTokenRequest represents the request body for token refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse represents the response after refreshing tokens
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// RefreshKeycloakToken refreshes an access token using a refresh token
func (h *KeycloakOAuthHandler) RefreshKeycloakToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Prepare token refresh request
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", h.keycloakBaseURL, h.realm)

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", h.clientID)
	data.Set("refresh_token", req.RefreshToken)

	// Add client secret if available (for confidential clients)
	if h.clientSecret != "" {
		data.Set("client_secret", h.clientSecret)
	}

	// Make HTTP request to Keycloak
	httpReq, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create refresh request"})
		return
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh token", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	// Check for errors from Keycloak
	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{
			"error":   "Token refresh failed",
			"details": string(body),
		})
		return
	}

	// Parse token response
	var tokenResp RefreshTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse token response"})
		return
	}

	// Return new tokens
	c.JSON(http.StatusOK, tokenResp)
}

// ValidateAccessToken validates an access token against Keycloak's userinfo endpoint
func (h *KeycloakOAuthHandler) ValidateAccessToken(c *gin.Context) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
		return
	}

	// Extract Bearer token
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
		return
	}

	// Call Keycloak's userinfo endpoint to validate token
	userinfoURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/userinfo", h.keycloakBaseURL, h.realm)

	httpReq, err := http.NewRequest("GET", userinfoURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create validation request"})
		return
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate token", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	// Check validation result
	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Token validation failed",
			"details": string(body),
		})
		return
	}

	// Parse userinfo
	var userInfo map[string]interface{}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user info"})
		return
	}

	// Return validation success with user info
	c.JSON(http.StatusOK, gin.H{
		"valid":     true,
		"user_info": userInfo,
	})
}

// RevokeKeycloakToken revokes a refresh token
func (h *KeycloakOAuthHandler) RevokeKeycloakToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Prepare token revocation request
	revokeURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/revoke", h.keycloakBaseURL, h.realm)

	data := url.Values{}
	data.Set("client_id", h.clientID)
	data.Set("token", req.RefreshToken)
	data.Set("token_type_hint", "refresh_token")

	// Add client secret if available
	if h.clientSecret != "" {
		data.Set("client_secret", h.clientSecret)
	}

	// Make HTTP request to Keycloak
	httpReq, err := http.NewRequest("POST", revokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create revocation request"})
		return
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	// Check revocation result
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, gin.H{
			"error":   "Token revocation failed",
			"details": string(body),
		})
		return
	}

	// Return success
	c.JSON(http.StatusOK, gin.H{
		"message": "Token revoked successfully",
	})
}
