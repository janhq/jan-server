package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/interfaces/httpserver/handlers"
)

// MockPlanService is a mock implementation of plan.Service for testing.
// Only includes the methods actually used by the handlers.
type MockPlanService struct {
	// Plan lifecycle
	CreateFunc           func(ctx context.Context, params plan.CreateParams) (*plan.Plan, error)
	GetByIDFunc          func(ctx context.Context, id string) (*plan.Plan, error)
	GetByResponseIDFunc  func(ctx context.Context, responseID string) (*plan.Plan, error)
	UpdateStatusFunc     func(ctx context.Context, id string, newStatus status.Status, errorMsg *string) error
	UpdateProgressFunc   func(ctx context.Context, id string, completedSteps int) error
	SetUserSelectionFunc func(ctx context.Context, id string, selection string) error
	SetFinalArtifactFunc func(ctx context.Context, id string, artifactID string) error
	CancelFunc           func(ctx context.Context, id, reason string) error
	DeleteFunc           func(ctx context.Context, id string) error

	// Task operations
	CreateTaskFunc    func(ctx context.Context, planID string, params plan.CreateTaskParams) (*plan.Task, error)
	StartNextTaskFunc func(ctx context.Context, planID string) (*plan.Task, error)
	CompleteTaskFunc  func(ctx context.Context, taskID string) error
	FailTaskFunc      func(ctx context.Context, taskID, errorMsg string) error

	// Step operations
	CreateStepFunc   func(ctx context.Context, taskID string, params plan.CreateStepParams) (*plan.Step, error)
	StartStepFunc    func(ctx context.Context, stepID string) error
	CompleteStepFunc func(ctx context.Context, stepID string, output []byte) error
	FailStepFunc     func(ctx context.Context, stepID, errorMsg string, severity status.ErrorSeverity) error
	RetryStepFunc    func(ctx context.Context, stepID string) (*plan.Step, error)
	SkipStepFunc     func(ctx context.Context, stepID, reason string) error

	// Detail operations
	AddStepDetailFunc func(ctx context.Context, stepID string, detail *plan.StepDetail) error

	// Query operations
	GetProgressFunc        func(ctx context.Context, planID string) (*plan.PlanProgress, error)
	GetPlanWithDetailsFunc func(ctx context.Context, planID string) (*plan.Plan, error)
	ListFunc               func(ctx context.Context, filter *plan.Filter) ([]*plan.Plan, int64, error)
}

// Plan lifecycle methods
func (m *MockPlanService) Create(ctx context.Context, params plan.CreateParams) (*plan.Plan, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockPlanService) GetByID(ctx context.Context, id string) (*plan.Plan, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockPlanService) GetByResponseID(ctx context.Context, responseID string) (*plan.Plan, error) {
	if m.GetByResponseIDFunc != nil {
		return m.GetByResponseIDFunc(ctx, responseID)
	}
	return nil, nil
}

func (m *MockPlanService) UpdateStatus(ctx context.Context, id string, newStatus status.Status, errorMsg *string) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, id, newStatus, errorMsg)
	}
	return nil
}

func (m *MockPlanService) UpdateProgress(ctx context.Context, id string, completedSteps int) error {
	if m.UpdateProgressFunc != nil {
		return m.UpdateProgressFunc(ctx, id, completedSteps)
	}
	return nil
}

func (m *MockPlanService) SetUserSelection(ctx context.Context, id string, selection string) error {
	if m.SetUserSelectionFunc != nil {
		return m.SetUserSelectionFunc(ctx, id, selection)
	}
	return nil
}

func (m *MockPlanService) SetFinalArtifact(ctx context.Context, id string, artifactID string) error {
	if m.SetFinalArtifactFunc != nil {
		return m.SetFinalArtifactFunc(ctx, id, artifactID)
	}
	return nil
}

func (m *MockPlanService) Cancel(ctx context.Context, id, reason string) error {
	if m.CancelFunc != nil {
		return m.CancelFunc(ctx, id, reason)
	}
	return nil
}

