package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"jan-server/mono/apps/backend/internal/infrastructure/config"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Server represents the HTTP server.
type Server struct {
	cfg        *config.Config
	db         *gorm.DB
	router     *gin.Engine
	httpServer *http.Server
}

// NewServer creates a new HTTP server instance.
func NewServer(cfg *config.Config, db *gorm.DB) *Server {
	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Apply global middlewares
	router.Use(gin.Recovery())
	router.Use(middlewares.RequestID())
	router.Use(middlewares.Logger())
	router.Use(middlewares.CORS(cfg))

	server := &Server{
		cfg:    cfg,
		db:     db,
		router: router,
	}

	// Register all routes
	server.registerRoutes()

	return server
}

func (s *Server) registerRoutes() {
	// Health check endpoints (no auth required)
	s.router.GET("/healthz", routes.HealthCheck)
	s.router.GET("/readyz", routes.ReadyCheck(s.db))

	// API version group
	v1 := s.router.Group("/v1")

	// Auth routes (public)
	routes.RegisterAuthRoutes(v1, s.cfg, s.db)

	// Protected routes (require authentication)
	protected := v1.Group("")
	protected.Use(middlewares.Auth(s.cfg, s.db))

	// Protected auth routes (me, api-keys, change-password)
	routes.RegisterProtectedAuthRoutes(protected, s.cfg, s.db)

	// LLM API routes
	routes.RegisterChatRoutes(protected, s.cfg, s.db)
	routes.RegisterConversationRoutes(protected, s.cfg, s.db)
	routes.RegisterModelRoutes(protected, s.cfg, s.db)
	routes.RegisterProviderRoutes(protected, s.cfg, s.db)
	routes.RegisterMessageRoutes(protected, s.cfg, s.db)
	routes.RegisterConnectorRoutes(protected, s.cfg, s.db)

	// Response API routes
	routes.RegisterResponseRoutes(protected, s.cfg, s.db)
	routes.RegisterArtifactRoutes(protected, s.cfg, s.db)
	routes.RegisterAgentRoutes(protected, s.cfg, s.db)

	// Media API routes
	routes.RegisterMediaRoutes(protected, s.cfg, s.db)

	// Memory routes (if enabled)
	if s.cfg.MemoryEnabled {
		routes.RegisterMemoryRoutes(protected, s.cfg, s.db)
	}

	// Realtime routes (if enabled)
	if s.cfg.RealtimeEnabled {
		routes.RegisterRealtimeRoutes(protected, s.cfg, s.db)
	}

	// MCP endpoint
	routes.RegisterMCPRoutes(s.router, s.cfg, s.db)

	// Admin routes
	admin := v1.Group("/admin")
	admin.Use(middlewares.Auth(s.cfg, s.db))
	admin.Use(middlewares.RequireAdmin())
	routes.RegisterAdminRoutes(admin, s.cfg, s.db)

	// Public share routes (no auth)
	routes.RegisterShareRoutes(s.router, s.cfg, s.db)

	// Swagger docs (if enabled)
	if s.cfg.EnableSwagger {
		routes.RegisterSwaggerRoutes(s.router)
	}

	log.Info().Msg("all routes registered")
}

// Run starts the HTTP server.
func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.cfg.HTTPPort)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: s.cfg.StreamTimeout + 10*time.Second, // Allow for streaming
		IdleTimeout:  120 * time.Second,
	}

	log.Info().
		Int("port", s.cfg.HTTPPort).
		Msg("starting HTTP server")

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	log.Info().Msg("shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}

// Router returns the underlying Gin router for testing.
func (s *Server) Router() *gin.Engine {
	return s.router
}
