package connectorhandler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"jan-server/services/llm-api/internal/domain/connector"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

// ConnectorHandler handles connector HTTP requests.
type ConnectorHandler struct {
	service *connector.Service
}

// NewConnectorHandler creates a new connector handler.
func NewConnectorHandler(service *connector.Service) *ConnectorHandler {
	return &ConnectorHandler{
		service: service,
	}
}

// ListConnectors godoc
// @Summary List all connectors
// @Description Returns all available connector types with their connection status for the authenticated user
// @Tags Connectors API
// @Security BearerAuth
// @Produce json
// @Success 200 {object} ListConnectorsResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /v1/connectors [get]
func (h *ConnectorHandler) ListConnectors(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		responses.HandleError(c, err, "failed to get user ID")
		return
	}

	connectors, err := h.service.ListConnectors(c.Request.Context(), userID)
	if err != nil {
		responses.HandleError(c, err, "failed to list connectors")
		return
	}

	responseConnectors := make([]ConnectorInfoResponse, 0, len(connectors))
	for _, conn := range connectors {
		responseConnectors = append(responseConnectors, ConnectorInfoResponse{
			Type:        conn.Type,
			DisplayName: conn.DisplayName,
			Description: conn.Description,
			IconURL:     conn.IconURL,
			IsConnected: conn.IsConnected,
			Username:    conn.Username,
			Email:       conn.Email,
			AvatarURL:   conn.AvatarURL,
			ConnectedAt: conn.ConnectedAt,
			Scopes:      conn.Scopes,
			HasWrite:    conn.HasWrite,
			Enabled:     h.service.IsEnabled(conn.Type),
		})
	}

	c.JSON(http.StatusOK, ListConnectorsResponse{
		Connectors: responseConnectors,
	})
}

// GetConnector godoc
// @Summary Get connector details
// @Description Returns details and connection status for a specific connector type
// @Tags Connectors API
// @Security BearerAuth
// @Produce json
// @Param type path string true "Connector type (github, gmail, google_drive, google_calendar)"
// @Success 200 {object} connector.ConnectorInfo
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Router /v1/connectors/{type} [get]
func (h *ConnectorHandler) GetConnector(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		responses.HandleError(c, err, "failed to get user ID")
		return
	}

	connectorType := connector.ConnectorType(c.Param("type"))
	if !connectorType.IsValid() {
		responses.HandleError(c, platformerrors.NewError(
			c.Request.Context(),
			platformerrors.LayerHandler,
			platformerrors.ErrorTypeValidation,
			"invalid connector type",
			nil,
			"",
		), "invalid connector type")
		return
	}

	connectors, err := h.service.ListConnectors(c.Request.Context(), userID)
	if err != nil {
		responses.HandleError(c, err, "failed to get connector")
		return
	}

	for _, conn := range connectors {
		if conn.Type == connectorType {
			c.JSON(http.StatusOK, conn)
			return
		}
	}

	responses.HandleError(c, platformerrors.NewError(
		c.Request.Context(),
		platformerrors.LayerHandler,
		platformerrors.ErrorTypeNotFound,
		"connector not found",
		nil,
		"",
	), "connector not found")
}

// GetAuthURL godoc
// @Summary Get OAuth authorization URL
// @Description Initiates the OAuth flow and returns the authorization URL for the connector
// @Tags Connectors API
// @Security BearerAuth
// @Produce json
// @Param type path string true "Connector type (github, gmail, google_drive, google_calendar)"
// @Success 200 {object} AuthURLResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 409 {object} responses.ErrorResponse "Already connected"
// @Router /v1/connectors/{type}/auth-url [get]
func (h *ConnectorHandler) GetAuthURL(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		responses.HandleError(c, err, "failed to get user ID")
		return
	}

	connectorType := connector.ConnectorType(c.Param("type"))
	if !connectorType.IsValid() {
		responses.HandleError(c, platformerrors.NewError(
			c.Request.Context(),
			platformerrors.LayerHandler,
			platformerrors.ErrorTypeValidation,
			"invalid connector type",
			nil,
			"",
		), "invalid connector type")
		return
	}

	authURL, state, err := h.service.InitiateOAuth(c.Request.Context(), userID, connectorType)
	if err != nil {
		if errors.Is(err, connector.ErrAlreadyConnected) {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "already_connected",
				"message": "Connector is already connected. Disconnect first to reconnect.",
			})
			return
		}
		if errors.Is(err, connector.ErrConnectorNotEnabled) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "connector_not_enabled",
				"message": "This connector type is not enabled",
			})
			return
		}
		responses.HandleError(c, err, "failed to initiate OAuth")
		return
	}

	c.JSON(http.StatusOK, AuthURLResponse{
		AuthURL: authURL,
		State:   state,
	})
}

