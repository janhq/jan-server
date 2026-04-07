package model

import (
	modelHandler "jan-server/services/llm-api/internal/interfaces/httpserver/handlers/modelhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/modelprompthandler"

	"jan-server/services/llm-api/internal/interfaces/httpserver/requests"
	requestmodels "jan-server/services/llm-api/internal/interfaces/httpserver/requests/models"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses"
	"jan-server/services/llm-api/internal/utils/platformerrors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const HeaderIncludeProviderData = "X-PROVIDER-DATA"
const MaxExceptModelsLimit = 1000

type AdminModelRoute struct {
	modelHandler               *modelHandler.ModelHandler
	modelCatalogHandler        *modelHandler.ModelCatalogHandler
	providerModelHandler       *modelHandler.ProviderModelHandler
	modelPromptTemplateHandler *modelprompthandler.ModelPromptTemplateHandler
}

func NewAdminModelRoute(
	modelHandler *modelHandler.ModelHandler,
	modelCatalogHandler *modelHandler.ModelCatalogHandler,
	providerModelHandler *modelHandler.ProviderModelHandler,
	modelPromptTemplateHandler *modelprompthandler.ModelPromptTemplateHandler,
) *AdminModelRoute {
	return &AdminModelRoute{
		modelHandler:               modelHandler,
		modelCatalogHandler:        modelCatalogHandler,
		providerModelHandler:       providerModelHandler,
		modelPromptTemplateHandler: modelPromptTemplateHandler,
	}
}

func (route *AdminModelRoute) RegisterRouter(router *gin.RouterGroup) {
	modelsRoute := router.Group("models")

	// Model Catalog endpoints
	catalogRoute := modelsRoute.Group("catalogs")
	catalogRoute.GET("", route.ListModelCatalogs)
	catalogRoute.POST("/bulk-toggle", route.BulkToggleModelCatalogs)

	// Model Prompt Template endpoints - using dedicated sub-routes with action prefix
	// Format: /prompt-templates/list/*model_id, /prompt-templates/assign/*model_id, etc.
	promptTemplatesRoute := modelsRoute.Group("prompt-templates")
	promptTemplatesRoute.GET("/list/*model_id", route.modelPromptTemplateHandler.List)
	promptTemplatesRoute.POST("/assign/*model_id", route.modelPromptTemplateHandler.Assign)
	promptTemplatesRoute.GET("/effective/*model_id", route.modelPromptTemplateHandler.GetEffective)
	promptTemplatesRoute.PATCH("/update/:template_key/*model_id", route.modelPromptTemplateHandler.Update)
	promptTemplatesRoute.DELETE("/unassign/:template_key/*model_id", route.modelPromptTemplateHandler.Unassign)

	// Model Catalog detail endpoints (wildcard for IDs with slashes)
	catalogRoute.GET("/*model_public_id", route.GetModelCatalog)
	catalogRoute.PATCH("/*model_public_id", route.UpdateModelCatalog)

	// Provider Model endpoints
	providerModelsRoute := modelsRoute.Group("provider-models")
	providerModelsRoute.GET("", route.ListProviderModels)
	providerModelsRoute.GET("/:provider_model_public_id", route.GetProviderModel)
	providerModelsRoute.PATCH("/:provider_model_public_id", route.UpdateProviderModel)
	providerModelsRoute.POST("/bulk-toggle", route.BulkToggleProviderModels)
}

// ListModelCatalogs
// @Summary List all model catalogs
// @Description Retrieves a paginated list of model catalogs with optional filtering and searching
// @Tags Admin Model API
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Number of records to return (default: 20, max: 100)"
// @Param offset query int false "Number of records to skip for pagination"
// @Param order query string false "Sort order: asc or desc (default: desc)"
// @Param status query string false "Filter by status: init, filled, updated"
// @Param is_moderated query bool false "Filter by moderation status"
// @Success 200 {object} modelresponses.ModelCatalogResponse "List of model catalogs"
// @Failure 400 {object} responses.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/admin/models/catalogs [get]
func (route *AdminModelRoute) ListModelCatalogs(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	pagination, err := requests.GetPaginationFromQuery(reqCtx)
	if err != nil {
		responses.HandleError(reqCtx, err, "Invalid pagination parameters")
		return
	}

	filter := route.buildModelCatalogFilter(reqCtx)

	catalogs, total, err := route.modelCatalogHandler.ListCatalogs(ctx, filter, pagination)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to retrieve model catalogs")
		return
	}

	reqCtx.JSON(http.StatusOK, gin.H{
		"data":  catalogs,
		"total": total,
		"limit": pagination.Limit,
	})
}

