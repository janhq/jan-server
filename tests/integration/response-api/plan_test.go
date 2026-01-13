package responseapi_test

import (
	"net/http"
	"testing"
)

// TestPlanAPI_GetPlan tests GET /v1/responses/:response_id/plan
func TestPlanAPI_GetPlan(t *testing.T) {
	skipIfNoAPI(t)

	t.Run("returns 404 for non-existent response", func(t *testing.T) {
		resp, body := makeRequest(t, http.MethodGet, "/v1/responses/non-existent-response/plan", nil)
		assertStatus(t, resp, http.StatusNotFound, body)
	})
}

// TestPlanAPI_GetPlanProgress tests GET /v1/responses/:response_id/plan/progress
func TestPlanAPI_GetPlanProgress(t *testing.T) {
	skipIfNoAPI(t)

	t.Run("returns 404 for non-existent response", func(t *testing.T) {
		resp, body := makeRequest(t, http.MethodGet, "/v1/responses/non-existent-response/plan/progress", nil)
		assertStatus(t, resp, http.StatusNotFound, body)
	})
}

// TestPlanAPI_GetPlanDetails tests GET /v1/responses/:response_id/plan/details
func TestPlanAPI_GetPlanDetails(t *testing.T) {
	skipIfNoAPI(t)

	t.Run("returns 404 for non-existent response", func(t *testing.T) {
		resp, body := makeRequest(t, http.MethodGet, "/v1/responses/non-existent-response/plan/details", nil)
		assertStatus(t, resp, http.StatusNotFound, body)
	})
}

// TestPlanAPI_CancelPlan tests POST /v1/responses/:response_id/plan/cancel
func TestPlanAPI_CancelPlan(t *testing.T) {
	skipIfNoAPI(t)

	t.Run("returns 404 for non-existent response", func(t *testing.T) {
		body := map[string]interface{}{
			"reason": "Test cancellation",
		}
		resp, respBody := makeRequest(t, http.MethodPost, "/v1/responses/non-existent-response/plan/cancel", body)
		assertStatus(t, resp, http.StatusNotFound, respBody)
	})
}

// TestPlanAPI_SubmitUserInput tests POST /v1/responses/:response_id/plan/input
func TestPlanAPI_SubmitUserInput(t *testing.T) {
	skipIfNoAPI(t)

	t.Run("returns 404 for non-existent response", func(t *testing.T) {
		body := map[string]interface{}{
			"input": map[string]interface{}{
				"answer": "yes",
			},
		}
		resp, respBody := makeRequest(t, http.MethodPost, "/v1/responses/non-existent-response/plan/input", body)
		assertStatus(t, resp, http.StatusNotFound, respBody)
	})
}

// TestPlanAPI_ListTasks tests GET /v1/responses/:response_id/plan/tasks
func TestPlanAPI_ListTasks(t *testing.T) {
	skipIfNoAPI(t)

	t.Run("returns 404 for non-existent response", func(t *testing.T) {
		resp, body := makeRequest(t, http.MethodGet, "/v1/responses/non-existent-response/plan/tasks", nil)
		assertStatus(t, resp, http.StatusNotFound, body)
	})
}

/*
Note: Full integration tests require a running Response API with database.
The tests below are placeholders for when a test fixture system is available.

// TestPlanAPI_FullWorkflow tests the complete plan lifecycle
func TestPlanAPI_FullWorkflow(t *testing.T) {
	skipIfNoAPI(t)

	// 1. Create a response (requires response creation endpoint)
	// 2. Get the plan associated with the response
	// 3. Get plan progress
	// 4. Get plan details with tasks and steps
	// 5. Cancel the plan
	// 6. Verify the plan is cancelled
}

// TestPlanAPI_UserInputFlow tests the user input submission flow
func TestPlanAPI_UserInputFlow(t *testing.T) {
	skipIfNoAPI(t)

	// 1. Create a response that triggers a plan requiring user input
	// 2. Wait for plan to reach "wait_for_user" status
	// 3. Submit user input
	// 4. Verify plan continues execution
}
*/