// HandleCallback godoc
// @Summary Handle OAuth callback
// @Description Handles the OAuth callback from the provider (redirected from authorization URL)
// @Tags Connectors API
// @Produce json
// @Param type path string true "Connector type"
// @Param code query string true "Authorization code"
// @Param state query string true "OAuth state parameter"
// @Success 302 "Redirect to frontend"
// @Failure 400 {object} responses.ErrorResponse
// @Router /v1/connectors/{type}/callback [get]
func (h *ConnectorHandler) HandleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_params",
			"message": "Missing code or state parameter",
		})
		return
	}

	// Error from provider
	if errParam := c.Query("error"); errParam != "" {
		errDesc := c.Query("error_description")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       errParam,
			"description": errDesc,
		})
		return
	}

	_, redirectURL, err := h.service.CompleteOAuth(c.Request.Context(), code, state)
	if err != nil {
		if errors.Is(err, connector.ErrOAuthStateNotFound) || errors.Is(err, connector.ErrOAuthStateExpired) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_state",
				"message": "OAuth state is invalid or expired. Please try again.",
			})
			return
		}
		responses.HandleError(c, err, "failed to complete OAuth")
		return
	}

	// Redirect to frontend
	c.Redirect(http.StatusFound, redirectURL)
}

// Connect godoc
// @Summary Connect using authorization code
// @Description Alternative to callback - complete OAuth flow with code and state
// @Tags Connectors API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param type path string true "Connector type"
// @Param request body ConnectRequest true "Authorization code and state"
// @Success 200 {object} ConnectResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Router /v1/connectors/{type}/connect [post]
func (h *ConnectorHandler) Connect(c *gin.Context) {
	var req ConnectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.HandleError(c, platformerrors.NewError(
			c.Request.Context(),
			platformerrors.LayerHandler,
			platformerrors.ErrorTypeValidation,
			"invalid request body",
			err,
			"",
		), "invalid request body")
		return
	}

	conn, _, err := h.service.CompleteOAuth(c.Request.Context(), req.Code, req.State)
	if err != nil {
		responses.HandleError(c, err, "failed to connect")
		return
	}

	c.JSON(http.StatusOK, ConnectResponse{
		Connected:    true,
		Username:     conn.ProviderUsername,
		Email:        conn.ProviderEmail,
		AvatarURL:    conn.ProviderAvatarURL,
		ConnectedAt:  conn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// Disconnect godoc
// @Summary Disconnect connector
// @Description Removes the connector connection and revokes OAuth tokens
// @Tags Connectors API
// @Security BearerAuth
// @Produce json
// @Param type path string true "Connector type"
// @Success 200 {object} DisconnectResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Router /v1/connectors/{type}/disconnect [delete]
func (h *ConnectorHandler) Disconnect(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		responses.HandleError(c, err, "failed to get user ID")
		return
	}

	connectorType := connector.ConnectorType(c.Param("type"))
	if !connectorType.IsValid() {
		responses.HandleError(c, platformerrors.NewError(
			c.Request.Context(),
			platformerrors.LayerHandler,
			platformerrors.ErrorTypeValidation,
			"invalid connector type",
			nil,
			"",
		), "invalid connector type")
		return
	}

	if err := h.service.Disconnect(c.Request.Context(), userID, connectorType); err != nil {
		if errors.Is(err, connector.ErrConnectorNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_connected",
				"message": "Connector is not connected",
			})
			return
		}
		responses.HandleError(c, err, "failed to disconnect")
		return
	}

	c.JSON(http.StatusOK, DisconnectResponse{
		Disconnected: true,
	})
}

// GetStatus godoc
// @Summary Get connector connection status
// @Description Returns the health and status of a connector connection
// @Tags Connectors API
// @Security BearerAuth
// @Produce json
// @Param type path string true "Connector type"
// @Success 200 {object} StatusResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Router /v1/connectors/{type}/status [get]
func (h *ConnectorHandler) GetStatus(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		responses.HandleError(c, err, "failed to get user ID")
		return
	}

	connectorType := connector.ConnectorType(c.Param("type"))
	if !connectorType.IsValid() {
		responses.HandleError(c, platformerrors.NewError(
			c.Request.Context(),
			platformerrors.LayerHandler,
			platformerrors.ErrorTypeValidation,
			"invalid connector type",
			nil,
			"",
		), "invalid connector type")
		return
	}

	conn, err := h.service.GetConnection(c.Request.Context(), userID, connectorType)
	if err != nil {
		responses.HandleError(c, err, "failed to get connection status")
		return
	}

	if conn == nil {
		c.JSON(http.StatusOK, StatusResponse{
			Connected: false,
			Enabled:   h.service.IsEnabled(connectorType),
		})
		return
	}

	status := "healthy"
	if conn.LastError != "" {
		status = "error"
	} else if conn.IsExpired() {
		status = "expired"
	} else if conn.NeedsRefresh() {
		status = "needs_refresh"
	}

	c.JSON(http.StatusOK, StatusResponse{
		Connected:    conn.IsConnected,
		Enabled:      h.service.IsEnabled(connectorType),
		Status:       status,
		Username:     conn.ProviderUsername,
		Email:        conn.ProviderEmail,
		LastError:    conn.LastError,
		LastSyncAt:   formatTimePtr(conn.LastSyncAt),
		ConnectedAt:  conn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// GetToken godoc
// @Summary Get decrypted access token for internal use
// @Description Returns the decrypted access token for the connector. For internal service use only.
// @Tags Connectors API
// @Security BearerAuth
// @Produce json
// @Param type path string true "Connector type"
// @Success 200 {object} TokenResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Router /v1/connectors/{type}/token [get]
func (h *ConnectorHandler) GetToken(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		responses.HandleError(c, err, "failed to get user ID")
		return
	}

	connectorType := connector.ConnectorType(c.Param("type"))
	if !connectorType.IsValid() {
		responses.HandleError(c, platformerrors.NewError(
			c.Request.Context(),
			platformerrors.LayerHandler,
			platformerrors.ErrorTypeValidation,
			"invalid connector type",
			nil,
			"",
		), "invalid connector type")
		return
	}

	accessToken, err := h.service.GetDecryptedAccessToken(c.Request.Context(), userID, connectorType)
	if err != nil {
		if errors.Is(err, connector.ErrConnectorNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_connected",
				"message": "Connector is not connected",
			})
			return
		}
		responses.HandleError(c, err, "failed to get token")
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	})
}