// GetModelCatalog
// @Summary Get a model catalog entry
// @Description Retrieves detailed information about a model catalog entry by its public ID (supports IDs with slashes)
// @Tags Admin Model API
// @Security BearerAuth
// @Produce json
// @Param model_public_id path string true "Model Catalog Public ID (can contain slashes)"
// @Success 200 {object} modelresponses.ModelCatalogResponse "Model catalog details"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 404 {object} responses.ErrorResponse "Model catalog not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/admin/models/catalogs/{model_public_id} [get]
func (route *AdminModelRoute) GetModelCatalog(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	publicID := strings.TrimPrefix(reqCtx.Param("model_public_id"), "/")

	catalog, err := route.modelCatalogHandler.GetCatalog(ctx, publicID)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to retrieve model catalog")
		return
	}

	reqCtx.JSON(http.StatusOK, catalog)
}

// UpdateModelCatalog
// @Summary Update a model catalog entry
// @Description Updates metadata for a model catalog entry. Marks it as manually updated to prevent auto-sync overwrites.
// @Tags Admin Model API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param model_public_id path string true "Model Catalog Public ID (can contain slashes)"
// @Param payload body requestmodels.UpdateModelCatalogRequest true "Update payload"
// @Success 200 {object} modelresponses.ModelCatalogResponse "Updated model catalog"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload"
// @Failure 404 {object} responses.ErrorResponse "Model catalog not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/admin/models/catalogs/{model_public_id} [patch]
func (route *AdminModelRoute) UpdateModelCatalog(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	publicID := strings.TrimPrefix(reqCtx.Param("model_public_id"), "/")

	var request requestmodels.UpdateModelCatalogRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		responses.HandleError(reqCtx, err, "Invalid request body")
		return
	}

	catalog, err := route.modelCatalogHandler.UpdateCatalog(ctx, publicID, request)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to update model catalog")
		return
	}

	reqCtx.JSON(http.StatusOK, catalog)
}

// BulkToggleCatalogs performs bulk enable/disable operations on provider models
// associated with model catalogs.
//
// This operation supports two modes:
//
// Mode 1: Specific Catalogs (catalog_ids provided)
//  1. Looks up all specified catalog IDs and validates they exist
//  2. Queries all provider models associated with those catalogs
//  3. Filters out models in the exception list
//  4. Skips models already in the desired state (optimization)
//  5. Updates remaining models and tracks success/failure metrics
//  6. Returns detailed results including counts and any failures
//
// Mode 2: All Catalogs (catalog_ids empty or omitted)
//  1. Queries ALL model catalogs in the system
//  2. Queries all provider models for all those catalogs
//  3. Filters out models in the exception list
//  4. Skips models already in the desired state (optimization)
//  5. Updates remaining models and tracks success/failure metrics
//  6. Returns detailed results including counts and any failures
//
// The operation is designed to be fault-tolerant: if individual model updates fail,
// the operation continues and reports the failures in the response.
//
// Supported patterns:
//   - Enable all models in specific catalogs: {"enable": true, "catalog_ids": ["cat1", "cat2"]}
//   - Disable all models in specific catalogs: {"enable": false, "catalog_ids": ["cat1", "cat2"]}
//   - Enable all catalog models except some: {"enable": true, "catalog_ids": ["cat1"], "except_models": ["model1"]}
//   - Disable all catalog models except some: {"enable": false, "catalog_ids": ["cat1"], "except_models": ["model1"]}
//   - Enable ALL catalog models globally: {"enable": true}
//   - Disable ALL catalog models globally: {"enable": false}
//   - Disable ALL catalog models except specific ones: {"enable": false, "except_models": ["model1", "model2", "model3"]}
//   - Enable ALL catalog models except specific ones: {"enable": true, "except_models": ["model1", "model2", "model3"]}
//
// Example use cases:
//   - "Enable all GPT-4 models except GPT-4-vision" (provide GPT-4 catalog ID)
//   - "Disable all Claude models except Claude-3-Opus" (provide Claude catalog IDs)
//   - "Disable ALL catalog models except 3 specific ones" (no catalog_ids, use except_models)
//   - "Enable all models in the system" (no catalog_ids, no except_models)
//
// @Summary Bulk enable/disable provider models by catalog IDs or all catalogs
// @Description Enable or disable provider models for specific catalogs or ALL catalogs, with optional exception list. Supports "enable/disable all except" patterns globally or scoped to catalogs.
// @Tags Admin Model API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body requestmodels.BulkToggleCatalogsRequest true "Bulk toggle request. If catalog_ids is empty, applies to ALL catalogs. Use except_models to exclude specific models."
// @Success 200 {object} modelresponses.BulkOperationResponse "Bulk operation result with counts and status"
// @Failure 400 {object} responses.ErrorResponse "Invalid request - exceeds limits or validation error"
// @Failure 404 {object} responses.ErrorResponse "One or more catalog IDs not found (when catalog_ids provided)"
// @Failure 500 {object} responses.ErrorResponse "Internal server error during bulk operation"
// @Router /v1/admin/models/catalogs/bulk-toggle [post]
func (route *AdminModelRoute) BulkToggleModelCatalogs(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	var request requestmodels.BulkToggleCatalogsRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		responses.HandleError(reqCtx, err, "Invalid request body")
		return
	}

	if len(request.ExceptModels) > MaxExceptModelsLimit {
		err := platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "except_models list exceeds maximum limit", nil, "e0e2b831-9ce4-4fff-8fb8-8aef01979d5f")
		responses.HandleError(reqCtx, err, "Validation error: except_models list exceeds maximum limit")
		return
	}

	response, err := route.modelCatalogHandler.BulkToggleCatalogs(ctx, request)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to perform bulk toggle operation on model catalogs")
		return
	}

	reqCtx.JSON(http.StatusOK, response)
}

