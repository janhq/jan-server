package requests

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"menlo.ai/menlo-platform/internal/domain/query"
	"menlo.ai/menlo-platform/internal/utils/platformerrors"
)

func GetCursorPaginationFromQuery(reqCtx *gin.Context, findByLastID func(string) (*uint, error)) (*query.Pagination, error) {
	limitStr := reqCtx.DefaultQuery("limit", "20")
	offsetStr := reqCtx.Query("offset")
	order := reqCtx.DefaultQuery("order", "desc")
	afterStr := reqCtx.DefaultQuery("after", "")

	var limit *int
	if limitStr != "" {
		limitInt, err := strconv.Atoi(limitStr)
		if err != nil || limitInt < 1 {
			return nil, platformerrors.NewError(reqCtx.Request.Context(), platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "invalid limit number", nil, "04aecd25-bd32-428b-864d-aeb7ecb06e53")
		}
		limit = &limitInt
	}

	var offset *int
	var after *uint
	if offsetStr != "" {
		offsetInt, err := strconv.Atoi(offsetStr)
		if err != nil {
			return nil, platformerrors.NewError(reqCtx.Request.Context(), platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "invalid offset number", nil, "a3e0ea22-afc6-45df-b686-a194868af415")
		}
		offset = &offsetInt
	} else if afterStr != "" {
		lastID, err := findByLastID(afterStr)
		if err != nil {
			return nil, platformerrors.NewError(reqCtx.Request.Context(), platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "invalid offset number", nil, "1f9ee4ee-56ed-448e-9296-d978c9a03726")
		}
		after = lastID
	}

	if order != "asc" && order != "desc" {
		return nil, platformerrors.NewError(reqCtx.Request.Context(), platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "invalid order", nil, "c3598493-7770-4e94-b44f-f571aabf2bdd")
	}

	return &query.Pagination{
		Limit:  limit,
		Offset: offset,
		Order:  order,
		After:  after,
	}, nil
}

func GetPaginationFromQuery(reqCtx *gin.Context) (*query.Pagination, error) {
	return GetCursorPaginationFromQuery(reqCtx, func(s string) (*uint, error) {
		return nil, platformerrors.NewError(reqCtx.Request.Context(), platformerrors.LayerHandler, platformerrors.ErrorTypeValidation, "invalid query parameter: last", nil, "6b72a4af-ea95-4fbc-b141-486f4da86e79")
	})
}
