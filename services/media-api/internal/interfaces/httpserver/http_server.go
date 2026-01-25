package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/janhq/jan-server/packages/go-common/analytics"

	mediaapidocs "jan-server/services/media-api/docs/swagger"
	"jan-server/services/media-api/internal/config"
	domain "jan-server/services/media-api/internal/domain/media"
	"jan-server/services/media-api/internal/infrastructure/auth"
	"jan-server/services/media-api/internal/interfaces/httpserver/handlers"
	v1 "jan-server/services/media-api/internal/interfaces/httpserver/routes/v1"
)

// HTTPServer wraps the gin engine with graceful shutdown helpers.
type HTTPServer struct {
	cfg     *config.Config
	engine  *gin.Engine
	log     zerolog.Logger
	auth    *auth.Validator
	tracker analytics.Tracker
}

// New constructs the HTTP server with default middleware and routes.
func New(cfg *config.Config, log zerolog.Logger, mediaService *domain.Service, authValidator *auth.Validator, tracker analytics.Tracker) *HTTPServer {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	mediaapidocs.SwaggerInfo.BasePath = "/"

	engine := gin.New()
	engine.Use(gin.Recovery(), gin.Logger())

	handlerProvider := handlers.NewProvider(cfg, mediaService, log)
	routeProvider := v1.NewRoutes(handlerProvider, cfg)

	// Register public routes (health checks, swagger) without authentication
	registerPublicRoutes(engine, cfg, authValidator)

	// Register public media serving endpoint (no auth required for img src usage)
	engine.GET("/api/media/:id", handlerProvider.Media.PublicServe)

	// Apply authentication middleware before protected routes
	if authValidator != nil {
		engine.Use(authValidator.Middleware())
	}

	// Register protected API routes
	routeProvider.Register(engine.Group("/"))

	return &HTTPServer{
		cfg:     cfg,
		engine:  engine,
		log:     log,
		auth:    authValidator,
		tracker: tracker,
	}
}

// Run starts the HTTP listener and handles graceful shutdown via context cancellation.
func (s *HTTPServer) Run(ctx context.Context) error {
	server := &http.Server{
		Addr:    s.cfg.Addr(),
		Handler: s.engine,
	}

	errCh := make(chan error, 1)
	go func() {
		s.log.Info().Str("addr", s.cfg.Addr()).Msg("media-api HTTP server listening")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		s.log.Info().Msg("context cancelled, shutting down HTTP server")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func registerPublicRoutes(engine *gin.Engine, cfg *config.Config, authValidator *auth.Validator) {
	engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"service": cfg.ServiceName, "status": "ok"})
	})
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	engine.GET("/readyz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	engine.GET("/health/auth", func(c *gin.Context) {
		if authValidator == nil || authValidator.Ready() {
			c.JSON(http.StatusOK, gin.H{"status": "ready"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "initializing"})
	})

	// Prometheus metrics endpoint
	engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
