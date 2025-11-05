package httpserver

import (
	"fmt"
	"net/http"

	"jan-server/services/llm-api/internal/config"
	"jan-server/services/llm-api/internal/infrastructure"
	middleware "jan-server/services/llm-api/internal/interfaces/httpserver/middlewares"
	"jan-server/services/llm-api/internal/interfaces/httpserver/routes/auth"
	v1 "jan-server/services/llm-api/internal/interfaces/httpserver/routes/v1"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "jan-server/services/llm-api/swagger"
)

type HttpServer struct {
	engine    *gin.Engine
	infra     *infrastructure.Infrastructure
	v1Route   *v1.V1Route
	authRoute *auth.AuthRoute
	config    *config.Config
}

func (s *HttpServer) bindSwagger() {
	g := s.engine.Group("/")
	g.GET("/api/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func NewHttpServer(
	v1Route *v1.V1Route,
	authRoute *auth.AuthRoute,
	infra *infrastructure.Infrastructure,
	cfg *config.Config,
) *HttpServer {
	gin.SetMode(gin.ReleaseMode)
	server := HttpServer{
		gin.New(),
		infra,
		v1Route,
		authRoute,
		cfg,
	}
	server.engine.Use(middleware.RequestID())
	server.engine.Use(middleware.TracingMiddleware(cfg.ServiceName))
	server.engine.Use(middleware.LoggingMiddleware(infra.Logger))
	server.engine.Use(middleware.CORSMiddleware())

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

	server.bindSwagger()
	return &server
}

func (httpServer *HttpServer) Run() error {
	// Public routes (no auth required)
	root := httpServer.engine.Group("/")

	// Protected routes (auth middleware applied)
	protected := httpServer.engine.Group("/")
	protected.Use(middleware.AuthMiddleware(httpServer.infra.KeycloakValidator, httpServer.infra.Logger), middleware.CORSMiddleware())

	// Register auth routes (passes both public and protected routers)
	httpServer.authRoute.RegisterRouter(root, protected)

	// Register v1 routes (with auth middleware)
	httpServer.v1Route.RegisterRouter(protected)

	if err := httpServer.engine.Run(fmt.Sprintf(":%d", httpServer.config.HTTPPort)); err != nil {
		return err
	}
	return nil
}