func (route *AdminModelRoute) buildModelCatalogFilter(reqCtx *gin.Context) requestmodels.ModelCatalogFilterParams {
	var filter requestmodels.ModelCatalogFilterParams

	if status := reqCtx.Query("status"); status != "" {
		filter.Status = &status
	}

	if isModeratedStr := reqCtx.Query("is_moderated"); isModeratedStr != "" {
		isModerated := isModeratedStr == "true"
		filter.IsModerated = &isModerated
	}

	if activeStr := reqCtx.Query("active"); activeStr != "" {
		active := activeStr == "true"
		filter.Active = &active
	}

	if supportsImagesStr := reqCtx.Query("supports_images"); supportsImagesStr != "" {
		value := supportsImagesStr == "true"
		filter.SupportsImages = &value
	}

	if supportsEmbeddingsStr := reqCtx.Query("supports_embeddings"); supportsEmbeddingsStr != "" {
		value := supportsEmbeddingsStr == "true"
		filter.SupportsEmbeddings = &value
	}

	if supportsReasoningStr := reqCtx.Query("supports_reasoning"); supportsReasoningStr != "" {
		value := supportsReasoningStr == "true"
		filter.SupportsReasoning = &value
	}

	if supportsAudioStr := reqCtx.Query("supports_audio"); supportsAudioStr != "" {
		value := supportsAudioStr == "true"
		filter.SupportsAudio = &value
	}

	if supportsVideoStr := reqCtx.Query("supports_video"); supportsVideoStr != "" {
		value := supportsVideoStr == "true"
		filter.SupportsVideo = &value
	}

	if supportsToolsStr := reqCtx.Query("supports_tools"); supportsToolsStr != "" {
		value := supportsToolsStr == "true"
		filter.SupportsTools = &value
	}

	if family := strings.TrimSpace(reqCtx.Query("family")); family != "" {
		filter.Family = &family
	}

	return filter
}

func (route *AdminModelRoute) buildProviderModelFilter(reqCtx *gin.Context) requestmodels.ProviderModelFilterParams {
	var filter requestmodels.ProviderModelFilterParams

	if providerID := reqCtx.Query("provider_id"); providerID != "" {
		filter.ProviderPublicID = &providerID
	}

	if modelKey := reqCtx.Query("model_key"); modelKey != "" {
		filter.ModelKey = &modelKey
	}

	if activeStr := reqCtx.Query("active"); activeStr != "" {
		active := activeStr == "true"
		filter.Active = &active
	}

	if supportsImagesStr := reqCtx.Query("supports_images"); supportsImagesStr != "" {
		supportsImages := supportsImagesStr == "true"
		filter.SupportsImages = &supportsImages
	}

	if search := strings.TrimSpace(reqCtx.Query("search")); search != "" {
		filter.SearchText = &search
	}

	return filter
}

// ListProviderModels
// @Summary List all provider models
// @Description Retrieves a paginated list of provider models with optional filtering
// @Tags Admin Model API
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Number of records to return (default: 20, max: 100)"
// @Param offset query int false "Number of records to skip for pagination"
// @Param order query string false "Sort order: asc or desc (default: desc)"
// @Param provider_id query string false "Filter by provider public ID"
// @Param model_key query string false "Filter by model key"
// @Param active query bool false "Filter by active status"
// @Param supports_images query bool false "Filter by image support"
// @Success 200 {object} modelresponses.ProviderModelResponse "List of provider models"
// @Failure 400 {object} responses.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/admin/models/provider-models [get]
func (route *AdminModelRoute) ListProviderModels(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	pagination, err := requests.GetPaginationFromQuery(reqCtx)
	if err != nil {
		responses.HandleError(reqCtx, err, "Invalid pagination parameters")
		return
	}

	filter := route.buildProviderModelFilter(reqCtx)

	providerModels, total, err := route.providerModelHandler.ListProviderModels(ctx, filter, pagination)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to retrieve provider models")
		return
	}

	reqCtx.JSON(http.StatusOK, gin.H{
		"data":  providerModels,
		"total": total,
		"limit": pagination.Limit,
	})
}

