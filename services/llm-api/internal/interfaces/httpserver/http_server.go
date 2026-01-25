package httpserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/domain/apikey"
	"jan-server/services/llm-api/internal/infrastructure"
	middleware "jan-server/services/llm-api/internal/interfaces/httpserver/middlewares"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/auth"
	v1 "jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1"

	"github.com/gin-gonic/gin"
	"github.com/janhq/jan-server/packages/go-common/analytics"
	analyticsMiddleware "github.com/janhq/jan-server/packages/go-common/analytics/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "jan-server/services/llm-api/docs/swagger"
)

// extractUserIDFromJWT extracts the user ID (subject) from a JWT token without validation.
// This is used for analytics purposes only - actual auth validation happens in auth middleware.
func extractUserIDFromJWT(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	// Check for Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	token := parts[1]

	// Skip API keys (sk_*)
	if strings.HasPrefix(token, "sk_") {
		return ""
	}

	// JWT has 3 parts: header.payload.signature
	jwtParts := strings.Split(token, ".")
	if len(jwtParts) != 3 {
		return ""
	}

	// Decode payload (second part) - handle both standard and raw URL encoding
	payload, err := base64.RawURLEncoding.DecodeString(jwtParts[1])
	if err != nil {
		// Try with padding added
		paddedPayload := jwtParts[1]
		if pad := len(paddedPayload) % 4; pad > 0 {
			paddedPayload += strings.Repeat("=", 4-pad)
		}
		payload, err = base64.URLEncoding.DecodeString(paddedPayload)
		if err != nil {
			return ""
		}
	}

	// Parse JSON to get 'sub' claim
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}

	if sub, ok := claims["sub"].(string); ok {
		return sub
	}

	return ""
}

type HTTPServer struct {
	engine        *gin.Engine
	infra         *infrastructure.Infrastructure
	v1Route       *v1.V1Route
	authRoute     *auth.AuthRoute
	config        *config.Config
	apiKeyService *apikey.Service
	tracker       analytics.Tracker
}

func (s *HTTPServer) bindSwagger() {
	g := s.engine.Group("/")

	// Serve swagger UI with custom URL pointing to combined swagger if available
	g.GET("/api/swagger/*any", func(c *gin.Context) {
		// If requesting doc.json, serve the combined version
		if c.Param("any") == "/doc.json" {
			ServeCombinedSwagger()(c)
			return
		}
		// Otherwise serve from swagger assets
		ginSwagger.WrapHandler(swaggerFiles.Handler)(c)
	})
}

func NewHttpServer(
	v1Route *v1.V1Route,
	authRoute *auth.AuthRoute,
	infra *infrastructure.Infrastructure,
	cfg *config.Config,
	apiKeyService *apikey.Service,
	tracker analytics.Tracker,
) *HTTPServer {
	gin.SetMode(gin.ReleaseMode)
	server := HTTPServer{
		engine:        gin.New(),
		infra:         infra,
		v1Route:       v1Route,
		authRoute:     authRoute,
		config:        cfg,
		apiKeyService: apiKeyService,
		tracker:       tracker,
	}
	server.engine.Use(middleware.RequestID())
	server.engine.Use(middleware.TracingMiddleware(cfg.ServiceName))
	server.engine.Use(middleware.LoggingMiddleware(infra.Logger))
	server.engine.Use(middleware.CORSMiddleware())
	server.engine.Use(middleware.MetricsMiddleware())

	// Analytics middleware - adds tracker and user context to requests
	server.engine.Use(analyticsMiddleware.Analytics(analyticsMiddleware.Config{
		Tracker:           tracker,
		TrackHTTPRequests: false,
		ExtractDistinctID: extractUserIDFromJWT,
	}))

	// Root health check (for backwards compatibility)
	server.engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	server.engine.GET("/readyz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	server.engine.GET("/healthcheck", func(c *gin.Context) {
		c.JSON(200, "ok")
	})

	// Prometheus metrics endpoint
	server.engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	server.bindSwagger()
	return &server
}

func (httpServer *HTTPServer) Run() error {
	// Public routes (no auth required)
	root := httpServer.engine.Group("/")

	// Protected routes (auth middleware applied)
	protected := httpServer.engine.Group("/")
	protected.Use(
		middleware.AuthMiddleware(httpServer.infra.KeycloakValidator, httpServer.apiKeyService, httpServer.infra.Logger, httpServer.config.Issuer),
		middleware.CORSMiddleware(),
	)

	// /llm prefixed routes (mirror behaviour for Kong proxy paths)
	llmRoot := httpServer.engine.Group("/llm")
	llmProtected := llmRoot.Group("/")
	llmProtected.Use(
		middleware.AuthMiddleware(httpServer.infra.KeycloakValidator, httpServer.apiKeyService, httpServer.infra.Logger, httpServer.config.Issuer),
		middleware.CORSMiddleware(),
	)

	// Register auth routes (passes both public and protected routers)
	httpServer.authRoute.RegisterRouter(root, protected)
	httpServer.authRoute.RegisterRouter(llmRoot, llmProtected)

	// Register v1 routes (with auth middleware)
	httpServer.v1Route.RegisterRouter(protected)
	httpServer.v1Route.RegisterRouter(llmProtected)
	httpServer.v1Route.RegisterPublicRouter(root)

	if err := httpServer.engine.Run(fmt.Sprintf(":%d", httpServer.config.HTTPPort)); err != nil {
		return err
	}
	return nil
}
