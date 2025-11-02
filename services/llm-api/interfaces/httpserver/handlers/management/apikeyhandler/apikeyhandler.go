package apikeyhandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"menlo.ai/menlo-platform/internal/domain/apikey"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/handlers/authhandler"
	"menlo.ai/menlo-platform/internal/interfaces/httpserver/responses"
)

type APIKeyHandler struct {
	apikeyService *apikey.APIKeyService
}

type APIKeyContextKey string

const (
	APIKeyContextKeyPublicID APIKeyContextKey = "apikey_public_id"
	APIKeyContextKeyEntity   APIKeyContextKey = "APIKeyContextKeyEntity"
)

func NewAPIKeyHandler(apikeyService *apikey.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{
		apikeyService: apikeyService,
	}
}

func (h *APIKeyHandler) GetPublicIDQueryPath() string {
	return string(APIKeyContextKeyPublicID)
}

func GetAPIKeyFromContext(reqCtx *gin.Context) (*apikey.APIKey, bool) {
	i, ok := reqCtx.Get(string(APIKeyContextKeyEntity))
	if !ok {
		return nil, false
	}
	v, ok := i.(*apikey.APIKey)
	if !ok {
		return nil, false
	}
	return v, true
}

func SetAPIKeyToContext(reqCtx *gin.Context, i *apikey.APIKey) {
	reqCtx.Set(string(APIKeyContextKeyEntity), i)
}

func (h *APIKeyHandler) GetAPIKeyFromQueryPathMiddleware() gin.HandlerFunc {
	return func(reqCtx *gin.Context) {
		ctx := reqCtx.Request.Context()
		user, ok := authhandler.GetUserFromContext(reqCtx)
		if !ok {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "3296ce86-783b-4c05-9fdb-930d3713024e",
			})
			return
		}
		publicID := reqCtx.Param(string(APIKeyContextKeyPublicID))
		entity, err := h.apikeyService.FindByPublicID(ctx, publicID)
		if err != nil || entity == nil {
			reqCtx.AbortWithStatusJSON(http.StatusNotFound, responses.ErrorResponse{
				Code: "b652e656-7301-44a5-b55f-bd1336ce57b7",
			})
			return
		}
		if entity.UserID != user.ID {
			reqCtx.AbortWithStatusJSON(http.StatusUnauthorized, responses.ErrorResponse{
				Code: "b1ef40e7-9db9-477d-bb59-f3783585195d",
			})
			return
		}
		SetAPIKeyToContext(reqCtx, entity)
		reqCtx.Next()
	}
}
