package httpserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"menlo.ai/menlo-platform/config"
	"menlo.ai/menlo-platform/internal/infrastructure"
	middleware "menlo.ai/menlo-platform/internal/interfaces/httpserver/middlewares"
	v1 "menlo.ai/menlo-platform/internal/interfaces/httpserver/routes/v1"

	_ "menlo.ai/menlo-platform/swagger"
)

type HttpServer struct {
	engine  *gin.Engine
	infra   *infrastructure.Infrastructure
	v1Route *v1.V1Route
}

func (s *HttpServer) bindSwagger() {
	g := s.engine.Group("/")
	g.GET("/api/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func (s *HttpServer) bindDev() {
	g := s.engine.Group("/")
	g.GET("/auth/googletest", func(c *gin.Context) {
		code := c.Query("code")
		state := c.Query("state")
		curlCommand := fmt.Sprintf(`curl --request POST \
  --url 'http://localhost:8080/v1/auth/google/callback' \
  --header 'Content-Type: application/json' \
  --cookie 'menlo_platform_oauth_state=%s' \
  --data '{"code": "%s", "state": "%s"}'`, state, code, state)
		c.String(http.StatusOK, curlCommand)
	})
}

func NewHttpServer(v1Route *v1.V1Route, infra *infrastructure.Infrastructure) *HttpServer {
	gin.SetMode(gin.ReleaseMode)
	server := HttpServer{
		gin.New(),
		infra,
		v1Route,
	}
	server.engine.Use(middleware.CORS())
	server.engine.Use(middleware.TransactionMiddleware(infra.DB))
	server.engine.Use(middleware.LoggerMiddleware())
	server.engine.Use(middleware.EndpointEventPublisherMiddleware(infra.MQ))
	server.engine.GET("/healthcheck", func(c *gin.Context) {
		c.JSON(200, "ok")
	})
	server.bindSwagger()
	if config.IsDev() {
		server.bindDev()
	}
	return &server
}

func (httpServer *HttpServer) Run() error {
	port := 8080
	root := httpServer.engine.Group("/")
	httpServer.v1Route.RegisterRouter(root)
	if err := httpServer.engine.Run(fmt.Sprintf(":%d", port)); err != nil {
		return err
	}
	return nil
}
