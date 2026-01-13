package image

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/services/llm-api/internal/interfaces/httpserver/handlers/imagehandler"
	imagerequest "jan-server/services/llm-api/internal/interfaces/httpserver/requests/image"
	"jan-server/services/llm-api/internal/interfaces/httpserver/responses"
	imageresponse "jan-server/services/llm-api/internal/interfaces/httpserver/responses/image"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

var _ = imageresponse.ImageGenerationResponse{}

// ImageRoute handles image generation routes.
type ImageRoute struct {
	imageHandler *imagehandler.ImageHandler
	authHandler  *authhandler.AuthHandler
}

// NewImageRoute creates a new ImageRoute instance.
func NewImageRoute(
	imageHandler *imagehandler.ImageHandler,
	authHandler *authhandler.AuthHandler,
) *ImageRoute {
	return &ImageRoute{
		imageHandler: imageHandler,
		authHandler:  authHandler,
	}
}

// RegisterRouter registers the image routes.
func (r *ImageRoute) RegisterRouter(router gin.IRouter) {
	images := router.Group("/images")
	{
		images.POST("/generations",
			r.authHandler.WithAppUserAuthChain(
				r.PostGeneration,
			)...,
		)

		images.POST("/edits",
			r.authHandler.WithAppUserAuthChain(
				r.PostEdit,
			)...,
		)
		images.POST("/variations",
			r.authHandler.WithAppUserAuthChain(
				r.PostVariation,
			)...,
		)
	}
}

// PostGeneration
// @Summary Create image generation
// @Description Generates images from a text prompt using the configured image provider.
// @Description This endpoint is compatible with the OpenAI Images API format.
// @Description
// @Description **Response Formats:**
// @Description - url: Returns direct URLs to download images (default, recommended)
// @Description - b64_json: Returns base64-encoded image data
// @Description
// @Description **Size Options:**
// @Description - 1024x1024 (default)
// @Description - 512x512
// @Description - 1792x1024 (landscape)
// @Description - 1024x1792 (portrait)
// @Description
// @Tags Images API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body imagerequest.ImageGenerationRequest true "Image generation request"
// @Success 200 {object} imageresponse.ImageGenerationResponse "Successful image generation response"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload or validation error"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 404 {object} responses.ErrorResponse "No active image provider configured"
// @Failure 500 {object} responses.ErrorResponse "Internal server error or image provider error"
// @Failure 501 {object} responses.ErrorResponse "Feature not implemented"
// @Router /v1/images/generations [post]
func (r *ImageRoute) PostGeneration(reqCtx *gin.Context) {
	// Get authenticated user ID
	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "image-auth-001")
		return
	}

	var request imagerequest.ImageGenerationRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "Invalid request body", "image-validation-000")
		return
	}

	// Validate prompt is not empty
	if request.Prompt == "" {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "prompt is required", "image-validation-001")
		return
	}

	// Delegate to image handler
	result, err := r.imageHandler.GenerateImage(reqCtx.Request.Context(), reqCtx, user.ID, request)
	if err != nil {
		// Check specific error types
		if platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
			responses.HandleNewError(reqCtx, platformerrors.ErrorTypeNotFound, err.Error(), "image-provider-not-found")
			return
		}
		if platformerrors.IsValidationError(err) {
			responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, err.Error(), "image-validation-error")
			return
		}
		// Generic error
		responses.HandleError(reqCtx, err, "Image generation failed")
		return
	}

	reqCtx.JSON(http.StatusOK, result.Response)
}

// PostEdit
// @Summary Create image edit
// @Description Creates an edited image given an original image and a prompt.
// @Tags Images API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body imagerequest.ImageEditRequest true "Image edit request"
// @Success 200 {object} imageresponse.ImageGenerationResponse "Successful image edit response"
// @Failure 400 {object} responses.ErrorResponse "Invalid request payload or validation error"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized - missing or invalid authentication"
// @Failure 404 {object} responses.ErrorResponse "No active image provider configured"
// @Failure 500 {object} responses.ErrorResponse "Internal server error or image provider error"
// @Router /v1/images/edits [post]
func (r *ImageRoute) PostEdit(reqCtx *gin.Context) {
	user, ok := authhandler.GetUserFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeUnauthorized, "authentication required", "image-auth-001")
		return
	}

	var request imagerequest.ImageEditRequest
	if err := reqCtx.ShouldBindJSON(&request); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "Invalid request body", "image-edit-validation-000")
		return
	}

	if request.Prompt == "" {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "prompt is required", "image-edit-validation-001")
		return
	}
	if request.Image == nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "image is required", "image-edit-validation-002")
		return
	}

	result, err := r.imageHandler.EditImage(reqCtx.Request.Context(), reqCtx, user.ID, request)
	if err != nil {
		if platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
			responses.HandleNewError(reqCtx, platformerrors.ErrorTypeNotFound, err.Error(), "image-provider-not-found")
			return
		}
		if platformerrors.IsValidationError(err) {
			responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, err.Error(), "image-edit-validation-error")
			return
		}
		responses.HandleError(reqCtx, err, "Image edit failed")
		return
	}

	reqCtx.JSON(http.StatusOK, result.Response)
}

// PostVariation
// @Summary Create image variation (Not Implemented)
// @Description Creates a variation of a given image.
// @Description NOTE: This endpoint is not yet implemented and will return 501 Not Implemented.
// @Tags Images API
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 501 {object} responses.ErrorResponse "Not implemented"
// @Router /v1/images/variations [post]
func (r *ImageRoute) PostVariation(reqCtx *gin.Context) {
	responses.HandleNewError(reqCtx, platformerrors.ErrorTypeNotImplemented, "image variations not implemented", "image-variation-not-impl")
}