// GetProviderModel
// @Summary Get a provider model
// @Description Retrieves detailed information about a provider model by its public ID
// @Tags Admin Model API
// @Security BearerAuth
// @Produce json
// @Param provider_model_public_id path string true "Provider Model Public ID"
// @Success 200 {object} modelresponses.ProviderModelResponse "Provider model details"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 404 {object} responses.ErrorResponse "Provider model not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/admin/models/provider-models/{provider_model_public_id} [get]
func (route *AdminModelRoute) GetProviderModel(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	publicID := reqCtx.Param("provider_model_public_id")

	providerModel, err := route.providerModelHandler.GetProviderModel(ctx, publicID)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to retrieve provider model")
		return
	}

	reqCtx.JSON(http.StatusOK, providerModel)
}

// UpdateProviderModel
// @Summary Update a provider model
// @Description Updates configuration for a provider model including pricing, limits, and feature flags
// @Tags Admin Model API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param provider_model_public_id path string true "Provider Model Public ID"
// @Param payload body requestmodels.UpdateProviderModelRequest true "Update payload"
// @Success 200 {object} modelresponses.ProviderModelResponse "Updated provider model"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload"
// @Failure 404 {object} responses.ErrorResponse "Provider model not found"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/admin/models/provider-models/{provider_model_public_id} [patch]
func (route *AdminModelRoute) UpdateProviderModel(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()
	publicID := reqCtx.Param("provider_model_public_id")

	var request requestmodels.UpdateProviderModelRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		responses.HandleError(reqCtx, err, "Invalid request body")
		return
	}

	providerModel, err := route.providerModelHandler.UpdateProviderModel(ctx, publicID, request)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to update provider model")
		return
	}

	reqCtx.JSON(http.StatusOK, providerModel)
}

// BulkToggleProviderModels
// @Summary Bulk enable or disable provider models
// @Description Enables or disables provider models with flexible patterns: enable all, disable all, enable all except, or disable all except. Optionally filter by provider.
// @Tags Admin Model API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param payload body requestmodels.BulkEnableModelsRequest true "Bulk toggle payload with enable flag, optional provider filter, and exception list"
// @Success 200 {object} modelresponses.BulkOperationResponse "Bulk operation result with counts and status"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload"
// @Failure 500 {object} responses.ErrorResponse "Internal server error"
// @Router /v1/admin/models/provider-models/bulk-toggle [post]
//
// Supported patterns:
//   - Enable all: {"enable": true, "except_models": []}
//   - Disable all: {"enable": false, "except_models": []}
//   - Enable all except: {"enable": true, "except_models": ["model-key-1", "model-key-2"]}
//   - Disable all except: {"enable": false, "except_models": ["model-key-1", "model-key-2"]}
//   - Enable all for provider except: {"enable": true, "provider_id": "prov_abc", "except_models": ["model-x"]}
//
// Example use cases:
//   - "Enable all models": {"enable": true, "except_models": []}
//   - "Disable all except production whitelist": {"enable": false, "except_models": ["gpt-4o", "claude-3-opus"]}
//   - "Enable all OpenAI models except experimental": {"enable": true, "provider_id": "prov_openai_123", "except_models": ["gpt-5-preview"]}
//   - "Disable all models from specific provider": {"enable": false, "provider_id": "prov_xyz_789", "except_models": []}
func (route *AdminModelRoute) BulkToggleProviderModels(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	var request requestmodels.BulkEnableModelsRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		responses.HandleError(reqCtx, err, "Invalid request body")
		return
	}

	if len(request.ExceptModels) > MaxExceptModelsLimit {
		err := platformerrors.NewError(ctx, platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "except_models list exceeds maximum limit", nil, "5e2bfc34-7433-4022-8996-928852526723")
		responses.HandleError(reqCtx, err, "Validation error: except_models list exceeds maximum limit")
		return
	}

	response, err := route.providerModelHandler.BulkEnableDisableProviderModels(ctx, request)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to perform bulk toggle operation on provider models")
		return
	}

	reqCtx.JSON(http.StatusOK, response)
}