func (m *MockPlanService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

// Task operations
func (m *MockPlanService) CreateTask(ctx context.Context, planID string, params plan.CreateTaskParams) (*plan.Task, error) {
	if m.CreateTaskFunc != nil {
		return m.CreateTaskFunc(ctx, planID, params)
	}
	return nil, nil
}

func (m *MockPlanService) StartNextTask(ctx context.Context, planID string) (*plan.Task, error) {
	if m.StartNextTaskFunc != nil {
		return m.StartNextTaskFunc(ctx, planID)
	}
	return nil, nil
}

func (m *MockPlanService) CompleteTask(ctx context.Context, taskID string) error {
	if m.CompleteTaskFunc != nil {
		return m.CompleteTaskFunc(ctx, taskID)
	}
	return nil
}

func (m *MockPlanService) FailTask(ctx context.Context, taskID, errorMsg string) error {
	if m.FailTaskFunc != nil {
		return m.FailTaskFunc(ctx, taskID, errorMsg)
	}
	return nil
}

// Step operations
func (m *MockPlanService) CreateStep(ctx context.Context, taskID string, params plan.CreateStepParams) (*plan.Step, error) {
	if m.CreateStepFunc != nil {
		return m.CreateStepFunc(ctx, taskID, params)
	}
	return nil, nil
}

func (m *MockPlanService) StartStep(ctx context.Context, stepID string) error {
	if m.StartStepFunc != nil {
		return m.StartStepFunc(ctx, stepID)
	}
	return nil
}

func (m *MockPlanService) CompleteStep(ctx context.Context, stepID string, output []byte) error {
	if m.CompleteStepFunc != nil {
		return m.CompleteStepFunc(ctx, stepID, output)
	}
	return nil
}

func (m *MockPlanService) FailStep(ctx context.Context, stepID, errorMsg string, severity status.ErrorSeverity) error {
	if m.FailStepFunc != nil {
		return m.FailStepFunc(ctx, stepID, errorMsg, severity)
	}
	return nil
}

func (m *MockPlanService) RetryStep(ctx context.Context, stepID string) (*plan.Step, error) {
	if m.RetryStepFunc != nil {
		return m.RetryStepFunc(ctx, stepID)
	}
	return nil, nil
}

func (m *MockPlanService) SkipStep(ctx context.Context, stepID, reason string) error {
	if m.SkipStepFunc != nil {
		return m.SkipStepFunc(ctx, stepID, reason)
	}
	return nil
}

// Detail operations
func (m *MockPlanService) AddStepDetail(ctx context.Context, stepID string, detail *plan.StepDetail) error {
	if m.AddStepDetailFunc != nil {
		return m.AddStepDetailFunc(ctx, stepID, detail)
	}
	return nil
}

// Query operations
func (m *MockPlanService) GetProgress(ctx context.Context, planID string) (*plan.PlanProgress, error) {
	if m.GetProgressFunc != nil {
		return m.GetProgressFunc(ctx, planID)
	}
	return nil, nil
}

func (m *MockPlanService) GetPlanWithDetails(ctx context.Context, planID string) (*plan.Plan, error) {
	if m.GetPlanWithDetailsFunc != nil {
		return m.GetPlanWithDetailsFunc(ctx, planID)
	}
	return nil, nil
}

func (m *MockPlanService) List(ctx context.Context, filter *plan.Filter) ([]*plan.Plan, int64, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, filter)
	}
	return nil, 0, nil
}

func setupPlanTestRouter(handler *handlers.PlanHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/v1/responses/:response_id/plan")
	{
		v1.GET("", handler.Get)
		v1.GET("/details", handler.GetWithDetails)
		v1.GET("/progress", handler.GetProgress)
		v1.POST("/cancel", handler.Cancel)
		v1.POST("/input", handler.SubmitUserInput)
		v1.GET("/tasks", handler.ListTasks)
	}
	return r
}

