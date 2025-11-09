package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	templateapidocs "jan-server/services/template-api/docs/swagger"
	"jan-server/services/template-api/internal/config"
	domain "jan-server/services/template-api/internal/domain/sample"
	"jan-server/services/template-api/internal/interfaces/httpserver/handlers"
	"jan-server/services/template-api/internal/interfaces/httpserver/routes"
)

// HttpServer wraps the gin engine with graceful shutdown helpers.
type HttpServer struct {
	cfg         *config.Config
	engine      *gin.Engine
	log         zerolog.Logger
	handlerProv *handlers.Provider
	routeProv   *routes.Provider
}

// New constructs the HTTP server with default middleware and routes.
func New(cfg *config.Config, log zerolog.Logger, sampleService domain.Service) *HttpServer {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	templateapidocs.SwaggerInfo.BasePath = "/"

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())
	handlerProvider := handlers.NewProvider(sampleService)
	routeProvider := routes.NewProvider(handlerProvider)
	registerCoreRoutes(engine, cfg, routeProvider)

	return &HttpServer{
		cfg:         cfg,
		engine:      engine,
		log:         log,
		handlerProv: handlerProvider,
		routeProv:   routeProvider,
	}
}

// Run starts the HTTP listener and handles graceful shutdown via context cancellation.
func (s *HttpServer) Run(ctx context.Context) error {
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

func registerCoreRoutes(engine *gin.Engine, cfg *config.Config, routeProvider *routes.Provider) {
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

	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	routeProvider.Register(engine)
}
