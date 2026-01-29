package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	domainsession "jan-server/mono/apps/backend/internal/domain/session"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/responses"
	sessionres "jan-server/mono/apps/backend/internal/interfaces/httpserver/responses/session"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// RegisterRealtimeRoutes registers the realtime session routes.
func RegisterRealtimeRoutes(router gin.IRoutes, handler *handlers.SessionHandler) {
	// Session management endpoints
	router.POST("/realtime/sessions", createSession(handler))

	// Extension endpoints
	router.GET("/realtime/sessions", listSessions(handler))
	router.GET("/realtime/sessions/:id", getSession(handler))
	router.DELETE("/realtime/sessions/:id", deleteSession(handler))
}

// createSession godoc
// @Summary      Create a realtime session
// @Description  Creates a new realtime session with LiveKit token. No request body required.
// @Tags         Realtime API
// @Accept       json
// @Produce      json
// @Success      201 {object} sessionres.SessionResponse
// @Failure      401 {object} responses.ErrorResponse
// @Failure      500 {object} responses.ErrorResponse
// @Security     BearerAuth
// @Router       /realtime/sessions [post]
func createSession(handler *handlers.SessionHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := extractUserID(c)

		sess, err := handler.CreateSession(c.Request.Context(), &domainsession.CreateSessionRequest{}, userID)
		if err != nil {
			responses.HandleError(c, err, "failed to create session")
			return
		}

		c.JSON(http.StatusCreated, sessionres.NewSessionResponse(sess))
	}
}

// listSessions godoc
// @Summary      List realtime sessions
// @Description  Lists all active sessions for the current user
// @Tags         Realtime API
// @Produce      json
// @Success      200 {object} sessionres.ListSessionsResponse
// @Failure      401 {object} responses.ErrorResponse
// @Failure      500 {object} responses.ErrorResponse
// @Security     BearerAuth
// @Router       /realtime/sessions [get]
func listSessions(handler *handlers.SessionHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := extractUserID(c)

		sessions, err := handler.ListUserSessions(c.Request.Context(), userID)
		if err != nil {
			responses.HandleError(c, err, "failed to list sessions")
			return
		}

		c.JSON(http.StatusOK, sessionres.NewListSessionsResponse(sessions))
	}
}

// getSession godoc
// @Summary      Get a realtime session
// @Description  Retrieves a specific session by ID. Users can only access their own sessions.
// @Tags         Realtime API
// @Produce      json
// @Param        id path string true "Session ID"
// @Success      200 {object} sessionres.SessionResponse
// @Failure      403 {object} responses.ErrorResponse
// @Failure      404 {object} responses.ErrorResponse
// @Failure      500 {object} responses.ErrorResponse
// @Security     BearerAuth
// @Router       /realtime/sessions/{id} [get]
func getSession(handler *handlers.SessionHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := extractUserID(c)

		sess, err := handler.GetSession(c.Request.Context(), id)
		if err != nil {
			responses.HandleError(c, err, "failed to get session")
			return
		}

		// Authorization: verify session belongs to the authenticated user
		if sess.UserID != userID {
			responses.HandleNewError(c, platformerrors.ErrorTypeForbidden, "access denied", "realtime-auth-001")
			return
		}

		c.JSON(http.StatusOK, sessionres.NewSessionResponseForGet(sess))
	}
}

// deleteSession godoc
// @Summary      Delete a realtime session
// @Description  Ends a session and invalidates its token. Users can only delete their own sessions.
// @Tags         Realtime API
// @Produce      json
// @Param        id path string true "Session ID"
// @Success      200 {object} sessionres.DeleteSessionResponse
// @Failure      403 {object} responses.ErrorResponse
// @Failure      404 {object} responses.ErrorResponse
// @Failure      500 {object} responses.ErrorResponse
// @Security     BearerAuth
// @Router       /realtime/sessions/{id} [delete]
func deleteSession(handler *handlers.SessionHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := extractUserID(c)

		// First, get the session to verify ownership
		sess, err := handler.GetSession(c.Request.Context(), id)
		if err != nil {
			responses.HandleError(c, err, "failed to get session")
			return
		}

		// Authorization: verify session belongs to the authenticated user
		if sess.UserID != userID {
			responses.HandleNewError(c, platformerrors.ErrorTypeForbidden, "access denied", "realtime-auth-001")
			return
		}

		if err := handler.DeleteSession(c.Request.Context(), id); err != nil {
			responses.HandleError(c, err, "failed to delete session")
			return
		}

		c.JSON(http.StatusOK, sessionres.NewDeleteSessionResponse(id))
	}
}

// Helper functions

func extractUserID(c *gin.Context) string {
	// Check for user_id set directly by middleware (for Kong API key auth)
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok && id != "" {
			return id
		}
	}

	// Fall back to extracting from JWT token
	if token, exists := c.Get("auth_token"); exists {
		if jwtToken, ok := token.(*jwt.Token); ok {
			if claims, ok := jwtToken.Claims.(jwt.MapClaims); ok {
				if sub, ok := claims["sub"].(string); ok {
					return sub
				}
			}
		}
	}
	return "anonymous"
}
