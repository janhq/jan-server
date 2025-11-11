package repository

import (
	"jan-server/services/llm-api/internal/infrastructure/database/repository/apikeyrepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/conversationrepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/modelrepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/projectrepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/userrepo"

	"github.com/google/wire"
)

var RepositoryProvider = wire.NewSet(
	conversationrepo.NewConversationGormRepository,
	projectrepo.NewProjectGormRepository,
	modelrepo.NewProviderGormRepository,
	modelrepo.NewProviderModelGormRepository,
	modelrepo.NewModelCatalogGormRepository,
	userrepo.NewUserGormRepository,
	apikeyrepo.NewAPIKeyRepository,
)
