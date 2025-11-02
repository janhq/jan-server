package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	gormlogger "gorm.io/gorm/logger"

	"jan-server/services/llm-api/config"
	"jan-server/services/llm-api/domain"
	domainmodel "jan-server/services/llm-api/domain/model"
	"jan-server/services/llm-api/infrastructure/auth"
	infraconfig "jan-server/services/llm-api/infrastructure/config"
	"jan-server/services/llm-api/infrastructure/db"
	"jan-server/services/llm-api/infrastructure/idempotency"
	"jan-server/services/llm-api/infrastructure/keycloak"
	"jan-server/services/llm-api/infrastructure/logger"
	"jan-server/services/llm-api/infrastructure/observability"
	"jan-server/services/llm-api/infrastructure/provider"
	"jan-server/services/llm-api/infrastructure/repo"
	"jan-server/services/llm-api/interfaces/httpserver/handlers/chathandler"
	"jan-server/services/llm-api/interfaces/httpserver/handlers/conversationhandler"
	"jan-server/services/llm-api/interfaces/httpserver/handlers/guestauth"
	"jan-server/services/llm-api/interfaces/httpserver/handlers/modelhandler"
	"jan-server/services/llm-api/interfaces/httpserver/middlewares"
	v1routes "jan-server/services/llm-api/interfaces/httpserver/routes/v1"
	adminroutes "jan-server/services/llm-api/interfaces/httpserver/routes/v1/admin"
	adminmodelroutes "jan-server/services/llm-api/interfaces/httpserver/routes/v1/admin/model"
	adminproviderroutes "jan-server/services/llm-api/interfaces/httpserver/routes/v1/admin/provider"
	chatroutes "jan-server/services/llm-api/interfaces/httpserver/routes/v1/chat"
	conversationroutes "jan-server/services/llm-api/interfaces/httpserver/routes/v1/conversation"
	modelroutes "jan-server/services/llm-api/interfaces/httpserver/routes/v1/model"
	"jan-server/services/llm-api/swagger"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := infraconfig.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}

	jwksURL, err := cfg.ResolveJWKSURL(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("resolve jwks url")
	}

	otelShutdown, err := observability.Setup(ctx, cfg, log)
	if err != nil {
		log.Error().Err(err).Msg("initialize observability")
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := otelShutdown(shutdownCtx); err != nil {
				log.Error().Err(err).Msg("shutdown telemetry")
			}
		}()
	}

	gormDB, err := db.Connect(db.Config{
		DatabaseURL: cfg.DatabaseURL,
		MaxIdle:     10,
		MaxOpen:     25,
		MaxLifetime: time.Hour,
		LogLevel:    gormlogger.Warn,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("connect database")
	}
	sqlDB, _ := gormDB.DB()
	defer sqlDB.Close()

	// Run database migrations (embedded SQL migrations)
	if cfg.AutoMigrate {
		log.Info().Msg("applying database migrations")
		if err := db.AutoMigrate(gormDB); err != nil {
			log.Fatal().Err(err).Msg("auto migrate database")
		}
		log.Info().Msg("database migrations applied")
	}

	// Initialize providers from YAML configuration
	providers, err := config.LoadProvidersFromYAML("config/providers.yml")
	if err != nil {
		log.Warn().Err(err).Msg("load providers from yaml - continuing with empty provider list")
		providers = []*domainmodel.Provider{} // Continue with empty list if YAML fails
	}

	// Create provider repository and initialize providers
	providerRepo := repo.NewProviderRepository(gormDB)
	for _, p := range providers {
		// Check if provider already exists
		existing, err := providerRepo.FindByPublicID(ctx, p.PublicID)
		if err == nil && existing != nil {
			log.Info().Str("provider_id", p.PublicID).Msg("provider already exists, skipping")
			continue
		}
		// Create new provider
		if err := providerRepo.Create(ctx, p); err != nil {
			log.Warn().Err(err).Str("provider_id", p.PublicID).Msg("failed to create provider")
		} else {
			log.Info().Str("provider_id", p.PublicID).Str("display_name", p.DisplayName).Msg("initialized provider")
		}
	}

	httpClient := &http.Client{Timeout: cfg.HTTPTimeout}
	registry, err := provider.LoadRegistry(ctx, cfg.ProvidersConfigPath, log, httpClient)
	if err != nil {
		log.Fatal().Err(err).Msg("load providers")
	}

	modelRepo := repo.NewModelRepository(gormDB)
	for _, modelCfg := range registry.Models() {
		route, err := registry.Resolve(modelCfg.ID)
		if err != nil {
			log.Warn().Str("model", modelCfg.ID).Err(err).Msg("resolve model during bootstrap")
			continue
		}
		if err := modelRepo.Upsert(ctx, &domain.Model{
			ID:           modelCfg.ID,
			Provider:     route.Provider.Name(),
			DisplayName:  modelCfg.ID,
			Family:       route.Provider.Name(),
			Capabilities: modelCfg.Capabilities,
			Active:       true,
		}); err != nil {
			log.Warn().Err(err).Str("model", modelCfg.ID).Msg("upsert model metadata")
		}
	}

	idempotencyStore := idempotency.NewStore(gormDB)
	conversationRepo := repo.NewConversationRepository(gormDB)
	messageRepo := repo.NewMessageRepository(gormDB)

	keycloakClient := keycloak.NewClient(
		cfg.KeycloakBaseURL,
		cfg.KeycloakRealm,
		cfg.BackendClientID,
		cfg.BackendClientSecret,
		cfg.TargetClientID,
		cfg.GuestRole,
		httpClient,
		log,
		cfg.KeycloakAdminUser,
		cfg.KeycloakAdminPass,
		cfg.KeycloakAdminRealm,
		cfg.KeycloakAdminClient,
		cfg.KeycloakAdminSecret,
	)

	validator, err := auth.NewKeycloakValidator(ctx, jwksURL, cfg.Issuer, cfg.Audience, cfg.RefreshJWKSInterval, log)
	if err != nil {
		log.Fatal().Err(err).Msg("initialize jwt validator")
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middlewares.RequestID())

	readyCheck := func() error {
		if err := sqlDB.PingContext(context.Background()); err != nil {
			return err
		}
		if !validator.Ready() {
			return errors.New("jwks not ready")
		}
		return nil
	}

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/readyz", func(c *gin.Context) {
		if err := readyCheck(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	modelsHandler := modelhandler.NewModelsHandler(registry, modelRepo, log)
	chatHandler := chathandler.NewChatHandler(registry, idempotencyStore, conversationRepo, log)
	convHandler := conversationhandler.NewConversationsHandler(conversationRepo, messageRepo, log)
	guestHandler := guestauth.NewGuestHandler(keycloakClient, log)
	upgradeHandler := guestauth.NewUpgradeHandler(keycloakClient, log)

	router.POST("/auth/guest", guestHandler.CreateGuest)

	// Initialize new route structure
	modelRoute := modelroutes.NewModelRoute(modelsHandler)
	chatRoute := chatroutes.NewChatRoute(chatHandler)
	conversationRoute := conversationroutes.NewConversationRoute(convHandler)
	adminModelRoute := adminmodelroutes.NewAdminModelRoute()
	adminProviderRoute := adminproviderroutes.NewAdminProviderRoute()
	adminRoute := adminroutes.NewAdminRoute(adminModelRoute, adminProviderRoute)
	v1Route := v1routes.NewV1Route(modelRoute, chatRoute, conversationRoute, adminRoute)

	protected := router.Group("/")
	protected.Use(middlewares.AuthMiddleware(validator, log))

	protected.POST("/auth/upgrade", upgradeHandler.Upgrade)

	// Register v1 routes
	v1Route.RegisterRouter(protected)

	swagger.Register(router)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("graceful shutdown failed")
		}
	}()

	log.Info().Int("port", cfg.HTTPPort).Msg("llm-api listening")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("http server failed")
	}
}
