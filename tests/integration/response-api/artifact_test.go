package responseapi_test

import (
	"net/http"
	"testing"
)

// TestArtifactAPI_GetArtifact tests GET /v1/artifacts/:artifact_id
func TestArtifactAPI_GetArtifact(t *testing.T) {
	skipIfNoAPI(t)

	t.Run("returns 404 for non-existent artifact", func(t *testing.T) {
		resp, body := makeRequest(t, http.MethodGet, "/v1/artifacts/non-existent-artifact", nil)
		assertStatus(t, resp, http.StatusNotFound, body)
	})
}

// TestArtifactAPI_GetArtifactsByResponse tests GET /v1/responses/:response_id/artifacts
func TestArtifactAPI_GetArtifactsByResponse(t *testing.T) {
	skipIfNoAPI(t)

	t.Run("returns empty array for response with no artifacts", func(t *testing.T) {
		resp, body := makeRequest(t, http.MethodGet, "/v1/responses/non-existent-response/artifacts", nil)
		// Should return 200 with empty array, or 404 if response doesn't exist
		if resp.StatusCode == http.StatusOK {
			result := parseJSON(t, body)
			artifacts := getArray(t, result, "data")
			if len(artifacts) != 0 {
				t.Errorf("Expected empty artifacts array, got %d items", len(artifacts))
			}
		} else {
			assertStatus(t, resp, http.StatusNotFound, body)
		}
	})
}

// TestArtifactAPI_GetLatestArtifact tests GET /v1/responses/:response_id/artifacts/latest
func TestArtifactAPI_GetLatestArtifact(t *testing.T) {
	skipIfNoAPI(t)

	t.Run("returns 404 for response with no artifacts", func(t *testing.T) {
		resp, body := makeRequest(t, http.MethodGet, "/v1/responses/non-existent-response/artifacts/latest", nil)
		assertStatus(t, resp, http.StatusNotFound, body)
	})
}

// TestArtifactAPI_GetVersions tests GET /v1/artifacts/:artifact_id/versions
func TestArtifactAPI_GetVersions(t *testing.T) {
	skipIfNoAPI(t)

	t.Run("returns 404 for non-existent artifact", func(t *testing.T) {
		resp, body := makeRequest(t, http.MethodGet, "/v1/artifacts/non-existent-artifact/versions", nil)
		assertStatus(t, resp, http.StatusNotFound, body)
	})
}

// TestArtifactAPI_Download tests GET /v1/artifacts/:artifact_id/download
func TestArtifactAPI_Download(t *testing.T) {
	skipIfNoAPI(t)

	t.Run("returns 404 for non-existent artifact", func(t *testing.T) {
		resp, body := makeRequest(t, http.MethodGet, "/v1/artifacts/non-existent-artifact/download", nil)
		assertStatus(t, resp, http.StatusNotFound, body)
	})
}

// TestArtifactAPI_Delete tests DELETE /v1/artifacts/:artifact_id
func TestArtifactAPI_Delete(t *testing.T) {
	skipIfNoAPI(t)

	t.Run("returns 404 for non-existent artifact", func(t *testing.T) {
		resp, body := makeRequest(t, http.MethodDelete, "/v1/artifacts/non-existent-artifact", nil)
		assertStatus(t, resp, http.StatusNotFound, body)
	})
}

/*
Note: Full integration tests require a running Response API with database.
The tests below are placeholders for when a test fixture system is available.

// TestArtifactAPI_FullWorkflow tests the complete artifact lifecycle
func TestArtifactAPI_FullWorkflow(t *testing.T) {
	skipIfNoAPI(t)

	// 1. Create a response that generates an artifact
	// 2. Get artifacts by response ID
	// 3. Get latest artifact
	// 4. Get specific artifact by ID
	// 5. Download artifact content
	// 6. Create a new version
	// 7. Get all versions
	// 8. Delete artifact
}

// TestArtifactAPI_Versioning tests artifact version management
func TestArtifactAPI_Versioning(t *testing.T) {
	skipIfNoAPI(t)

	// 1. Create initial artifact (version 1)
	// 2. Update artifact (creates version 2)
	// 3. Update artifact again (creates version 3)
	// 4. Get all versions - should return 3
	// 5. Get latest - should return version 3
	// 6. Delete - should delete all versions
}

// TestArtifactAPI_ContentTypes tests different artifact content types
func TestArtifactAPI_ContentTypes(t *testing.T) {
	skipIfNoAPI(t)

	contentTypes := []string{"slides", "document", "image", "code", "data"}

	for _, contentType := range contentTypes {
		t.Run(contentType, func(t *testing.T) {
			// Create artifact with content type
			// Verify content type is preserved
			// Download and verify MIME type
		})
	}
}
*/
