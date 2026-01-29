package routes

import (
	"errors"
	"net/http"
	"strconv"

	"jan-server/mono/apps/backend/internal/domain/artifact"
	"jan-server/mono/apps/backend/internal/infrastructure/config"
	"jan-server/mono/apps/backend/internal/infrastructure/database/repository"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func getArtifactService(db *gorm.DB) *artifact.Service {
	repo := repository.NewArtifactRepository(db)
	return artifact.NewService(repo)
}

// ============================================
// Request types
// ============================================

type createArtifactRequest struct {
	ConversationID *string        `json:"conversation_id"`
	ResponseID     *string        `json:"response_id"`
	Title          string         `json:"title" binding:"required"`
	Description    string         `json:"description"`
	Type           string         `json:"type" binding:"required"`
	Language       string         `json:"language"`
	Content        string         `json:"content" binding:"required"`
	Metadata       map[string]any `json:"metadata"`
	IsPublic       bool           `json:"is_public"`
}

type updateArtifactRequest struct {
	Title       *string        `json:"title"`
	Description *string        `json:"description"`
	Content     *string        `json:"content"`
	Language    *string        `json:"language"`
	IsPublic    *bool          `json:"is_public"`
	Metadata    map[string]any `json:"metadata"`
}

// ============================================
// Handlers
// ============================================

func listArtifactsHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		search := c.Query("search")
		typeFilter := c.Query("type")

		var conversationID *string
		if v := c.Query("conversation_id"); v != "" {
			conversationID = &v
		}

		svc := getArtifactService(db)
		artifacts, total, err := svc.List(c.Request.Context(), artifact.ListArtifactsFilter{
			UserID:         principal.ID,
			ConversationID: conversationID,
			Type:           typeFilter,
			Search:         search,
			Limit:          limit,
			Offset:         offset,
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		responses := make([]artifact.ArtifactResponse, len(artifacts))
		for i, a := range artifacts {
			responses[i] = a.ToResponse()
		}

		c.JSON(http.StatusOK, gin.H{
			"artifacts": responses,
			"total":     total,
			"limit":     limit,
			"offset":    offset,
		})
	}
}

func createArtifactHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req createArtifactRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		svc := getArtifactService(db)
		a, err := svc.Create(c.Request.Context(), principal.ID, artifact.CreateArtifactRequest{
			ConversationID: req.ConversationID,
			ResponseID:     req.ResponseID,
			Title:          req.Title,
			Description:    req.Description,
			Type:           req.Type,
			Language:       req.Language,
			Content:        req.Content,
			Metadata:       req.Metadata,
			IsPublic:       req.IsPublic,
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, a.ToResponse())
	}
}

func getArtifactHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		svc := getArtifactService(db)
		a, err := svc.GetByID(c.Request.Context(), principal.ID, id)

		if err != nil {
			if errors.Is(err, artifact.ErrArtifactNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "artifact not found"})
				return
			}
			if errors.Is(err, artifact.ErrUnauthorized) {
				c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, a.ToResponse())
	}
}

func updateArtifactHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		var req updateArtifactRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		svc := getArtifactService(db)
		a, err := svc.Update(c.Request.Context(), principal.ID, id, artifact.UpdateArtifactRequest{
			Title:       req.Title,
			Description: req.Description,
			Content:     req.Content,
			Language:    req.Language,
			IsPublic:    req.IsPublic,
			Metadata:    req.Metadata,
		})

		if err != nil {
			if errors.Is(err, artifact.ErrArtifactNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "artifact not found"})
				return
			}
			if errors.Is(err, artifact.ErrUnauthorized) {
				c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, a.ToResponse())
	}
}

func deleteArtifactHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		svc := getArtifactService(db)
		err := svc.Delete(c.Request.Context(), principal.ID, id)

		if err != nil {
			if errors.Is(err, artifact.ErrArtifactNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "artifact not found"})
				return
			}
			if errors.Is(err, artifact.ErrUnauthorized) {
				c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "artifact deleted"})
	}
}

func listArtifactVersionsHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		svc := getArtifactService(db)
		versions, err := svc.ListVersions(c.Request.Context(), principal.ID, id)

		if err != nil {
			if errors.Is(err, artifact.ErrArtifactNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "artifact not found"})
				return
			}
			if errors.Is(err, artifact.ErrUnauthorized) {
				c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		responses := make([]artifact.ArtifactVersionResponse, len(versions))
		for i, v := range versions {
			responses[i] = v.ToResponse()
		}

		c.JSON(http.StatusOK, gin.H{"versions": responses})
	}
}

func downloadArtifactHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		svc := getArtifactService(db)
		a, err := svc.GetByID(c.Request.Context(), principal.ID, id)

		if err != nil {
			if errors.Is(err, artifact.ErrArtifactNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "artifact not found"})
				return
			}
			if errors.Is(err, artifact.ErrUnauthorized) {
				c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Set content type based on artifact type
		contentType := "text/plain"
		filename := a.Title
		switch a.Type {
		case "html":
			contentType = "text/html"
			filename += ".html"
		case "svg":
			contentType = "image/svg+xml"
			filename += ".svg"
		case "markdown":
			contentType = "text/markdown"
			filename += ".md"
		case "code":
			if a.Language != "" {
				filename += getExtensionForLanguage(a.Language)
			} else {
				filename += ".txt"
			}
		case "mermaid":
			contentType = "text/plain"
			filename += ".mmd"
		}

		c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
		c.Data(http.StatusOK, contentType, []byte(a.Content))
	}
}

func getExtensionForLanguage(lang string) string {
	extensions := map[string]string{
		"javascript": ".js",
		"typescript": ".ts",
		"python":     ".py",
		"go":         ".go",
		"rust":       ".rs",
		"java":       ".java",
		"c":          ".c",
		"cpp":        ".cpp",
		"csharp":     ".cs",
		"ruby":       ".rb",
		"php":        ".php",
		"swift":      ".swift",
		"kotlin":     ".kt",
		"scala":      ".scala",
		"html":       ".html",
		"css":        ".css",
		"json":       ".json",
		"yaml":       ".yaml",
		"xml":        ".xml",
		"sql":        ".sql",
		"shell":      ".sh",
		"bash":       ".sh",
		"powershell": ".ps1",
	}
	if ext, ok := extensions[lang]; ok {
		return ext
	}
	return ".txt"
}