func TestPlanHandler_Get(t *testing.T) {
	mockService := &MockPlanService{
		GetByResponseIDFunc: func(ctx context.Context, responseID string) (*plan.Plan, error) {
			return &plan.Plan{
				ID:         "plan-123",
				ResponseID: responseID,
				Status:     status.StatusInProgress,
				AgentType:  plan.AgentTypeDeepResearch,
				CreatedAt:  time.Now(),
			}, nil
		},
	}

	handler := handlers.NewPlanHandler(mockService, zerolog.Nop())
	router := setupPlanTestRouter(handler)

	req, _ := http.NewRequest("GET", "/v1/responses/resp-123/plan", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["id"] != "plan-123" {
		t.Errorf("Expected plan id 'plan-123', got %v", response["id"])
	}
}

func TestPlanHandler_GetProgress(t *testing.T) {
	mockService := &MockPlanService{
		GetByResponseIDFunc: func(ctx context.Context, responseID string) (*plan.Plan, error) {
			return &plan.Plan{
				ID:         "plan-123",
				ResponseID: responseID,
			}, nil
		},
		GetProgressFunc: func(ctx context.Context, planID string) (*plan.PlanProgress, error) {
			return &plan.PlanProgress{
				PlanID:         planID,
				Status:         status.StatusInProgress,
				Progress:       40.0,
				EstimatedSteps: 10,
				CompletedSteps: 4,
				FailedSteps:    0,
			}, nil
		},
	}

	handler := handlers.NewPlanHandler(mockService, zerolog.Nop())
	router := setupPlanTestRouter(handler)

	req, _ := http.NewRequest("GET", "/v1/responses/resp-123/plan/progress", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["progress"] != 40.0 {
		t.Errorf("Expected progress 40.0, got %v", response["progress"])
	}
}

func TestPlanHandler_Cancel(t *testing.T) {
	cancelCalled := false
	mockService := &MockPlanService{
		GetByResponseIDFunc: func(ctx context.Context, responseID string) (*plan.Plan, error) {
			return &plan.Plan{
				ID:         "plan-123",
				ResponseID: responseID,
				Status:     status.StatusInProgress,
			}, nil
		},
		CancelFunc: func(ctx context.Context, planID, reason string) error {
			cancelCalled = true
			return nil
		},
		GetByIDFunc: func(ctx context.Context, id string) (*plan.Plan, error) {
			return &plan.Plan{
				ID:     id,
				Status: status.StatusCancelled,
			}, nil
		},
	}

	handler := handlers.NewPlanHandler(mockService, zerolog.Nop())
	router := setupPlanTestRouter(handler)

	body := bytes.NewBufferString(`{"reason": "User cancelled"}`)
	req, _ := http.NewRequest("POST", "/v1/responses/resp-123/plan/cancel", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !cancelCalled {
		t.Error("Expected Cancel to be called")
	}
}

func TestPlanHandler_SubmitUserInput(t *testing.T) {
	selectionReceived := false
	mockService := &MockPlanService{
		GetByResponseIDFunc: func(ctx context.Context, responseID string) (*plan.Plan, error) {
			return &plan.Plan{
				ID:         "plan-123",
				ResponseID: responseID,
				Status:     status.StatusWaitForUser,
			}, nil
		},
		SetUserSelectionFunc: func(ctx context.Context, id, selection string) error {
			selectionReceived = true
			if selection != "option_a" {
				t.Errorf("Expected selection 'option_a', got %v", selection)
			}
			return nil
		},
		UpdateStatusFunc: func(ctx context.Context, id string, newStatus status.Status, errorMsg *string) error {
			return nil
		},
		GetByIDFunc: func(ctx context.Context, id string) (*plan.Plan, error) {
			return &plan.Plan{
				ID:     id,
				Status: status.StatusInProgress,
			}, nil
		},
	}

	handler := handlers.NewPlanHandler(mockService, zerolog.Nop())
	router := setupPlanTestRouter(handler)

	body := bytes.NewBufferString(`{"selection": "option_a"}`)
	req, _ := http.NewRequest("POST", "/v1/responses/resp-123/plan/input", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !selectionReceived {
		t.Error("Expected SetUserSelection to be called")
	}
}

func TestPlanHandler_ListTasks(t *testing.T) {
	mockService := &MockPlanService{
		GetByResponseIDFunc: func(ctx context.Context, responseID string) (*plan.Plan, error) {
			return &plan.Plan{
				ID:         "plan-123",
				ResponseID: responseID,
			}, nil
		},
		GetPlanWithDetailsFunc: func(ctx context.Context, planID string) (*plan.Plan, error) {
			return &plan.Plan{
				ID:         planID,
				ResponseID: "resp-123",
				Tasks: []plan.Task{
					{
						ID:       "task-1",
						PlanID:   planID,
						Title:    "Research Task",
						TaskType: plan.TaskTypeResearch,
						Status:   status.StatusCompleted,
						Sequence: 1,
					},
					{
						ID:       "task-2",
						PlanID:   planID,
						Title:    "Generate Task",
						TaskType: plan.TaskTypeGeneration,
						Status:   status.StatusPending,
						Sequence: 2,
					},
				},
			}, nil
		},
	}

	handler := handlers.NewPlanHandler(mockService, zerolog.Nop())
	router := setupPlanTestRouter(handler)

	req, _ := http.NewRequest("GET", "/v1/responses/resp-123/plan/tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var tasks []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
}
