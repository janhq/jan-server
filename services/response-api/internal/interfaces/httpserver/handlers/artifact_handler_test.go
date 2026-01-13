package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/response-api/internal/domain/artifact"
	"jan-server/services/response-api/internal/interfaces/httpserver/handlers"
)

// MockArtifactService is a mock implementation of artifact.Service for testing.
// Implements the actual artifact.Service interface.
type MockArtifactService struct {
	CreateFunc                func(ctx context.Context, params artifact.CreateParams) (*artifact.Artifact, error)
	CreateVersionFunc         func(ctx context.Context, artifactID string, params artifact.UpdateParams) (*artifact.Artifact, error)
	GetByIDFunc               func(ctx context.Context, id string) (*artifact.Artifact, error)
	GetLatestByResponseIDFunc func(ctx context.Context, responseID string) (*artifact.Artifact, error)
	GetLatestByPlanIDFunc     func(ctx context.Context, planID string) (*artifact.Artifact, error)
	GetVersionsFunc           func(ctx context.Context, artifactID string) ([]*artifact.Artifact, error)
	UpdateMetadataFunc        func(ctx context.Context, id string, metadata json.RawMessage) error
	SetRetentionFunc          func(ctx context.Context, id string, policy artifact.RetentionPolicy, expiresAt *time.Time) error
	DeleteFunc                func(ctx context.Context, id string) error
	CleanupExpiredFunc        func(ctx context.Context) (int64, error)
	ListFunc                  func(ctx context.Context, filter *artifact.Filter) ([]*artifact.Artifact, int64, error)
}

func (m *MockArtifactService) Create(ctx context.Context, params artifact.CreateParams) (*artifact.Artifact, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockArtifactService) CreateVersion(ctx context.Context, artifactID string, params artifact.UpdateParams) (*artifact.Artifact, error) {
	if m.CreateVersionFunc != nil {
		return m.CreateVersionFunc(ctx, artifactID, params)
	}
	return nil, nil
}

func (m *MockArtifactService) GetByID(ctx context.Context, id string) (*artifact.Artifact, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockArtifactService) GetLatestByResponseID(ctx context.Context, responseID string) (*artifact.Artifact, error) {
	if m.GetLatestByResponseIDFunc != nil {
		return m.GetLatestByResponseIDFunc(ctx, responseID)
	}
	return nil, nil
}

func (m *MockArtifactService) GetLatestByPlanID(ctx context.Context, planID string) (*artifact.Artifact, error) {
	if m.GetLatestByPlanIDFunc != nil {
		return m.GetLatestByPlanIDFunc(ctx, planID)
	}
	return nil, nil
}

func (m *MockArtifactService) GetVersions(ctx context.Context, artifactID string) ([]*artifact.Artifact, error) {
	if m.GetVersionsFunc != nil {
		return m.GetVersionsFunc(ctx, artifactID)
	}
	return nil, nil
}

func (m *MockArtifactService) UpdateMetadata(ctx context.Context, id string, metadata json.RawMessage) error {
	if m.UpdateMetadataFunc != nil {
		return m.UpdateMetadataFunc(ctx, id, metadata)
	}
	return nil
}

func (m *MockArtifactService) SetRetention(ctx context.Context, id string, policy artifact.RetentionPolicy, expiresAt *time.Time) error {
	if m.SetRetentionFunc != nil {
		return m.SetRetentionFunc(ctx, id, policy, expiresAt)
	}
	return nil
}

func (m *MockArtifactService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockArtifactService) CleanupExpired(ctx context.Context) (int64, error) {
	if m.CleanupExpiredFunc != nil {
		return m.CleanupExpiredFunc(ctx)
	}
	return 0, nil
}

func (m *MockArtifactService) List(ctx context.Context, filter *artifact.Filter) ([]*artifact.Artifact, int64, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, filter)
	}
	return nil, 0, nil
}

func setupArtifactTestRouter(handler *handlers.ArtifactHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Artifact routes
	artifacts := r.Group("/v1/artifacts")
	{
		artifacts.GET("/:artifact_id", handler.Get)
		artifacts.GET("/:artifact_id/versions", handler.GetVersions)
		artifacts.GET("/:artifact_id/download", handler.Download)
		artifacts.DELETE("/:artifact_id", handler.Delete)
	}

	// Response-scoped artifact routes
	responses := r.Group("/v1/responses/:response_id/artifacts")
	{
		responses.GET("", handler.GetByResponse)
		responses.GET("/latest", handler.GetLatestByResponse)
	}

	return r
}

