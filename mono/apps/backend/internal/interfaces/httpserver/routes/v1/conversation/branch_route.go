package conversation

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/authhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/handlers/conversationhandler"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/responses"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

type BranchRoute struct {
	handler             *conversationhandler.ConversationHandler
	branchHandler       *conversationhandler.BranchHandler
	authHandler         *authhandler.AuthHandler
}

func NewBranchRoute(
	handler *conversationhandler.ConversationHandler,
	branchHandler *conversationhandler.BranchHandler,
	authHandler *authhandler.AuthHandler,
) *BranchRoute {
	return &BranchRoute{
		handler:       handler,
		branchHandler: branchHandler,
		authHandler:   authHandler,
	}
}

func (route *BranchRoute) RegisterRouter(router gin.IRouter) {
	conversations := router.Group("/conversations")
	
	// Branch CRUD endpoints
	conversations.GET("/:conv_public_id/branches", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.listBranches)...)
	conversations.POST("/:conv_public_id/branches", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.createBranch)...)
	conversations.GET("/:conv_public_id/branches/:branch_name", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.getBranch)...)
	conversations.DELETE("/:conv_public_id/branches/:branch_name", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.deleteBranch)...)
	conversations.POST("/:conv_public_id/branches/:branch_name/activate", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.activateBranch)...)
	
	// Message action endpoints
	conversations.POST("/:conv_public_id/items/:item_id/edit", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.editMessage)...)
	conversations.POST("/:conv_public_id/items/:item_id/regenerate", route.authHandler.WithAppUserAuthChain(route.handler.ConversationMiddleware(), route.regenerateMessage)...)
}

// listBranches godoc
// @Summary List branches
// @Description List all branches for a conversation
// @Tags Conversation Branches
// @Security BearerAuth
// @Produce json
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Success 200 {object} conversationhandler.ListBranchesResponse "Successfully retrieved branches"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Router /v1/conversations/{conv_public_id}/branches [get]
func (route *BranchRoute) listBranches(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d")
		return
	}

	response, err := route.branchHandler.ListBranches(ctx, conv)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to list branches")
		return
	}

	reqCtx.JSON(http.StatusOK, response)
}

// createBranch godoc
// @Summary Create a branch
// @Description Create a new branch in a conversation, optionally forking from an existing item
// @Tags Conversation Branches
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Param request body conversationhandler.CreateBranchRequest true "Create branch request"
// @Success 201 {object} conversationhandler.BranchResponse "Successfully created branch"
// @Failure 400 {object} responses.ErrorResponse "Invalid request"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 404 {object} responses.ErrorResponse "Conversation not found"
// @Router /v1/conversations/{conv_public_id}/branches [post]
func (route *BranchRoute) createBranch(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "b2c3d4e5-f6a7-4b8c-9d0e-1f2a3b4c5d6e")
		return
	}

	var req conversationhandler.CreateBranchRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "invalid request body", "c3d4e5f6-a7b8-4c9d-0e1f-2a3b4c5d6e7f")
		return
	}

	response, err := route.branchHandler.CreateBranch(ctx, conv, req)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to create branch")
		return
	}

	reqCtx.JSON(http.StatusCreated, response)
}

// getBranch godoc
// @Summary Get branch details
// @Description Get details of a specific branch
// @Tags Conversation Branches
// @Security BearerAuth
// @Produce json
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Param branch_name path string true "Branch name"
// @Success 200 {object} conversationhandler.BranchResponse "Successfully retrieved branch"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 404 {object} responses.ErrorResponse "Branch not found"
// @Router /v1/conversations/{conv_public_id}/branches/{branch_name} [get]
func (route *BranchRoute) getBranch(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "d4e5f6a7-b8c9-4d0e-1f2a-3b4c5d6e7f8a")
		return
	}

	branchName := reqCtx.Param("branch_name")
	if branchName == "" {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "branch name is required", "e5f6a7b8-c9d0-4e1f-2a3b-4c5d6e7f8a9b")
		return
	}

	response, err := route.branchHandler.GetBranch(ctx, conv, branchName)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to get branch")
		return
	}

	reqCtx.JSON(http.StatusOK, response)
}

// deleteBranch godoc
// @Summary Delete a branch
// @Description Delete a branch from a conversation (cannot delete MAIN or active branch)
// @Tags Conversation Branches
// @Security BearerAuth
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Param branch_name path string true "Branch name"
// @Success 204 "Branch deleted successfully"
// @Failure 400 {object} responses.ErrorResponse "Cannot delete MAIN or active branch"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 404 {object} responses.ErrorResponse "Branch not found"
// @Router /v1/conversations/{conv_public_id}/branches/{branch_name} [delete]
func (route *BranchRoute) deleteBranch(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "f6a7b8c9-d0e1-4f2a-3b4c-5d6e7f8a9b0c")
		return
	}

	branchName := reqCtx.Param("branch_name")
	if branchName == "" {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "branch name is required", "a7b8c9d0-e1f2-4a3b-4c5d-6e7f8a9b0c1d")
		return
	}

	if err := route.branchHandler.DeleteBranch(ctx, conv, branchName); err != nil {
		responses.HandleError(reqCtx, err, "Failed to delete branch")
		return
	}

	reqCtx.Status(http.StatusNoContent)
}

