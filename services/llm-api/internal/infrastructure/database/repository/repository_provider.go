package repository

import (
	"jan-server/services/llm-api/internal/infrastructure/database/repository/apikeyrepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/conversationrepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/documentrepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/mcptoolrepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/modelrepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/modelprompttemplaterepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/projectrepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/prompttemplaterepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/sharerepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/tokenusagerepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/userrepo"
	"jan-server/services/llm-api/internal/infrastructure/database/repository/usersettingsrepo"

	"github.com/google/wire"
)

var RepositoryProvider = wire.NewSet(
	conversationrepo.NewConversationGormRepository,
	conversationrepo.NewItemGormRepository,
	projectrepo.NewProjectGormRepository,
	modelrepo.NewProviderGormRepository,
	modelrepo.NewProviderModelGormRepository,
	modelrepo.NewModelCatalogGormRepository,
	userrepo.NewUserGormRepository,
	apikeyrepo.NewAPIKeyRepository,
	usersettingsrepo.NewUserSettingsGormRepository,
	prompttemplaterepo.NewPromptTemplateGormRepository,
	modelprompttemplaterepo.NewModelPromptTemplateGormRepository,
	sharerepo.NewShareGormRepository,
	mcptoolrepo.NewMCPToolGormRepository,
	tokenusagerepo.NewTokenUsageGormRepository,
	documentrepo.NewDocumentContentGormRepository,
	documentrepo.NewProjectFileGormRepository,
)