// TokenResponse is the response for getting a token.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// RefreshTokens godoc
// @Summary Force refresh OAuth tokens
// @Description Forces a token refresh for the connector
// @Tags Connectors API
// @Security BearerAuth
// @Produce json
// @Param type path string true "Connector type"
// @Success 200 {object} RefreshResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Router /v1/connectors/{type}/refresh [post]
func (h *ConnectorHandler) RefreshTokens(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		responses.HandleError(c, err, "failed to get user ID")
		return
	}

	connectorType := connector.ConnectorType(c.Param("type"))
	if !connectorType.IsValid() {
		responses.HandleError(c, platformerrors.NewError(
			c.Request.Context(),
			platformerrors.LayerHandler,
			platformerrors.ErrorTypeValidation,
			"invalid connector type",
			nil,
			"",
		), "invalid connector type")
		return
	}

	// Getting the decrypted token will trigger a refresh if needed
	_, err = h.service.GetDecryptedAccessToken(c.Request.Context(), userID, connectorType)
	if err != nil {
		if errors.Is(err, connector.ErrConnectorNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_connected",
				"message": "Connector is not connected",
			})
			return
		}
		responses.HandleError(c, err, "failed to refresh tokens")
		return
	}

	c.JSON(http.StatusOK, RefreshResponse{
		Refreshed: true,
	})
}

// getUserID extracts the user ID from the gin context using the auth handler.
func getUserID(c *gin.Context) (uint, error) {
	user, ok := authhandler.GetUserFromContext(c)
	if !ok || user == nil {
		return 0, platformerrors.NewError(
			c.Request.Context(),
			platformerrors.LayerHandler,
			platformerrors.ErrorTypeUnauthorized,
			"user not authenticated",
			nil,
			"",
		)
	}

	return user.ID, nil
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02T15:04:05Z07:00")
}

// Response types

// ListConnectorsResponse is the response for listing connectors.
type ListConnectorsResponse struct {
	Connectors []ConnectorInfoResponse `json:"connectors"`
}

// ConnectorInfoResponse includes connector metadata plus enabled status.
type ConnectorInfoResponse struct {
	Type        connector.ConnectorType `json:"type"`
	DisplayName string                  `json:"display_name"`
	Description string                  `json:"description"`
	IconURL     string                  `json:"icon_url"`
	IsConnected bool                    `json:"is_connected"`
	Username    string                  `json:"username,omitempty"`
	Email       string                  `json:"email,omitempty"`
	AvatarURL   string                  `json:"avatar_url,omitempty"`
	ConnectedAt *time.Time              `json:"connected_at,omitempty"`
	Scopes      []string                `json:"scopes,omitempty"`
	HasWrite    bool                    `json:"has_write"`
	Enabled     bool                    `json:"enabled"`
}

// AuthURLResponse is the response for getting an auth URL.
type AuthURLResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

// ConnectRequest is the request for connecting with code/state.
type ConnectRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// ConnectResponse is the response for a successful connection.
type ConnectResponse struct {
	Connected   bool   `json:"connected"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	ConnectedAt string `json:"connected_at"`
}

// DisconnectResponse is the response for disconnecting.
type DisconnectResponse struct {
	Disconnected bool `json:"disconnected"`
}

// StatusResponse is the response for connection status.
type StatusResponse struct {
	Connected   bool   `json:"connected"`
	Enabled     bool   `json:"enabled"`
	Status      string `json:"status,omitempty"`
	Username    string `json:"username,omitempty"`
	Email       string `json:"email,omitempty"`
	LastError   string `json:"last_error,omitempty"`
	LastSyncAt  string `json:"last_sync_at,omitempty"`
	ConnectedAt string `json:"connected_at,omitempty"`
}

// RefreshResponse is the response for token refresh.
type RefreshResponse struct {
	Refreshed bool `json:"refreshed"`
}