// activateBranch godoc
// @Summary Activate a branch
// @Description Set a branch as the active branch for a conversation
// @Tags Conversation Branches
// @Security BearerAuth
// @Produce json
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Param branch_name path string true "Branch name"
// @Success 200 {object} conversationhandler.ActivateBranchResponse "Branch activated successfully"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 404 {object} responses.ErrorResponse "Branch not found"
// @Router /v1/conversations/{conv_public_id}/branches/{branch_name}/activate [post]
func (route *BranchRoute) activateBranch(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "b8c9d0e1-f2a3-4b4c-5d6e-7f8a9b0c1d2e")
		return
	}

	branchName := reqCtx.Param("branch_name")
	if branchName == "" {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "branch name is required", "c9d0e1f2-a3b4-4c5d-6e7f-8a9b0c1d2e3f")
		return
	}

	response, err := route.branchHandler.ActivateBranch(ctx, conv, branchName)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to activate branch")
		return
	}

	reqCtx.JSON(http.StatusOK, response)
}

// editMessage godoc
// @Summary Edit a message
// @Description Edit a user message and create a new branch with the edited content
// @Tags Message Actions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Param item_id path string true "Message item ID (format: msg_xxxxx)"
// @Param request body conversationhandler.EditMessageRequest true "Edit message request"
// @Success 200 {object} conversationhandler.EditMessageResponse "Message edited successfully"
// @Failure 400 {object} responses.ErrorResponse "Invalid request or not a user message"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 404 {object} responses.ErrorResponse "Message not found"
// @Router /v1/conversations/{conv_public_id}/items/{item_id}/edit [post]
func (route *BranchRoute) editMessage(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "d0e1f2a3-b4c5-4d6e-7f8a-9b0c1d2e3f4a")
		return
	}

	itemID := reqCtx.Param("item_id")
	if itemID == "" {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "item ID is required", "e1f2a3b4-c5d6-4e7f-8a9b-0c1d2e3f4a5b")
		return
	}

	var req conversationhandler.EditMessageRequest
	if err := reqCtx.ShouldBindJSON(&req); err != nil {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "invalid request body", "f2a3b4c5-d6e7-4f8a-9b0c-1d2e3f4a5b6c")
		return
	}

	response, err := route.branchHandler.EditMessage(ctx, conv, itemID, req)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to edit message")
		return
	}

	reqCtx.JSON(http.StatusOK, response)
}

// regenerateMessage godoc
// @Summary Regenerate a response
// @Description Regenerate an assistant response by creating a new branch
// @Tags Message Actions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param conv_public_id path string true "Conversation ID (format: conv_xxxxx)"
// @Param item_id path string true "Assistant message item ID (format: msg_xxxxx)"
// @Param request body conversationhandler.RegenerateMessageRequest false "Regenerate options"
// @Success 200 {object} conversationhandler.RegenerateMessageResponse "Regeneration initiated"
// @Failure 400 {object} responses.ErrorResponse "Invalid request or not an assistant message"
// @Failure 401 {object} responses.ErrorResponse "Unauthorized"
// @Failure 404 {object} responses.ErrorResponse "Message not found"
// @Router /v1/conversations/{conv_public_id}/items/{item_id}/regenerate [post]
func (route *BranchRoute) regenerateMessage(reqCtx *gin.Context) {
	ctx := reqCtx.Request.Context()

	conv, ok := conversationhandler.GetConversationFromContext(reqCtx)
	if !ok {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeInternal, "conversation not found in context", "a3b4c5d6-e7f8-4a9b-0c1d-2e3f4a5b6c7d")
		return
	}

	itemID := reqCtx.Param("item_id")
	if itemID == "" {
		responses.HandleNewError(reqCtx, platformerrors.ErrorTypeValidation, "item ID is required", "b4c5d6e7-f8a9-4b0c-1d2e-3f4a5b6c7d8e")
		return
	}

	var req conversationhandler.RegenerateMessageRequest
	// Body is optional for regenerate
	_ = reqCtx.ShouldBindJSON(&req)

	response, err := route.branchHandler.RegenerateMessage(ctx, conv, itemID, req)
	if err != nil {
		responses.HandleError(reqCtx, err, "Failed to regenerate message")
		return
	}

	reqCtx.JSON(http.StatusOK, response)
}
