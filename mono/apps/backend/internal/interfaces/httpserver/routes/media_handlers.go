package routes

import (
	"errors"
	"net/http"

	"jan-server/mono/apps/backend/internal/domain/media"
	"jan-server/mono/apps/backend/internal/infrastructure/config"
	"jan-server/mono/apps/backend/internal/infrastructure/database/repository"
	"jan-server/mono/apps/backend/internal/interfaces/httpserver/middlewares"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Note: In a full implementation, you would also inject the S3 storage client
func getMediaService(cfg *config.Config, db *gorm.DB) *media.Service {
	repo := repository.NewMediaRepository(db)
	// TODO: Initialize actual S3 storage client
	// For now, return nil storage client - handlers will return not implemented
	return media.NewService(repo, nil, media.ServiceConfig{
		Bucket:        cfg.S3Bucket,
		MaxUploadSize: cfg.MediaMaxUploadSize,
		PresignTTL:    cfg.MediaPresignTTL,
	})
}

// ============================================
// Request types
// ============================================

type presignedUploadRequest struct {
	Filename string `json:"filename" binding:"required"`
	MimeType string `json:"mime_type" binding:"required"`
	Size     int64  `json:"size" binding:"required"`
	Purpose  string `json:"purpose"`
}

// ============================================
// Handlers
// ============================================

func uploadMediaHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// Get file from form
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
			return
		}
		defer file.Close()

		purpose := c.PostForm("purpose")
		if purpose == "" {
			purpose = "attachment"
		}

		// Check file size
		if header.Size > cfg.MediaMaxUploadSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "file_too_large",
				"message": "File exceeds maximum allowed size",
			})
			return
		}

		svc := getMediaService(cfg, db)
		if svc == nil {
			c.JSON(http.StatusNotImplemented, gin.H{
				"error":   "not_implemented",
				"message": "Media upload is not yet configured",
			})
			return
		}

		m, err := svc.Upload(c.Request.Context(), principal.ID, media.UploadRequest{
			Filename: header.Filename,
			MimeType: header.Header.Get("Content-Type"),
			Size:     header.Size,
			Purpose:  purpose,
		}, file)

		if err != nil {
			if errors.Is(err, media.ErrFileTooLarge) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "file too large"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, m.ToResponse(""))
	}
}

func getPresignedUploadURLHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req presignedUploadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Purpose == "" {
			req.Purpose = "attachment"
		}

		svc := getMediaService(cfg, db)
		if svc == nil {
			c.JSON(http.StatusNotImplemented, gin.H{
				"error":   "not_implemented",
				"message": "Media upload is not yet configured",
			})
			return
		}

		resp, err := svc.GetPresignedUploadURL(c.Request.Context(), principal.ID, media.PresignedUploadRequest{
			Filename: req.Filename,
			MimeType: req.MimeType,
			Size:     req.Size,
			Purpose:  req.Purpose,
		})

		if err != nil {
			if errors.Is(err, media.ErrFileTooLarge) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "file too large"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func getMediaHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		svc := getMediaService(cfg, db)
		if svc == nil {
			c.JSON(http.StatusNotImplemented, gin.H{
				"error":   "not_implemented",
				"message": "Media service is not yet configured",
			})
			return
		}

		m, url, err := svc.GetByID(c.Request.Context(), principal.ID, id)

		if err != nil {
			if errors.Is(err, media.ErrMediaNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "media not found"})
				return
			}
			if errors.Is(err, media.ErrUnauthorized) {
				c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Redirect to presigned URL or return metadata with URL
		if c.Query("redirect") == "true" {
			c.Redirect(http.StatusTemporaryRedirect, url)
			return
		}

		c.JSON(http.StatusOK, m.ToResponse(url))
	}
}

func getMediaMetadataHandler(cfg *config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := middlewares.GetPrincipal(c)
		if principal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := c.Param("id")
		svc := getMediaService(cfg, db)
		if svc == nil {
			c.JSON(http.StatusNotImplemented, gin.H{
				"error":   "not_implemented",
				"message": "Media service is not yet configured",
			})
			return
		}

		m, err := svc.GetMetadata(c.Request.Context(), principal.ID, id)

		if err != nil {
			if errors.Is(err, media.ErrMediaNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "media not found"})
				return
			}
			if errors.Is(err, media.ErrUnauthorized) {
				c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, m.ToMetadataResponse())
	}
}
