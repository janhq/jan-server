//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"

	"jan-server/mono/apps/backend/internal/domain"
	"jan-server/mono/apps/backend/internal/infrastructure"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/routes"
)

// InitializeHTTPServer creates a fully-wired HTTPServer with all dependencies.
// The cleanup function can be used to gracefully shutdown resources.
func InitializeHTTPServer() (*httpserver.HTTPServer, func(), error) {
	panic(wire.Build(
		// Infrastructure layer (config, DB, repositories, keycloak, etc.)
		infrastructure.InfrastructureProvider,

		// Domain services layer
		domain.ServiceProvider,

		// Route and handler providers
		routes.RouteProvider,

		// HTTP server
		httpserver.NewHttpServer,
	))
}
