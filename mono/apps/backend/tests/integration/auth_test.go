package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupAuthTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Mock auth endpoints for testing
	r.POST("/api/v1/auth/register", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Username string `json:"username"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Email == "" || req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email and password required"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"user": gin.H{
				"id":       "user-123",
				"email":    req.Email,
				"username": req.Username,
			},
			"access_token":  "mock-access-token",
			"refresh_token": "mock-refresh-token",
		})
	})

	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.Email == "" || req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email and password required"})
			return
		}
		// Mock successful login
		c.JSON(http.StatusOK, gin.H{
			"user": gin.H{
				"id":    "user-123",
				"email": req.Email,
			},
			"access_token":  "mock-access-token",
			"refresh_token": "mock-refresh-token",
		})
	})

	return r
}

func TestRegisterEndpoint(t *testing.T) {
	router := setupAuthTestRouter()

	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
	}{
		{
			name: "successful registration",
			body: map[string]string{
				"email":    "test@example.com",
				"password": "password123",
				"username": "testuser",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "missing email",
			body: map[string]string{
				"password": "password123",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			body: map[string]string{
				"email": "test@example.com",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestLoginEndpoint(t *testing.T) {
	router := setupAuthTestRouter()

	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
	}{
		{
			name: "successful login",
			body: map[string]string{
				"email":    "test@example.com",
				"password": "password123",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "missing credentials",
			body: map[string]string{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