func TestArtifactHandler_Get(t *testing.T) {
	mockService := &MockArtifactService{
		GetByIDFunc: func(ctx context.Context, id string) (*artifact.Artifact, error) {
			return &artifact.Artifact{
				ID:          id,
				ResponseID:  "resp-123",
				ContentType: artifact.ContentTypeSlides,
				Title:       "Test Presentation",
				Version:     1,
				CreatedAt:   time.Now(),
			}, nil
		},
	}

	handler := handlers.NewArtifactHandler(mockService, zerolog.Nop())
	router := setupArtifactTestRouter(handler)

	req, _ := http.NewRequest("GET", "/v1/artifacts/art-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["id"] != "art-123" {
		t.Errorf("Expected artifact id 'art-123', got %v", response["id"])
	}

	if response["title"] != "Test Presentation" {
		t.Errorf("Expected title 'Test Presentation', got %v", response["title"])
	}
}

func TestArtifactHandler_GetByResponse(t *testing.T) {
	mockService := &MockArtifactService{
		ListFunc: func(ctx context.Context, filter *artifact.Filter) ([]*artifact.Artifact, int64, error) {
			return []*artifact.Artifact{
				{
					ID:          "art-1",
					ResponseID:  "resp-123",
					ContentType: artifact.ContentTypeSlides,
					Title:       "Presentation 1",
					Version:     1,
				},
				{
					ID:          "art-2",
					ResponseID:  "resp-123",
					ContentType: artifact.ContentTypeDocument,
					Title:       "Document 1",
					Version:     1,
				},
			}, 2, nil
		},
	}

	handler := handlers.NewArtifactHandler(mockService, zerolog.Nop())
	router := setupArtifactTestRouter(handler)

	req, _ := http.NewRequest("GET", "/v1/responses/resp-123/artifacts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	data, ok := response["data"].([]interface{})
	if !ok {
		t.Fatalf("Expected data array, got %T", response["data"])
	}

	if len(data) != 2 {
		t.Errorf("Expected 2 artifacts, got %d", len(data))
	}
}

func TestArtifactHandler_GetLatestByResponse(t *testing.T) {
	mockService := &MockArtifactService{
		GetLatestByResponseIDFunc: func(ctx context.Context, responseID string) (*artifact.Artifact, error) {
			return &artifact.Artifact{
				ID:          "art-latest",
				ResponseID:  responseID,
				ContentType: artifact.ContentTypeSlides,
				Title:       "Latest Artifact",
				Version:     3,
			}, nil
		},
	}

	handler := handlers.NewArtifactHandler(mockService, zerolog.Nop())
	router := setupArtifactTestRouter(handler)

	req, _ := http.NewRequest("GET", "/v1/responses/resp-123/artifacts/latest", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["id"] != "art-latest" {
		t.Errorf("Expected artifact id 'art-latest', got %v", response["id"])
	}

	if response["version"] != float64(3) {
		t.Errorf("Expected version 3, got %v", response["version"])
	}
}

func TestArtifactHandler_GetVersions(t *testing.T) {
	mockService := &MockArtifactService{
		GetVersionsFunc: func(ctx context.Context, artifactID string) ([]*artifact.Artifact, error) {
			return []*artifact.Artifact{
				{
					ID:      "art-v1",
					Version: 1,
				},
				{
					ID:      "art-v2",
					Version: 2,
				},
				{
					ID:      "art-v3",
					Version: 3,
				},
			}, nil
		},
	}

	handler := handlers.NewArtifactHandler(mockService, zerolog.Nop())
	router := setupArtifactTestRouter(handler)

	req, _ := http.NewRequest("GET", "/v1/artifacts/art-123/versions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var versions []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &versions); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(versions))
	}
}

func TestArtifactHandler_Delete(t *testing.T) {
	deleteCalled := false
	mockService := &MockArtifactService{
		DeleteFunc: func(ctx context.Context, id string) error {
			deleteCalled = true
			if id != "art-123" {
				t.Errorf("Expected artifact id 'art-123', got %v", id)
			}
			return nil
		},
	}

	handler := handlers.NewArtifactHandler(mockService, zerolog.Nop())
	router := setupArtifactTestRouter(handler)

	req, _ := http.NewRequest("DELETE", "/v1/artifacts/art-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	if !deleteCalled {
		t.Error("Expected Delete to be called")
	}
}

func TestArtifactHandler_Download(t *testing.T) {
	content := `{"slides": [{"title": "Slide 1"}]}`
	mockService := &MockArtifactService{
		GetByIDFunc: func(ctx context.Context, id string) (*artifact.Artifact, error) {
			return &artifact.Artifact{
				ID:          id,
				ContentType: artifact.ContentTypeSlides,
				Title:       "Test Presentation",
				Content:     &content,
				MimeType:    "application/json",
			}, nil
		},
	}

	handler := handlers.NewArtifactHandler(mockService, zerolog.Nop())
	router := setupArtifactTestRouter(handler)

	req, _ := http.NewRequest("GET", "/v1/artifacts/art-123/download", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %v", contentType)
	}

	contentDisposition := w.Header().Get("Content-Disposition")
	if contentDisposition == "" {
		t.Error("Expected Content-Disposition header to be set")
	}
}
