package artifact

import (
	"context"
	"testing"
)

// MockArtifactRepository is a mock implementation for testing
type MockArtifactRepository struct {
	artifacts map[string]*Artifact
	versions  map[string][]*ArtifactVersion
}

func NewMockArtifactRepository() *MockArtifactRepository {
	return &MockArtifactRepository{
		artifacts: make(map[string]*Artifact),
		versions:  make(map[string][]*ArtifactVersion),
	}
}

func (m *MockArtifactRepository) Create(ctx context.Context, a *Artifact) error {
	m.artifacts[a.ID] = a
	return nil
}

func (m *MockArtifactRepository) FindByID(ctx context.Context, id string) (*Artifact, error) {
	a, ok := m.artifacts[id]
	if !ok {
		return nil, ErrArtifactNotFound
	}
	return a, nil
}

func (m *MockArtifactRepository) List(ctx context.Context, filter ListArtifactsFilter) ([]*Artifact, int64, error) {
	var result []*Artifact
	for _, a := range m.artifacts {
		if a.UserID != filter.UserID {
			continue
		}
		if filter.Type != "" && a.Type != filter.Type {
			continue
		}
		if filter.ConversationID != nil && (a.ConversationID == nil || *a.ConversationID != *filter.ConversationID) {
			continue
		}
		result = append(result, a)
	}
	return result, int64(len(result)), nil
}

func (m *MockArtifactRepository) Update(ctx context.Context, a *Artifact) error {
	if _, ok := m.artifacts[a.ID]; !ok {
		return ErrArtifactNotFound
	}
	m.artifacts[a.ID] = a
	return nil
}

func (m *MockArtifactRepository) Delete(ctx context.Context, id string) error {
	delete(m.artifacts, id)
	delete(m.versions, id)
	return nil
}

func (m *MockArtifactRepository) CreateVersion(ctx context.Context, v *ArtifactVersion) error {
	m.versions[v.ArtifactID] = append(m.versions[v.ArtifactID], v)
	return nil
}

func (m *MockArtifactRepository) GetVersions(ctx context.Context, artifactID string) ([]*ArtifactVersion, error) {
	versions := m.versions[artifactID]
	return versions, nil
}

func TestService_Create(t *testing.T) {
	repo := NewMockArtifactRepository()
	svc := NewService(repo)

	tests := []struct {
		name    string
		userID  string
		req     CreateArtifactRequest
		wantErr bool
	}{
		{
			name:   "create code artifact",
			userID: "user-123",
			req: CreateArtifactRequest{
				Title:    "Hello World",
				Type:     "code",
				Language: "javascript",
				Content:  "console.log('Hello, World!');",
			},
			wantErr: false,
		},
		{
			name:   "create HTML artifact",
			userID: "user-123",
			req: CreateArtifactRequest{
				Title:   "My Page",
				Type:    "html",
				Content: "<html><body>Hello</body></html>",
			},
			wantErr: false,
		},
		{
			name:   "missing title",
			userID: "user-123",
			req: CreateArtifactRequest{
				Type:    "code",
				Content: "some code",
			},
			wantErr: true,
		},
		{
			name:   "missing content",
			userID: "user-123",
			req: CreateArtifactRequest{
				Title: "Empty",
				Type:  "code",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact, err := svc.Create(context.Background(), tt.userID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if artifact.Title != tt.req.Title {
					t.Errorf("Expected title %s, got %s", tt.req.Title, artifact.Title)
				}
				if artifact.UserID != tt.userID {
					t.Errorf("Expected userID %s, got %s", tt.userID, artifact.UserID)
				}
			}
		})
	}
}

func TestService_GetByID(t *testing.T) {
	repo := NewMockArtifactRepository()
	svc := NewService(repo)

	// Create an artifact first
	artifact, err := svc.Create(context.Background(), "user-123", CreateArtifactRequest{
		Title:   "Test Artifact",
		Type:    "code",
		Content: "test content",
	})
	if err != nil {
		t.Fatalf("Failed to create artifact: %v", err)
	}

	tests := []struct {
		name       string
		userID     string
		artifactID string
		wantErr    bool
	}{
		{
			name:       "get existing artifact",
			userID:     "user-123",
			artifactID: artifact.ID,
			wantErr:    false,
		},
		{
			name:       "get non-existent artifact",
			userID:     "user-123",
			artifactID: "non-existent",
			wantErr:    true,
		},
		{
			name:       "get artifact from different user",
			userID:     "user-456",
			artifactID: artifact.ID,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.GetByID(context.Background(), tt.userID, tt.artifactID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result.ID != tt.artifactID {
				t.Errorf("Expected artifact ID %s, got %s", tt.artifactID, result.ID)
			}
		})
	}
}

func TestService_Update(t *testing.T) {
	repo := NewMockArtifactRepository()
	svc := NewService(repo)

	// Create an artifact first
	artifact, err := svc.Create(context.Background(), "user-123", CreateArtifactRequest{
		Title:   "Original Title",
		Type:    "code",
		Content: "original content",
	})
	if err != nil {
		t.Fatalf("Failed to create artifact: %v", err)
	}

	newTitle := "Updated Title"
	newContent := "updated content"

	updated, err := svc.Update(context.Background(), "user-123", artifact.ID, UpdateArtifactRequest{
		Title:   &newTitle,
		Content: &newContent,
	})
	if err != nil {
		t.Fatalf("Service.Update() error = %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("Expected title %s, got %s", newTitle, updated.Title)
	}

	if updated.Content != newContent {
		t.Errorf("Expected content %s, got %s", newContent, updated.Content)
	}

	// Version should be incremented
	if updated.Version != 2 {
		t.Errorf("Expected version 2, got %d", updated.Version)
	}
}

func TestService_List(t *testing.T) {
	repo := NewMockArtifactRepository()
	svc := NewService(repo)

	// Create some artifacts
	for i := 0; i < 5; i++ {
		svc.Create(context.Background(), "user-123", CreateArtifactRequest{
			Title:   "Artifact",
			Type:    "code",
			Content: "content",
		})
	}

	// Create artifacts for another user
	for i := 0; i < 3; i++ {
		svc.Create(context.Background(), "user-456", CreateArtifactRequest{
			Title:   "Other Artifact",
			Type:    "html",
			Content: "content",
		})
	}

	// List user-123's artifacts
	artifacts, total, err := svc.List(context.Background(), ListArtifactsFilter{
		UserID: "user-123",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("Service.List() error = %v", err)
	}

	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}

	if len(artifacts) != 5 {
		t.Errorf("Expected 5 artifacts, got %d", len(artifacts))
	}
}

func TestService_ListVersions(t *testing.T) {
	repo := NewMockArtifactRepository()
	svc := NewService(repo)

	// Create an artifact
	artifact, _ := svc.Create(context.Background(), "user-123", CreateArtifactRequest{
		Title:   "Versioned Artifact",
		Type:    "code",
		Content: "v1",
	})

	// Update it a few times
	for i := 2; i <= 4; i++ {
		content := "v" + string(rune('0'+i))
		svc.Update(context.Background(), "user-123", artifact.ID, UpdateArtifactRequest{
			Content: &content,
		})
	}

	// List versions
	versions, err := svc.ListVersions(context.Background(), "user-123", artifact.ID)
	if err != nil {
		t.Fatalf("Service.ListVersions() error = %v", err)
	}

	// Should have 3 versions (updates create versions, not the initial create)
	if len(versions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(versions))
	}
}
