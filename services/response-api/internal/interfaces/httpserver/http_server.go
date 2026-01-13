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

	responseapidocs "jan-server/services/response-api/docs/swagger"
	"jan-server/services/response-api/internal/config"
	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/domain/plan"
	domain "jan-server/services/response-api/internal/domain/response"
	"jan-server/services/response-api/internal/infrastructure/auth"
	"jan-server/services/response-api/internal/interfaces/httpserver/handlers"
	"jan-server/services/response-api/internal/interfaces/httpserver/routes"
)

// HTTPServer wraps the gin engine with graceful shutdown helpers.
type HTTPServer struct {
	cfg         *config.Config
	engine      *gin.Engine
	log         zerolog.Logger
	handlerProv *handlers.Provider
	routeProv   *routes.Provider
	auth        *auth.Validator
}

// New constructs the HTTP server with default middleware and routes.
func New(
	cfg *config.Config,
	log zerolog.Logger,
	responseService domain.Service,
	planService plan.Service,
	artifactService artifact.Service,
	authValidator *auth.Validator,
) *HTTPServer {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	responseapidocs.SwaggerInfo.BasePath = "/"

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	handlerProvider := handlers.NewProviderWithPlanAndArtifact(responseService, planService, artifactService, log)
	routeProvider := routes.NewProvider(handlerProvider)

	// Register public routes (health checks, swagger) without authentication
	registerPublicRoutes(engine, cfg, authValidator)

	// Apply authentication middleware before protected routes
	if authValidator != nil {
		engine.Use(authValidator.Middleware())
	}

	// Register protected API routes
	routeProvider.Register(engine)

	return &HTTPServer{
		cfg:         cfg,
		engine:      engine,
		log:         log,
		handlerProv: handlerProvider,
		routeProv:   routeProvider,
		auth:        authValidator,
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
		s.log.Info().Str("addr", s.cfg.Addr()).Msg("HTTP server listening")
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error().Err(err).Msg("HTTP server error")
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		s.log.Info().Msg("Context cancelled, shutting down HTTP server")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return nil
}

func registerPublicRoutes(engine *gin.Engine, cfg *config.Config, authValidator *auth.Validator) {
	engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": cfg.ServiceName,
			"status":  "ok",
		})
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
